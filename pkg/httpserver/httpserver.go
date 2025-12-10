// Package httpserver implements HTTP monitoring and optional AuthZen API for SPOCP.
//
// This package provides an HTTP server that:
//  1. Always provides health and monitoring endpoints (/health, /ready, /stats, /metrics)
//  2. Optionally provides the AuthZen Authorization API 1.0 endpoint (/access/v1/evaluation)
//
// The HTTP server serves as a unified monitoring interface for both TCP and HTTP/AuthZen
// protocols, while the AuthZen API endpoint can be enabled independently via configuration.
//
// The server can operate in two modes:
//
//  1. Standalone mode: Creates its own SPOCP engine and loads rules from a directory
//  2. Shared mode: Uses an engine shared with the TCP server for dual-protocol operation
//
// Example standalone usage with AuthZen enabled:
//
//	config := &httpserver.Config{
//	    Address:       ":8000",
//	    EnableAuthZen: true,
//	    RulesDir:      "/etc/spocp/rules",
//	    LogLevel:      server.LogLevelInfo,
//	}
//	srv, err := httpserver.NewHTTPServer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	srv.Start()
//
// Example shared mode with TCP server (monitoring only, no AuthZen):
//
//	tcpSrv, _ := server.NewServer(&server.Config{...})
//	config := &httpserver.Config{
//	    Address:       ":8000",
//	    EnableAuthZen: false,  // Only monitoring endpoints
//	    Engine:        tcpSrv.GetEngine(),
//	    EngineMutex:   tcpSrv.GetEngineMutex(),
//	    LogLevel:      server.LogLevelInfo,
//	}
//	httpSrv, _ := httpserver.NewHTTPServer(config)
//	httpSrv.Start()
//
// The server exposes the following endpoints:
//
//	GET  /health                - Health check endpoint (always enabled)
//	GET  /ready                 - Readiness check (always enabled, checks if rules are loaded)
//	GET  /stats                 - JSON statistics (always enabled, requests, rules, indexing)
//	GET  /metrics               - Prometheus-style metrics (always enabled)
//	POST /access/v1/evaluation  - AuthZen API (optional, enabled via EnableAuthZen flag)
//
// AuthZen API Request format (JSON):
//
//	{
//	  "subject": {"type": "user", "id": "alice"},
//	  "resource": {"type": "document", "id": "123"},
//	  "action": {"name": "can_read"}
//	}
//
// AuthZen API Response format (JSON):
//
//	{
//	  "decision": true
//	}
package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/authzen"
	"github.com/sirosfoundation/go-spocp/pkg/persist"
	"github.com/sirosfoundation/go-spocp/pkg/server"
)

// HTTPServer provides an HTTP/AuthZen interface to SPOCP engine.
type HTTPServer struct {
	server   *http.Server
	engine   *spocp.Engine
	mu       *sync.RWMutex // Pointer to allow sharing mutex with other components
	logger   *log.Logger
	logLevel server.LogLevel
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Metrics
	metrics struct {
		requestsTotal atomic.Int64
		requestsOK    atomic.Int64
		requestsDeny  atomic.Int64
		errors        atomic.Int64
	}
}

// Config contains HTTP server configuration.
type Config struct {
	// Address to listen on (e.g., ":8080")
	Address string

	// EnableAuthZen enables the AuthZen API endpoint (default: false, always provides health/stats/metrics)
	EnableAuthZen bool

	// RulesDir for loading rules (required if Engine not provided)
	RulesDir string

	// Engine is the SPOCP engine (optional - will be created if not provided)
	Engine *spocp.Engine

	// EngineMutex protects engine access (optional - will be created if not provided)
	EngineMutex *sync.RWMutex

	// Logger (optional)
	Logger *log.Logger

	// LogLevel controls verbosity
	LogLevel server.LogLevel

	// ReloadInterval for automatic rule reloading (0 to disable)
	ReloadInterval time.Duration

	// PidFile path (optional)
	PidFile string
}

// NewHTTPServer creates a new HTTP/AuthZen server.
//
// The server can operate in two modes:
//  1. Standalone mode: Provide RulesDir in config, server creates and manages its own engine
//  2. Shared mode: Provide Engine and EngineMutex in config, server shares engine with other components
//
// Required config fields:
//   - Address: HTTP listen address (e.g., ":8000")
//   - Either RulesDir (standalone) or Engine + EngineMutex (shared)
//
// Optional config fields:
//   - Logger: Custom logger (defaults to standard logger with [SPOCP-HTTP] prefix)
//   - LogLevel: Controls verbosity (0=silent, 1=error, 2=warn, 3=info, 4=debug)
//
// Example (standalone):
//
//	srv, err := NewHTTPServer(&Config{
//	    Address: ":8000",
//	    RulesDir: "/etc/spocp/rules",
//	    LogLevel: 3,
//	})
//
// Example (shared with TCP server):
//
//	tcpServer := server.NewServer(...)
//	httpSrv, err := NewHTTPServer(&Config{
//	    Address: ":8000",
//	    Engine: tcpServer.GetEngine(),
//	    EngineMutex: tcpServer.GetEngineMutex(),
//	})
func NewHTTPServer(config *Config) (*HTTPServer, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Create engine if not provided
	if config.Engine == nil {
		if config.RulesDir == "" {
			return nil, fmt.Errorf("either engine or rules directory is required")
		}
		config.Engine = spocp.NewEngine()

		// Load rules from directory
		if err := loadRulesFromDir(config.Engine, config.RulesDir); err != nil {
			return nil, fmt.Errorf("failed to load rules: %w", err)
		}
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(log.Writer(), "[SPOCP-HTTP] ", log.LstdFlags)
	}

	ctx, cancel := context.WithCancel(context.Background())

	hs := &HTTPServer{
		engine:   config.Engine,
		logger:   logger,
		logLevel: config.LogLevel,
		ctx:      ctx,
		cancel:   cancel,
	}

	// If engine mutex provided, use it; otherwise create own mutex
	if config.EngineMutex != nil {
		hs.mu = config.EngineMutex
	} else {
		hs.mu = &sync.RWMutex{}
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// AuthZen API endpoint (optional)
	if config.EnableAuthZen {
		mux.HandleFunc("/access/v1/evaluation", hs.handleEvaluation)
	}

	// Health and monitoring endpoints (always enabled)
	mux.HandleFunc("/health", hs.handleHealth)
	mux.HandleFunc("/ready", hs.handleReady)
	mux.HandleFunc("/stats", hs.handleStats)
	mux.HandleFunc("/metrics", hs.handleMetrics)

	hs.server = &http.Server{
		Addr:         config.Address,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return hs, nil
}

// Start begins accepting HTTP requests in a background goroutine.
//
// This method returns immediately after launching the HTTP server.
// The server will continue running until Close() is called or an
// unrecoverable error occurs.
//
// The server handles POST requests to /access/v1/evaluation according
// to the AuthZen Authorization API 1.0 specification.
//
// Returns an error only if the server cannot be started (e.g., port
// already in use). Runtime errors are logged but don't propagate to
// the caller.
func (hs *HTTPServer) Start() error {
	hs.logInfo("AuthZen HTTP server listening on %s", hs.server.Addr)

	hs.wg.Add(1)
	go func() {
		defer hs.wg.Done()
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hs.logError("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Close gracefully shuts down the HTTP server.
//
// This method:
//  1. Stops accepting new connections
//  2. Waits up to 5 seconds for existing requests to complete
//  3. Forcefully closes remaining connections after timeout
//  4. Waits for background goroutines to exit
//
// Returns an error if shutdown fails or times out.
func (hs *HTTPServer) Close() error {
	hs.logInfo("Shutting down HTTP server...")
	hs.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hs.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown error: %w", err)
	}

	hs.wg.Wait()
	hs.logInfo("HTTP server stopped")
	return nil
}

// handleEvaluation handles AuthZen access evaluation requests.
//
// Endpoint: POST /access/v1/evaluation
//
// Request format (JSON):
//
//	{
//	  "subject": {"type": "user", "id": "alice@acmecorp.com"},
//	  "resource": {"type": "account", "id": "123"},
//	  "action": {"name": "can_read"}
//	}
//
// Response format (JSON):
//
//	{
//	  "decision": true,
//	  "context": {"id": "<request-id>"}
//	}
//
// The handler:
//  1. Validates HTTP method (must be POST)
//  2. Parses JSON request body into EvaluationRequest
//  3. Converts AuthZen request to SPOCP S-expression
//  4. Queries the SPOCP engine (with read lock for thread safety)
//  5. Returns decision as JSON response
//
// Supports X-Request-ID header for distributed tracing.
// Updates internal metrics for monitoring.
func (hs *HTTPServer) handleEvaluation(w http.ResponseWriter, r *http.Request) {
	// Only allow POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Track request ID if provided
	requestID := r.Header.Get("X-Request-ID")

	// Parse request
	var req authzen.EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		hs.metrics.errors.Add(1)
		hs.logError("Failed to decode request: %v", err)
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
		return
	}

	hs.logDebug("AuthZen request: subject=%s/%s resource=%s/%s action=%s",
		req.Subject.Type, req.Subject.ID,
		req.Resource.Type, req.Resource.ID,
		req.Action.Name)

	// Convert to S-expression
	query, err := req.ToSExpression()
	if err != nil {
		hs.metrics.errors.Add(1)
		hs.logError("Failed to convert to S-expression: %v", err)
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
		return
	}

	hs.logDebug("SPOCP query: %s", query.String())

	// Evaluate query against engine
	hs.mu.RLock()
	decision := hs.engine.QueryElement(query)
	hs.mu.RUnlock()

	// Update metrics
	hs.metrics.requestsTotal.Add(1)
	if decision {
		hs.metrics.requestsOK.Add(1)
	} else {
		hs.metrics.requestsDeny.Add(1)
	}

	// Build response
	resp := authzen.EvaluationResponse{
		Decision: decision,
	}

	hs.logDebug("AuthZen decision: %t", decision)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		hs.logError("Failed to encode response: %v", err)
	}
}

// handleHealth returns the health status of the HTTP server.
func (hs *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// handleReady returns readiness status based on whether rules are loaded.
func (hs *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if we have any rules loaded
	hs.mu.RLock()
	ruleCount := hs.engine.RuleCount()
	hs.mu.RUnlock()

	if ruleCount == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not ready","reason":"no rules loaded"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ready"}`)
}

// handleMetrics returns Prometheus-style metrics.
func (hs *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintf(w, "# HELP spocp_http_requests_total Total number of AuthZen evaluation requests\n")
	fmt.Fprintf(w, "# TYPE spocp_http_requests_total counter\n")
	fmt.Fprintf(w, "spocp_http_requests_total %d\n", hs.metrics.requestsTotal.Load())

	fmt.Fprintf(w, "# HELP spocp_http_requests_ok Total number of allowed decisions\n")
	fmt.Fprintf(w, "# TYPE spocp_http_requests_ok counter\n")
	fmt.Fprintf(w, "spocp_http_requests_ok %d\n", hs.metrics.requestsOK.Load())

	fmt.Fprintf(w, "# HELP spocp_http_requests_deny Total number of denied decisions\n")
	fmt.Fprintf(w, "# TYPE spocp_http_requests_deny counter\n")
	fmt.Fprintf(w, "spocp_http_requests_deny %d\n", hs.metrics.requestsDeny.Load())

	fmt.Fprintf(w, "# HELP spocp_http_errors Total number of request errors\n")
	fmt.Fprintf(w, "# TYPE spocp_http_errors counter\n")
	fmt.Fprintf(w, "spocp_http_errors %d\n", hs.metrics.errors.Load())

	// Add engine statistics
	hs.mu.RLock()
	ruleCount := hs.engine.RuleCount()
	indexStats := hs.engine.GetIndexStats()
	hs.mu.RUnlock()

	fmt.Fprintf(w, "# HELP spocp_rules_loaded Current number of rules loaded\n")
	fmt.Fprintf(w, "# TYPE spocp_rules_loaded gauge\n")
	fmt.Fprintf(w, "spocp_rules_loaded %d\n", ruleCount)

	if indexingEnabled, ok := indexStats["indexing_enabled"].(bool); ok && indexingEnabled {
		if tagCount, ok := indexStats["tag_count"].(int); ok {
			fmt.Fprintf(w, "# HELP spocp_index_tags Total number of indexed tags\n")
			fmt.Fprintf(w, "# TYPE spocp_index_tags gauge\n")
			fmt.Fprintf(w, "spocp_index_tags %d\n", tagCount)
		}
		if rulesByTag, ok := indexStats["rules_by_tag"].(int); ok {
			fmt.Fprintf(w, "# HELP spocp_index_rules_by_tag Number of rules indexed by tag\n")
			fmt.Fprintf(w, "# TYPE spocp_index_rules_by_tag gauge\n")
			fmt.Fprintf(w, "spocp_index_rules_by_tag %d\n", rulesByTag)
		}
	}
}

// handleStats returns JSON statistics about the HTTP server and engine.
func (hs *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hs.mu.RLock()
	ruleCount := hs.engine.RuleCount()
	indexStats := hs.engine.GetIndexStats()
	hs.mu.RUnlock()

	totalRules := int64(ruleCount)
	rulesByTag := int64(0)
	indexingEnabled := false
	tagCount := int64(0)

	if v, ok := indexStats["rules_by_tag"].(int); ok {
		rulesByTag = int64(v)
	}
	if v, ok := indexStats["indexing_enabled"].(bool); ok {
		indexingEnabled = v
	}
	if v, ok := indexStats["tag_count"].(int); ok {
		tagCount = int64(v)
	}

	fmt.Fprintf(w, `{
  "requests": {
    "total": %d,
    "ok": %d,
    "denied": %d,
    "errors": %d
  },
  "rules": {
    "loaded": %d,
    "total": %d,
    "by_tag": %d
  },
  "indexing": {
    "enabled": %t,
    "tag_count": %d
  }
}`,
		hs.metrics.requestsTotal.Load(),
		hs.metrics.requestsOK.Load(),
		hs.metrics.requestsDeny.Load(),
		hs.metrics.errors.Load(),
		totalRules,
		totalRules,
		rulesByTag,
		indexingEnabled,
		tagCount,
	)
}

// Logging helpers
func (hs *HTTPServer) logDebug(format string, v ...interface{}) {
	if hs.logLevel >= server.LogLevelDebug {
		hs.logger.Printf("[DEBUG] "+format, v...)
	}
}

func (hs *HTTPServer) logInfo(format string, v ...interface{}) {
	if hs.logLevel >= server.LogLevelInfo {
		hs.logger.Printf("[INFO] "+format, v...)
	}
}

func (hs *HTTPServer) logWarn(format string, v ...interface{}) {
	if hs.logLevel >= server.LogLevelWarn {
		hs.logger.Printf("[WARN] "+format, v...)
	}
}

func (hs *HTTPServer) logError(format string, v ...interface{}) {
	if hs.logLevel >= server.LogLevelError {
		hs.logger.Printf("[ERROR] "+format, v...)
	}
}

// GetMetrics returns current metrics.
func (hs *HTTPServer) GetMetrics() map[string]int64 {
	return map[string]int64{
		"requests_total": hs.metrics.requestsTotal.Load(),
		"requests_ok":    hs.metrics.requestsOK.Load(),
		"requests_deny":  hs.metrics.requestsDeny.Load(),
		"errors":         hs.metrics.errors.Load(),
	}
}

// loadRulesFromDir loads all .spoc files from a directory into the engine.
func loadRulesFromDir(engine *spocp.Engine, dir string) error {
	var ruleCount int

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".spoc") {
			return nil
		}

		rules, err := persist.LoadFileToSlice(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		for _, rule := range rules {
			engine.AddRuleElement(rule)
			ruleCount++
		}

		return nil
	})

	if err != nil {
		return err
	}

	if ruleCount == 0 {
		return fmt.Errorf("no rules loaded from %s", dir)
	}

	return nil
}
