package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/authzen"
	"github.com/sirosfoundation/go-spocp/pkg/server"
)

// Helper to create a test engine with rules
func createTestEngine(rules []string) *spocp.Engine {
	engine := spocp.NewEngine()
	for _, rule := range rules {
		engine.AddRule(rule)
	}
	return engine
}

// Helper to create a temp directory with rule files
func createTempRulesDir(t *testing.T, rules []string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "spocp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a .spoc file with canonical S-expression rules
	ruleFile := filepath.Join(dir, "test.spoc")
	var content bytes.Buffer
	for _, rule := range rules {
		content.WriteString(rule)
		content.WriteString("\n")
	}
	if err := os.WriteFile(ruleFile, content.Bytes(), 0644); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("Failed to write rules file: %v", err)
	}

	return dir
}

// TestNewHTTPServer tests server creation
func TestNewHTTPServer(t *testing.T) {
	// Test with empty address
	_, err := NewHTTPServer(&Config{Address: ""})
	if err == nil {
		t.Error("Expected error for empty address")
	}

	// Test with neither engine nor rules dir
	_, err = NewHTTPServer(&Config{Address: ":0"})
	if err == nil {
		t.Error("Expected error when neither engine nor rules dir provided")
	}

	// Test with pre-existing engine
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address: ":0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server with engine: %v", err)
	}
	if srv == nil {
		t.Error("Expected non-nil server")
	}

	// Test with rules directory
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err = NewHTTPServer(&Config{
		Address:  ":0",
		RulesDir: rulesDir,
	})
	if err != nil {
		t.Fatalf("Failed to create server with rules dir: %v", err)
	}
	if srv == nil {
		t.Error("Expected non-nil server")
	}
}

// TestNewHTTPServerWithMutex tests shared mutex handling
func TestNewHTTPServerWithMutex(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	sharedMutex := &sync.RWMutex{}

	srv, err := NewHTTPServer(&Config{
		Address:     ":0",
		Engine:      engine,
		EngineMutex: sharedMutex,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Verify the shared mutex is used
	if srv.mu != sharedMutex {
		t.Error("Expected server to use shared mutex")
	}
}

// TestHealthEndpoint tests the /health endpoint
func TestHealthEndpoint(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address: ":0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("Unexpected body: %s", body)
	}
}

// TestReadyEndpoint tests the /ready endpoint
func TestReadyEndpoint(t *testing.T) {
	// Test with rules loaded
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address: ":0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.handleReady(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 with rules, got %d", resp.StatusCode)
	}

	// Test with no rules loaded
	emptyEngine := spocp.NewEngine()
	srv2, err := NewHTTPServer(&Config{
		Address: ":0",
		Engine:  emptyEngine,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w2 := httptest.NewRecorder()

	srv2.handleReady(w2, req2)

	resp2 := w2.Result()
	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 without rules, got %d", resp2.StatusCode)
	}
}

// TestMetricsEndpoint tests the /metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address: ":0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	srv.handleMetrics(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check for Prometheus-style metrics
	if !strings.Contains(bodyStr, "spocp_http_requests_total") {
		t.Error("Expected spocp_http_requests_total metric")
	}
	if !strings.Contains(bodyStr, "spocp_rules_loaded") {
		t.Error("Expected spocp_rules_loaded metric")
	}
}

// TestStatsEndpoint tests the /stats endpoint
func TestStatsEndpoint(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address: ":0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()

	srv.handleStats(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check for JSON structure
	if !strings.Contains(bodyStr, `"requests"`) {
		t.Error("Expected requests in stats")
	}
	if !strings.Contains(bodyStr, `"rules"`) {
		t.Error("Expected rules in stats")
	}
}

// TestEvaluationEndpoint tests the AuthZen /access/v1/evaluation endpoint
func TestEvaluationEndpoint(t *testing.T) {
	// Create engine with a rule that matches AuthZen query format exactly
	// AuthZen generates: (resource-type (id resource-id)(action action-name)(subject (type sub-type)(id sub-id)))
	// For "account" resource with id "123", "can_read" action, "alice" user:
	// Query: (7:account(2:id3:123)(6:action8:can_read)(7:subject(4:type4:user)(2:id5:alice)))
	engine := createTestEngine([]string{
		"(7:account(2:id3:123)(6:action8:can_read)(7:subject(4:type4:user)(2:id5:alice)))",
	})

	srv, err := NewHTTPServer(&Config{
		Address:       ":0",
		Engine:        engine,
		EnableAuthZen: true,
		LogLevel:      server.LogLevelDebug,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name             string
		method           string
		request          *authzen.EvaluationRequest
		expectedStatus   int
		expectedDecision bool
	}{
		{
			name:   "Valid request - allowed",
			method: http.MethodPost,
			request: &authzen.EvaluationRequest{
				Subject:  authzen.Subject{Type: "user", ID: "alice"},
				Resource: authzen.Resource{Type: "account", ID: "123"},
				Action:   authzen.Action{Name: "can_read"},
			},
			expectedStatus:   http.StatusOK,
			expectedDecision: true,
		},
		{
			name:   "Valid request - denied (no matching rule)",
			method: http.MethodPost,
			request: &authzen.EvaluationRequest{
				Subject:  authzen.Subject{Type: "user", ID: "bob"},
				Resource: authzen.Resource{Type: "account", ID: "123"},
				Action:   authzen.Action{Name: "can_read"},
			},
			expectedStatus:   http.StatusOK,
			expectedDecision: false,
		},
		{
			name:           "Wrong method",
			method:         http.MethodGet,
			request:        nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.request != nil {
				jsonData, _ := json.Marshal(tt.request)
				body = bytes.NewReader(jsonData)
			}

			req := httptest.NewRequest(tt.method, "/access/v1/evaluation", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.handleEvaluation(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedStatus == http.StatusOK && tt.request != nil {
				var evalResp authzen.EvaluationResponse
				if err := json.NewDecoder(resp.Body).Decode(&evalResp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if evalResp.Decision != tt.expectedDecision {
					t.Errorf("Expected decision %v, got %v", tt.expectedDecision, evalResp.Decision)
				}
			}
		})
	}
}

// TestEvaluationEndpointBadRequest tests error handling for bad requests
func TestEvaluationEndpointBadRequest(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address:       ":0",
		Engine:        engine,
		EnableAuthZen: true,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/access/v1/evaluation", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleEvaluation(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// TestEvaluationWithRequestID tests X-Request-ID header handling
func TestEvaluationWithRequestID(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address:       ":0",
		Engine:        engine,
		EnableAuthZen: true,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	evalReq := &authzen.EvaluationRequest{
		Subject:  authzen.Subject{Type: "user", ID: "alice"},
		Resource: authzen.Resource{Type: "doc", ID: "1"},
		Action:   authzen.Action{Name: "read"},
	}
	jsonData, _ := json.Marshal(evalReq)

	req := httptest.NewRequest(http.MethodPost, "/access/v1/evaluation", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request-123")
	w := httptest.NewRecorder()

	srv.handleEvaluation(w, req)

	resp := w.Result()
	if resp.Header.Get("X-Request-ID") != "test-request-123" {
		t.Errorf("Expected X-Request-ID to be echoed, got %q", resp.Header.Get("X-Request-ID"))
	}
}

// TestStartAndClose tests server start and shutdown
func TestStartAndClose(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address: "127.0.0.1:0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	if err := srv.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Close server
	if err := srv.Close(); err != nil {
		t.Errorf("Failed to close server: %v", err)
	}
}

// TestGetMetrics tests the GetMetrics method
func TestGetMetrics(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})
	srv, err := NewHTTPServer(&Config{
		Address:       ":0",
		Engine:        engine,
		EnableAuthZen: true,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Make some requests to generate metrics
	evalReq := &authzen.EvaluationRequest{
		Subject:  authzen.Subject{Type: "user", ID: "alice"},
		Resource: authzen.Resource{Type: "doc", ID: "1"},
		Action:   authzen.Action{Name: "read"},
	}
	jsonData, _ := json.Marshal(evalReq)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/access/v1/evaluation", bytes.NewReader(jsonData))
		w := httptest.NewRecorder()
		srv.handleEvaluation(w, req)
	}

	metrics := srv.GetMetrics()
	if metrics["requests_total"] != 3 {
		t.Errorf("Expected 3 requests_total, got %d", metrics["requests_total"])
	}
}

// TestLogging tests logging at different levels
func TestLogging(t *testing.T) {
	engine := createTestEngine([]string{"(4:read)"})

	// Test different log levels
	levels := []server.LogLevel{
		server.LogLevelSilent,
		server.LogLevelError,
		server.LogLevelWarn,
		server.LogLevelInfo,
		server.LogLevelDebug,
	}

	for _, level := range levels {
		srv, err := NewHTTPServer(&Config{
			Address:  ":0",
			Engine:   engine,
			LogLevel: level,
		})
		if err != nil {
			t.Fatalf("Failed to create server with log level %d: %v", level, err)
		}
		if srv == nil {
			t.Errorf("Expected non-nil server with log level %d", level)
		}
	}
}

// TestLoadRulesFromDirErrors tests error handling for rule loading
func TestLoadRulesFromDirErrors(t *testing.T) {
	// Test with non-existent directory
	_, err := NewHTTPServer(&Config{
		Address:  ":0",
		RulesDir: "/nonexistent/path",
	})
	if err == nil {
		t.Error("Expected error for non-existent rules dir")
	}

	// Test with empty directory (no .spoc files)
	emptyDir, err := os.MkdirTemp("", "spocp-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(emptyDir)

	_, err = NewHTTPServer(&Config{
		Address:  ":0",
		RulesDir: emptyDir,
	})
	if err == nil {
		t.Error("Expected error for empty rules dir")
	}
}
