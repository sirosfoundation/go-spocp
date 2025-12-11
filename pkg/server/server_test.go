package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/protocol"
)

// Helper to create a temp directory with rule files
func createTempRulesDir(t *testing.T, rules []string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "spocp-server-test-*")
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

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	// Test with empty address
	_, err := NewServer(&Config{Address: ""})
	if err == nil {
		t.Error("Expected error for empty address")
	}

	// Test with no rules dir and no engine
	_, err = NewServer(&Config{Address: ":0"})
	if err == nil {
		t.Error("Expected error when neither engine nor rules dir provided")
	}

	// Test with non-existent rules dir
	_, err = NewServer(&Config{
		Address:  ":0",
		RulesDir: "/nonexistent/path",
	})
	if err == nil {
		t.Error("Expected error for non-existent rules dir")
	}

	// Test with valid rules directory
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:  ":0",
		RulesDir: rulesDir,
		LogLevel: LogLevelDebug,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	if srv == nil {
		t.Error("Expected non-nil server")
	}
}

// TestNewServerWithEngine tests server creation with pre-existing engine
func TestNewServerWithEngine(t *testing.T) {
	engine := spocp.NewEngine()
	engine.AddRule("(4:read)")

	srv, err := NewServer(&Config{
		Address: ":0",
		Engine:  engine,
	})
	if err != nil {
		t.Fatalf("Failed to create server with engine: %v", err)
	}
	defer srv.Close()

	if srv == nil {
		t.Error("Expected non-nil server")
	}

	// Verify the engine is used
	gotEngine := srv.GetEngine()
	if gotEngine != engine {
		t.Error("Expected server to use provided engine")
	}
}

// TestGetEngineMutex tests the GetEngineMutex method
func TestGetEngineMutex(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:  ":0",
		RulesDir: rulesDir,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	mu := srv.GetEngineMutex()
	if mu == nil {
		t.Error("Expected non-nil mutex")
	}

	// Test that mutex is usable - verify engine access under lock
	mu.RLock()
	engine := srv.GetEngine()
	if engine == nil {
		mu.RUnlock()
		t.Error("Expected non-nil engine under lock")
		return
	}
	mu.RUnlock()
}

// TestHandleMessage tests message handling
func TestHandleMessage(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:  ":0",
		RulesDir: rulesDir,
		LogLevel: LogLevelDebug,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	tests := []struct {
		name         string
		message      *protocol.Message
		expectedCode string
	}{
		{
			name: "QUERY OK",
			message: &protocol.Message{
				Operation: "QUERY",
				Arguments: []string{"(4:read)"},
			},
			expectedCode: protocol.CodeOK,
		},
		{
			name: "QUERY Denied",
			message: &protocol.Message{
				Operation: "QUERY",
				Arguments: []string{"(5:write)"},
			},
			expectedCode: protocol.CodeDenied,
		},
		{
			name: "QUERY error - no args",
			message: &protocol.Message{
				Operation: "QUERY",
				Arguments: []string{},
			},
			expectedCode: protocol.CodeError,
		},
		{
			name: "QUERY error - invalid sexp",
			message: &protocol.Message{
				Operation: "QUERY",
				Arguments: []string{"invalid("},
			},
			expectedCode: protocol.CodeError,
		},
		{
			name: "ADD OK",
			message: &protocol.Message{
				Operation: "ADD",
				Arguments: []string{"(5:write)"},
			},
			expectedCode: protocol.CodeOK,
		},
		{
			name: "ADD error - no args",
			message: &protocol.Message{
				Operation: "ADD",
				Arguments: []string{},
			},
			expectedCode: protocol.CodeError,
		},
		{
			name: "ADD error - invalid sexp",
			message: &protocol.Message{
				Operation: "ADD",
				Arguments: []string{"invalid("},
			},
			expectedCode: protocol.CodeError,
		},
		{
			name: "LOGOUT",
			message: &protocol.Message{
				Operation: "LOGOUT",
				Arguments: []string{},
			},
			expectedCode: protocol.CodeBye,
		},
		{
			name: "RELOAD",
			message: &protocol.Message{
				Operation: "RELOAD",
				Arguments: []string{},
			},
			expectedCode: protocol.CodeOK,
		},
		{
			name: "Unknown operation",
			message: &protocol.Message{
				Operation: "UNKNOWN",
				Arguments: []string{},
			},
			expectedCode: protocol.CodeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := srv.handleMessage(tt.message)
			if resp.Code != tt.expectedCode {
				t.Errorf("Expected code %s, got %s: %s", tt.expectedCode, resp.Code, resp.Message)
			}
		})
	}
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:    ":0",
		RulesDir:   rulesDir,
		HealthAddr: ":0",
		LogLevel:   LogLevelInfo,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Test health endpoint
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

// TestReadyEndpoint tests the readiness endpoint
func TestReadyEndpoint(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:    ":0",
		RulesDir:   rulesDir,
		HealthAddr: ":0",
		LogLevel:   LogLevelInfo,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Test ready endpoint with rules loaded
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.handleReady(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestReadyEndpointNotReady tests readiness endpoint without rules
func TestReadyEndpointNotReady(t *testing.T) {
	engine := spocp.NewEngine()
	// Don't add any rules

	srv, err := NewServer(&Config{
		Address:    ":0",
		Engine:     engine,
		HealthAddr: ":0",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Test ready endpoint without rules
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	srv.handleReady(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 without rules, got %d", resp.StatusCode)
	}
}

// TestMetricsEndpoint tests the metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:    ":0",
		RulesDir:   rulesDir,
		HealthAddr: ":0",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

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
	if !strings.Contains(bodyStr, "spocp_queries_total") {
		t.Error("Expected spocp_queries_total metric")
	}
	if !strings.Contains(bodyStr, "spocp_rules_loaded") {
		t.Error("Expected spocp_rules_loaded metric")
	}
}

// TestStatsEndpoint tests the stats endpoint
func TestStatsEndpoint(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:    ":0",
		RulesDir:   rulesDir,
		HealthAddr: ":0",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()

	srv.handleStats(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	// Parse as JSON to verify structure
	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("Failed to parse stats JSON: %v", err)
	}

	if _, ok := stats["queries"]; !ok {
		t.Error("Expected queries in stats")
	}
	if _, ok := stats["rules"]; !ok {
		t.Error("Expected rules in stats")
	}
}

// TestPidFile tests PID file creation and removal
func TestPidFile(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	pidFile := filepath.Join(rulesDir, "test.pid")

	srv, err := NewServer(&Config{
		Address:  ":0",
		RulesDir: rulesDir,
		PidFile:  pidFile,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Check PID file was created
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Error("Expected PID file to be created")
	}

	// Close server
	srv.Close()

	// Check PID file was removed
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("Expected PID file to be removed after close")
	}
}

// TestLogging tests different log levels
func TestLogging(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	levels := []LogLevel{
		LogLevelSilent,
		LogLevelError,
		LogLevelWarn,
		LogLevelInfo,
		LogLevelDebug,
	}

	for _, level := range levels {
		srv, err := NewServer(&Config{
			Address:  ":0",
			RulesDir: rulesDir,
			LogLevel: level,
		})
		if err != nil {
			t.Fatalf("Failed to create server with log level %d: %v", level, err)
		}
		srv.Close()
	}
}

// TestReloadRules tests rule reloading
func TestReloadRules(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:  ":0",
		RulesDir: rulesDir,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Reload should succeed
	resp := srv.handleReload()
	if resp.Code != protocol.CodeOK {
		t.Errorf("Expected reload OK, got %s: %s", resp.Code, resp.Message)
	}
}

// TestClientConnection tests a full client-server interaction
func TestClientConnection(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:  "127.0.0.1:0",
		RulesDir: rulesDir,
		LogLevel: LogLevelDebug,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go srv.Serve()
	defer srv.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Get the actual address
	addr := srv.listener.Addr().String()

	// Connect as client
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send a QUERY message
	msg := protocol.EncodeMessage(&protocol.Message{
		Operation: "QUERY",
		Arguments: []string{"(4:read)"},
	})
	conn.Write([]byte(msg))

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, protocol.CodeOK) {
		t.Errorf("Expected OK response, got: %s", response)
	}
}

// TestEmptyRulesDir tests behavior with empty rules directory
func TestEmptyRulesDir(t *testing.T) {
	// Create empty directory
	dir, err := os.MkdirTemp("", "spocp-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Server creation should succeed, but no rules are loaded (just logs a warning)
	srv, err := NewServer(&Config{
		Address:  ":0",
		RulesDir: dir,
	})
	if err != nil {
		t.Fatalf("Server creation failed: %v", err)
	}
	defer srv.Close()

	// Verify that no rules are loaded
	if srv.metrics.rulesLoaded.Load() != 0 {
		t.Errorf("Expected 0 rules loaded, got %d", srv.metrics.rulesLoaded.Load())
	}
}

// TestHealthCheckStart tests starting the health check endpoint
func TestHealthCheckStart(t *testing.T) {
	rulesDir := createTempRulesDir(t, []string{"(4:read)"})
	defer os.RemoveAll(rulesDir)

	srv, err := NewServer(&Config{
		Address:    "127.0.0.1:0",
		RulesDir:   rulesDir,
		HealthAddr: "127.0.0.1:0",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer srv.Close()

	// Verify health listener is set
	if srv.healthListener == nil {
		t.Error("Expected health listener to be set")
	}

	// Give health endpoint time to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to health endpoint
	healthAddr := srv.healthListener.Addr().String()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+healthAddr+"/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
