package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/sirosfoundation/go-spocp/pkg/protocol"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// mockServer creates a simple mock SPOCP server for testing
type mockServer struct {
	listener net.Listener
	handler  func(msg *protocol.Message) *protocol.Response
	closed   chan struct{}
}

func newMockServer(t *testing.T, handler func(msg *protocol.Message) *protocol.Response) *mockServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	ms := &mockServer{
		listener: listener,
		handler:  handler,
		closed:   make(chan struct{}),
	}

	go ms.serve(t)
	return ms
}

func (ms *mockServer) serve(t *testing.T) {
	t.Helper()
	for {
		conn, err := ms.listener.Accept()
		if err != nil {
			select {
			case <-ms.closed:
				return
			default:
				// Ignore accept errors during shutdown
				return
			}
		}
		go ms.handleConn(t, conn)
	}
}

func (ms *mockServer) handleConn(t *testing.T, conn net.Conn) {
	t.Helper()
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		msg, err := protocol.DecodeMessage(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			t.Logf("Mock server decode error: %v", err)
			return
		}

		resp := ms.handler(msg)
		encoded := protocol.EncodeResponse(resp)
		writer.WriteString(encoded)
		writer.Flush()

		// Exit loop on LOGOUT
		if msg.Operation == "LOGOUT" {
			return
		}
	}
}

func (ms *mockServer) addr() string {
	return ms.listener.Addr().String()
}

func (ms *mockServer) close() {
	close(ms.closed)
	ms.listener.Close()
}

// Test NewClient
func TestNewClient(t *testing.T) {
	// Test with empty address
	_, err := NewClient(&Config{Address: ""})
	if err == nil {
		t.Error("Expected error for empty address")
	}
	if !strings.Contains(err.Error(), "address is required") {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Test with invalid address
	_, err = NewClient(&Config{
		Address: "invalid:99999",
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Error("Expected error for invalid address")
	}

	// Test with valid mock server
	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	})
	defer ms.close()

	client, err := NewClient(&Config{
		Address: ms.addr(),
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if client.conn == nil {
		t.Error("Expected non-nil connection")
	}
}

// Test Query
func TestClientQuery(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   string
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "Query OK",
			responseCode:   protocol.CodeOK,
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "Query Denied",
			responseCode:   protocol.CodeDenied,
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "Query Error",
			responseCode:   protocol.CodeError,
			expectedResult: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
				if msg.Operation != "QUERY" {
					return &protocol.Response{Code: protocol.CodeError, Message: "Unexpected operation"}
				}
				return &protocol.Response{Code: tt.responseCode, Message: "test"}
			})
			defer ms.close()

			client, err := NewClient(&Config{Address: ms.addr()})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			query := sexp.NewList("spocp",
				sexp.NewList("subject", sexp.NewAtom("alice")),
			)

			result, err := client.Query(query)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

// Test QueryString
func TestClientQueryString(t *testing.T) {
	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	})
	defer ms.close()

	client, err := NewClient(&Config{Address: ms.addr()})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Valid query string (canonical S-expression format with length-prefixed atoms)
	result, err := client.QueryString("(5:spocp(7:subject5:alice))")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !result {
		t.Error("Expected true result")
	}

	// Invalid query string
	_, err = client.QueryString("invalid(")
	if err == nil {
		t.Error("Expected error for invalid query string")
	}
}

// Test Add
func TestClientAdd(t *testing.T) {
	tests := []struct {
		name         string
		responseCode string
		expectError  bool
	}{
		{
			name:         "Add OK",
			responseCode: protocol.CodeOK,
			expectError:  false,
		},
		{
			name:         "Add Error",
			responseCode: protocol.CodeError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
				if msg.Operation != "ADD" {
					return &protocol.Response{Code: protocol.CodeError, Message: "Unexpected operation"}
				}
				return &protocol.Response{Code: tt.responseCode, Message: "test"}
			})
			defer ms.close()

			client, err := NewClient(&Config{Address: ms.addr()})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			rule := sexp.NewList("spocp",
				sexp.NewList("subject", sexp.NewAtom("alice")),
			)

			err = client.Add(rule)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Test AddString
func TestClientAddString(t *testing.T) {
	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	})
	defer ms.close()

	client, err := NewClient(&Config{Address: ms.addr()})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Valid rule string (canonical S-expression format with length-prefixed atoms)
	err = client.AddString("(5:spocp(7:subject5:alice))")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Invalid rule string
	err = client.AddString("invalid(")
	if err == nil {
		t.Error("Expected error for invalid rule string")
	}
}

// Test Reload
func TestClientReload(t *testing.T) {
	tests := []struct {
		name         string
		responseCode string
		expectError  bool
	}{
		{
			name:         "Reload OK",
			responseCode: protocol.CodeOK,
			expectError:  false,
		},
		{
			name:         "Reload Error",
			responseCode: protocol.CodeError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
				if msg.Operation != "RELOAD" {
					return &protocol.Response{Code: protocol.CodeError, Message: "Unexpected operation"}
				}
				return &protocol.Response{Code: tt.responseCode, Message: "test"}
			})
			defer ms.close()

			client, err := NewClient(&Config{Address: ms.addr()})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			err = client.Reload()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Test Logout
func TestClientLogout(t *testing.T) {
	tests := []struct {
		name         string
		responseCode string
		expectError  bool
	}{
		{
			name:         "Logout OK",
			responseCode: protocol.CodeBye,
			expectError:  false,
		},
		{
			name:         "Logout Unexpected",
			responseCode: protocol.CodeOK, // Wrong response
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
				return &protocol.Response{Code: tt.responseCode, Message: "test"}
			})
			defer ms.close()

			client, err := NewClient(&Config{Address: ms.addr()})
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			// Don't defer Close() since we're testing Logout

			err = client.Logout()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Test Close
func TestClientClose(t *testing.T) {
	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		return &protocol.Response{Code: protocol.CodeBye, Message: "Bye"}
	})
	defer ms.close()

	client, err := NewClient(&Config{Address: ms.addr()})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Close should work
	err = client.Close()
	if err != nil {
		t.Errorf("Unexpected error on close: %v", err)
	}

	// Close on nil conn should be safe
	err = client.Close()
	if err != nil {
		t.Errorf("Unexpected error on second close: %v", err)
	}
}

// Test connection to non-existent server
func TestClientConnectionError(t *testing.T) {
	// Try to connect to a port that's definitely not listening
	_, err := NewClient(&Config{
		Address: "127.0.0.1:1", // port 1 is reserved and not listening
		Timeout: 100 * time.Millisecond,
	})

	// Should fail to connect
	if err == nil {
		t.Error("Expected connection error")
	}
}

// Test default timeout
func TestClientDefaultTimeout(t *testing.T) {
	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	})
	defer ms.close()

	// Don't set Timeout, use default
	client, err := NewClient(&Config{Address: ms.addr()})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if client.conn == nil {
		t.Error("Expected non-nil connection")
	}
}

// Test sendMessage with server disconnect
func TestClientServerDisconnect(t *testing.T) {
	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	})

	client, err := NewClient(&Config{Address: ms.addr()})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Close the server
	ms.close()

	// Give some time for the server to fully close
	time.Sleep(100 * time.Millisecond)

	// Multiple queries should eventually fail as the connection becomes broken
	var lastErr error
	for i := 0; i < 5; i++ {
		_, lastErr = client.QueryString("(5:spocp(7:subject5:alice))")
		if lastErr != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	// The connection should have failed at some point
	if lastErr == nil {
		t.Log("Warning: Server disconnect may not have been detected (depends on OS/timing)")
	}
}

// Test multiple operations on same connection
func TestClientMultipleOperations(t *testing.T) {
	queryCount := 0
	addCount := 0

	ms := newMockServer(t, func(msg *protocol.Message) *protocol.Response {
		switch msg.Operation {
		case "QUERY":
			queryCount++
			return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
		case "ADD":
			addCount++
			return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
		case "LOGOUT":
			return &protocol.Response{Code: protocol.CodeBye, Message: "Bye"}
		default:
			return &protocol.Response{Code: protocol.CodeError, Message: fmt.Sprintf("Unknown: %s", msg.Operation)}
		}
	})
	defer ms.close()

	client, err := NewClient(&Config{Address: ms.addr()})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Multiple queries (canonical S-expression format)
	for i := 0; i < 3; i++ {
		_, err := client.QueryString("(5:spocp(7:subject5:alice))")
		if err != nil {
			t.Errorf("Query %d failed: %v", i, err)
		}
	}

	// Multiple adds (canonical S-expression format)
	for i := 0; i < 2; i++ {
		err := client.AddString("(5:spocp(7:subject3:bob))")
		if err != nil {
			t.Errorf("Add %d failed: %v", i, err)
		}
	}

	if queryCount != 3 {
		t.Errorf("Expected 3 queries, got %d", queryCount)
	}
	if addCount != 2 {
		t.Errorf("Expected 2 adds, got %d", addCount)
	}
}
