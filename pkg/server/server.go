// Package server implements a SPOCP TCP server with TLS support and dynamic rule reloading.
package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/persist"
	"github.com/sirosfoundation/go-spocp/pkg/protocol"
)

// LogLevel defines the verbosity of logging
type LogLevel int

const (
	LogLevelSilent LogLevel = iota // No logging except errors
	LogLevelError                  // Errors only
	LogLevelWarn                   // Warnings and errors
	LogLevelInfo                   // Informational messages (default)
	LogLevelDebug                  // Verbose debugging
)

// Server represents a SPOCP TCP server
type Server struct {
	listener       net.Listener
	engine         *spocp.Engine
	rulesDir       string
	tlsConfig      *tls.Config
	mu             sync.RWMutex
	reloadMutex    sync.Mutex
	logger         *log.Logger
	logLevel       LogLevel
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	pidFile        string
	healthAddr     string
	healthListener net.Listener

	// Metrics
	metrics struct {
		queriesTotal     atomic.Int64
		queriesOK        atomic.Int64
		queriesDenied    atomic.Int64
		addsTotal        atomic.Int64
		reloadsTotal     atomic.Int64
		reloadsFailed    atomic.Int64
		connectionsTotal atomic.Int64
		lastReloadTime   atomic.Value // time.Time
		rulesLoaded      atomic.Int64
	}
}

// Config contains server configuration
type Config struct {
	// Address to listen on (e.g., ":6000")
	Address string

	// Directory containing .spoc rule files
	RulesDir string

	// TLS configuration (optional, nil for plain TCP)
	TLSConfig *tls.Config

	// Logger (optional, defaults to discard logger)
	Logger *log.Logger

	// LogLevel controls verbosity (default: LogLevelError)
	LogLevel LogLevel

	// ReloadInterval for automatic rule reloading (0 to disable)
	ReloadInterval time.Duration

	// PidFile path for storing process ID (optional)
	PidFile string

	// HealthAddr for health check endpoint (e.g., ":8080", optional)
	HealthAddr string
}

// NewServer creates a new SPOCP server
func NewServer(config *Config) (*Server, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}
	if config.RulesDir == "" {
		return nil, fmt.Errorf("rules directory is required")
	}

	// Check if rules directory exists
	if _, err := os.Stat(config.RulesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("rules directory does not exist: %s", config.RulesDir)
	}

	// Setup logger
	logger := config.Logger
	if logger == nil {
		logger = log.New(io.Discard, "[SPOCP] ", log.LstdFlags)
	}

	logLevel := config.LogLevel
	if logLevel == 0 {
		logLevel = LogLevelError // Default to minimal logging
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		engine:     spocp.NewEngine(),
		rulesDir:   config.RulesDir,
		tlsConfig:  config.TLSConfig,
		logger:     logger,
		logLevel:   logLevel,
		ctx:        ctx,
		cancel:     cancel,
		pidFile:    config.PidFile,
		healthAddr: config.HealthAddr,
	}

	// Initialize last reload time
	s.metrics.lastReloadTime.Store(time.Now())

	// Write PID file if configured
	if config.PidFile != "" {
		if err := s.writePidFile(); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to write PID file: %w", err)
		}
	}

	// Load initial rules
	if err := s.reloadRules(); err != nil {
		cancel()
		s.removePidFile()
		return nil, fmt.Errorf("failed to load initial rules: %w", err)
	}

	// Create listener
	var err error
	if config.TLSConfig != nil {
		s.listener, err = tls.Listen("tcp", config.Address, config.TLSConfig)
		if err != nil {
			cancel()
			s.removePidFile()
			return nil, fmt.Errorf("failed to create TLS listener: %w", err)
		}
		s.logInfo("TLS server listening on %s", config.Address)
	} else {
		s.listener, err = net.Listen("tcp", config.Address)
		if err != nil {
			cancel()
			s.removePidFile()
			return nil, fmt.Errorf("failed to create listener: %w", err)
		}
		s.logInfo("Server listening on %s (plain TCP)", config.Address)
	}

	// Start health check endpoint if configured
	if config.HealthAddr != "" {
		if err := s.startHealthCheck(); err != nil {
			cancel()
			s.listener.Close()
			s.removePidFile()
			return nil, fmt.Errorf("failed to start health check: %w", err)
		}
	}

	// Start automatic reloading if configured
	if config.ReloadInterval > 0 {
		s.wg.Add(1)
		go s.autoReload(config.ReloadInterval)
	}

	return s, nil
}

// Serve starts accepting connections
func (s *Server) Serve() error {
	s.logInfo("Server started, waiting for connections...")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				s.logError("Accept error: %v", err)
				continue
			}
		}

		s.metrics.connectionsTotal.Add(1)
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// Close gracefully shuts down the server
func (s *Server) Close() error {
	s.logInfo("Shutting down server...")
	s.cancel()

	err := s.listener.Close()

	// Close health check if running
	if s.healthListener != nil {
		s.healthListener.Close()
	}

	// Wait for all connections to close
	s.wg.Wait()

	// Remove PID file
	s.removePidFile()

	s.logInfo("Server stopped")
	return err
}

// handleConnection processes a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	s.logDebug("New connection from %s", remoteAddr)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		// Read message
		msg, err := protocol.DecodeMessage(reader)
		if err != nil {
			if err.Error() == "EOF" {
				s.logDebug("Client %s disconnected", remoteAddr)
				return
			}
			s.logError("Error reading from %s: %v", remoteAddr, err)
			s.sendResponse(writer, &protocol.Response{
				Code:    protocol.CodeError,
				Message: "Protocol error",
			})
			return
		}

		s.logDebug("Received from %s: %s %v", remoteAddr, msg.Operation, msg.Arguments)

		// Handle message
		resp := s.handleMessage(msg)

		// Send response
		if err := s.sendResponse(writer, resp); err != nil {
			s.logError("Error sending response to %s: %v", remoteAddr, err)
			return
		}

		// Close connection if LOGOUT
		if msg.Operation == "LOGOUT" {
			s.logDebug("Client %s logged out", remoteAddr)
			return
		}
	}
}

// handleMessage processes a protocol message and returns a response
func (s *Server) handleMessage(msg *protocol.Message) *protocol.Response {
	switch msg.Operation {
	case "QUERY":
		return s.handleQuery(msg)
	case "ADD":
		return s.handleAdd(msg)
	case "LOGOUT":
		return &protocol.Response{Code: protocol.CodeBye, Message: "Bye"}
	case "RELOAD":
		return s.handleReload()
	default:
		return &protocol.Response{
			Code:    protocol.CodeUnknown,
			Message: fmt.Sprintf("Unknown operation: %s", msg.Operation),
		}
	}
}

// handleQuery processes a QUERY operation
func (s *Server) handleQuery(msg *protocol.Message) *protocol.Response {
	s.metrics.queriesTotal.Add(1)

	if len(msg.Arguments) != 1 {
		return &protocol.Response{
			Code:    protocol.CodeError,
			Message: "QUERY requires exactly one argument",
		}
	}

	// Parse query
	query, err := protocol.ParseQuery(msg.Arguments[0])
	if err != nil {
		return &protocol.Response{
			Code:    protocol.CodeError,
			Message: fmt.Sprintf("Invalid query: %v", err),
		}
	}

	// Execute query
	s.mu.RLock()
	result := s.engine.QueryElement(query)
	s.mu.RUnlock()

	if result {
		s.metrics.queriesOK.Add(1)
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	}
	s.metrics.queriesDenied.Add(1)
	return &protocol.Response{Code: protocol.CodeDenied, Message: "Denied"}
}

// handleAdd processes an ADD operation
func (s *Server) handleAdd(msg *protocol.Message) *protocol.Response {
	s.metrics.addsTotal.Add(1)

	if len(msg.Arguments) != 1 {
		return &protocol.Response{
			Code:    protocol.CodeError,
			Message: "ADD requires exactly one argument",
		}
	}

	// Parse rule
	rule, err := protocol.ParseRule(msg.Arguments[0])
	if err != nil {
		return &protocol.Response{
			Code:    protocol.CodeError,
			Message: fmt.Sprintf("Invalid rule: %v", err),
		}
	}

	// Add rule
	s.mu.Lock()
	s.engine.AddRuleElement(rule)
	s.mu.Unlock()

	return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
}

// handleReload processes a RELOAD operation
func (s *Server) handleReload() *protocol.Response {
	s.metrics.reloadsTotal.Add(1)

	if err := s.reloadRules(); err != nil {
		s.metrics.reloadsFailed.Add(1)
		return &protocol.Response{
			Code:    protocol.CodeError,
			Message: fmt.Sprintf("Reload failed: %v", err),
		}
	}
	return &protocol.Response{Code: protocol.CodeOK, Message: "Reloaded"}
}

// sendResponse sends a response to the client
func (s *Server) sendResponse(writer *bufio.Writer, resp *protocol.Response) error {
	encoded := protocol.EncodeResponse(resp)
	if _, err := writer.WriteString(encoded); err != nil {
		return err
	}
	return writer.Flush()
}

// reloadRules reloads all .spoc files from the rules directory
// Uses atomic swap to ensure no downtime
func (s *Server) reloadRules() error {
	s.reloadMutex.Lock()
	defer s.reloadMutex.Unlock()

	s.logDebug("Reloading rules from %s", s.rulesDir)

	// Create new engine
	newEngine := spocp.NewEngine()

	// Find all .spoc files
	var ruleFiles []string
	err := filepath.WalkDir(s.rulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".spoc") {
			ruleFiles = append(ruleFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan rules directory: %w", err)
	}

	if len(ruleFiles) == 0 {
		s.logWarn("No .spoc files found in %s", s.rulesDir)
	}

	// Load each file
	totalRules := 0
	for _, file := range ruleFiles {
		rules, err := persist.LoadFileToSlice(file)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}

		for _, rule := range rules {
			newEngine.AddRuleElement(rule)
			totalRules++
		}
	}

	// Replace engine atomically
	s.mu.Lock()
	s.engine = newEngine
	s.mu.Unlock()

	// Update metrics
	s.metrics.rulesLoaded.Store(int64(totalRules))
	s.metrics.lastReloadTime.Store(time.Now())

	s.logInfo("Loaded %d rules from %d files", totalRules, len(ruleFiles))
	return nil
}

// autoReload periodically reloads rules
func (s *Server) autoReload(interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.reloadRules(); err != nil {
				s.logError("Auto-reload failed: %v", err)
				s.metrics.reloadsFailed.Add(1)
			}
		}
	}
}

// Logging helpers with level filtering

func (s *Server) logDebug(format string, v ...interface{}) {
	if s.logLevel >= LogLevelDebug {
		s.logger.Printf("[DEBUG] "+format, v...)
	}
}

func (s *Server) logInfo(format string, v ...interface{}) {
	if s.logLevel >= LogLevelInfo {
		s.logger.Printf("[INFO] "+format, v...)
	}
}

func (s *Server) logWarn(format string, v ...interface{}) {
	if s.logLevel >= LogLevelWarn {
		s.logger.Printf("[WARN] "+format, v...)
	}
}

func (s *Server) logError(format string, v ...interface{}) {
	if s.logLevel >= LogLevelError {
		s.logger.Printf("[ERROR] "+format, v...)
	}
}

// PID file management

func (s *Server) writePidFile() error {
	if s.pidFile == "" {
		return nil
	}

	pid := os.Getpid()
	content := fmt.Sprintf("%d\n", pid)

	// Write atomically
	tmpFile := s.pidFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp PID file: %w", err)
	}

	if err := os.Rename(tmpFile, s.pidFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename PID file: %w", err)
	}

	s.logDebug("PID file written: %s (PID: %d)", s.pidFile, pid)
	return nil
}

func (s *Server) removePidFile() {
	if s.pidFile == "" {
		return
	}

	if err := os.Remove(s.pidFile); err != nil && !os.IsNotExist(err) {
		s.logWarn("Failed to remove PID file %s: %v", s.pidFile, err)
	} else {
		s.logDebug("PID file removed: %s", s.pidFile)
	}
}

// Health check endpoint

func (s *Server) startHealthCheck() error {
	if s.healthAddr == "" {
		return nil
	}

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// Readiness endpoint
	mux.HandleFunc("/ready", s.handleReady)

	// Metrics endpoint
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Stats endpoint (detailed)
	mux.HandleFunc("/stats", s.handleStats)

	listener, err := net.Listen("tcp", s.healthAddr)
	if err != nil {
		return fmt.Errorf("failed to create health listener: %w", err)
	}

	s.healthListener = listener

	srv := &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logInfo("Health check endpoint listening on %s", s.healthAddr)

		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.logError("Health check server error: %v", err)
		}
	}()

	// Shutdown handler
	go func() {
		<-s.ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check if we have any rules loaded
	if s.metrics.rulesLoaded.Load() == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not ready","reason":"no rules loaded"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ready"}`)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	// Prometheus-style metrics
	fmt.Fprintf(w, "# HELP spocp_queries_total Total number of queries\n")
	fmt.Fprintf(w, "# TYPE spocp_queries_total counter\n")
	fmt.Fprintf(w, "spocp_queries_total %d\n", s.metrics.queriesTotal.Load())

	fmt.Fprintf(w, "# HELP spocp_queries_ok Total number of successful queries\n")
	fmt.Fprintf(w, "# TYPE spocp_queries_ok counter\n")
	fmt.Fprintf(w, "spocp_queries_ok %d\n", s.metrics.queriesOK.Load())

	fmt.Fprintf(w, "# HELP spocp_queries_denied Total number of denied queries\n")
	fmt.Fprintf(w, "# TYPE spocp_queries_denied counter\n")
	fmt.Fprintf(w, "spocp_queries_denied %d\n", s.metrics.queriesDenied.Load())

	fmt.Fprintf(w, "# HELP spocp_adds_total Total number of ADD operations\n")
	fmt.Fprintf(w, "# TYPE spocp_adds_total counter\n")
	fmt.Fprintf(w, "spocp_adds_total %d\n", s.metrics.addsTotal.Load())

	fmt.Fprintf(w, "# HELP spocp_reloads_total Total number of rule reloads\n")
	fmt.Fprintf(w, "# TYPE spocp_reloads_total counter\n")
	fmt.Fprintf(w, "spocp_reloads_total %d\n", s.metrics.reloadsTotal.Load())

	fmt.Fprintf(w, "# HELP spocp_reloads_failed Total number of failed reloads\n")
	fmt.Fprintf(w, "# TYPE spocp_reloads_failed counter\n")
	fmt.Fprintf(w, "spocp_reloads_failed %d\n", s.metrics.reloadsFailed.Load())

	fmt.Fprintf(w, "# HELP spocp_connections_total Total number of connections\n")
	fmt.Fprintf(w, "# TYPE spocp_connections_total counter\n")
	fmt.Fprintf(w, "spocp_connections_total %d\n", s.metrics.connectionsTotal.Load())

	fmt.Fprintf(w, "# HELP spocp_rules_loaded Current number of rules loaded\n")
	fmt.Fprintf(w, "# TYPE spocp_rules_loaded gauge\n")
	fmt.Fprintf(w, "spocp_rules_loaded %d\n", s.metrics.rulesLoaded.Load())

	if lastReload, ok := s.metrics.lastReloadTime.Load().(time.Time); ok {
		fmt.Fprintf(w, "# HELP spocp_last_reload_timestamp_seconds Timestamp of last reload\n")
		fmt.Fprintf(w, "# TYPE spocp_last_reload_timestamp_seconds gauge\n")
		fmt.Fprintf(w, "spocp_last_reload_timestamp_seconds %d\n", lastReload.Unix())
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	lastReload := "never"
	if t, ok := s.metrics.lastReloadTime.Load().(time.Time); ok {
		lastReload = t.Format(time.RFC3339)
	}

	s.mu.RLock()
	indexStats := s.engine.GetIndexStats()
	s.mu.RUnlock()

	totalRules := int64(0)
	rulesByTag := int64(0)
	indexingEnabled := false
	tagCount := int64(0)

	if v, ok := indexStats["total_rules"].(int); ok {
		totalRules = int64(v)
	}
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
  "queries": {
    "total": %d,
    "ok": %d,
    "denied": %d
  },
  "adds": %d,
  "reloads": {
    "total": %d,
    "failed": %d,
    "last": %q
  },
  "connections": %d,
  "rules": {
    "loaded": %d,
    "total": %d,
    "by_tag": %d
  },
  "indexing": {
    "enabled": %t,
    "tags": %d
  }
}`,
		s.metrics.queriesTotal.Load(),
		s.metrics.queriesOK.Load(),
		s.metrics.queriesDenied.Load(),
		s.metrics.addsTotal.Load(),
		s.metrics.reloadsTotal.Load(),
		s.metrics.reloadsFailed.Load(),
		lastReload,
		s.metrics.connectionsTotal.Load(),
		s.metrics.rulesLoaded.Load(),
		totalRules,
		rulesByTag,
		indexingEnabled,
		tagCount,
	)
}
