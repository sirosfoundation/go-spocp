// Package server implements a SPOCP TCP server with TLS support and dynamic rule reloading.package server

package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/persist"
	"github.com/sirosfoundation/go-spocp/pkg/protocol"
)

// Server represents a SPOCP TCP server
type Server struct {
	listener    net.Listener
	engine      *spocp.Engine
	rulesDir    string
	tlsConfig   *tls.Config
	mu          sync.RWMutex
	reloadMutex sync.Mutex
	logger      *log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// Config contains server configuration
type Config struct {
	// Address to listen on (e.g., ":6000")
	Address string

	// Directory containing .spoc rule files
	RulesDir string

	// TLS configuration (optional, nil for plain TCP)
	TLSConfig *tls.Config

	// Logger (optional, defaults to standard logger)
	Logger *log.Logger

	// ReloadInterval for automatic rule reloading (0 to disable)
	ReloadInterval time.Duration
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

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "[SPOCP] ", log.LstdFlags)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		engine:    spocp.NewEngine(),
		rulesDir:  config.RulesDir,
		tlsConfig: config.TLSConfig,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Load initial rules
	if err := s.reloadRules(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load initial rules: %w", err)
	}

	// Create listener
	var err error
	if config.TLSConfig != nil {
		s.listener, err = tls.Listen("tcp", config.Address, config.TLSConfig)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create TLS listener: %w", err)
		}
		s.logger.Printf("TLS server listening on %s", config.Address)
	} else {
		s.listener, err = net.Listen("tcp", config.Address)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create listener: %w", err)
		}
		s.logger.Printf("Server listening on %s (plain TCP)", config.Address)
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
	s.logger.Println("Server started, waiting for connections...")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				s.logger.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// Close gracefully shuts down the server
func (s *Server) Close() error {
	s.logger.Println("Shutting down server...")
	s.cancel()

	err := s.listener.Close()

	// Wait for all connections to close
	s.wg.Wait()

	s.logger.Println("Server stopped")
	return err
}

// handleConnection processes a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	s.logger.Printf("New connection from %s", remoteAddr)

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
				s.logger.Printf("Client %s disconnected", remoteAddr)
				return
			}
			s.logger.Printf("Error reading from %s: %v", remoteAddr, err)
			s.sendResponse(writer, &protocol.Response{
				Code:    protocol.CodeError,
				Message: "Protocol error",
			})
			return
		}

		s.logger.Printf("Received from %s: %s %v", remoteAddr, msg.Operation, msg.Arguments)

		// Handle message
		resp := s.handleMessage(msg)

		// Send response
		if err := s.sendResponse(writer, resp); err != nil {
			s.logger.Printf("Error sending response to %s: %v", remoteAddr, err)
			return
		}

		// Close connection if LOGOUT
		if msg.Operation == "LOGOUT" {
			s.logger.Printf("Client %s logged out", remoteAddr)
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
		return &protocol.Response{Code: protocol.CodeOK, Message: "Ok"}
	}
	return &protocol.Response{Code: protocol.CodeDenied, Message: "Denied"}
}

// handleAdd processes an ADD operation
func (s *Server) handleAdd(msg *protocol.Message) *protocol.Response {
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
	if err := s.reloadRules(); err != nil {
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
func (s *Server) reloadRules() error {
	s.reloadMutex.Lock()
	defer s.reloadMutex.Unlock()

	s.logger.Printf("Reloading rules from %s", s.rulesDir)

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
		s.logger.Printf("Warning: No .spoc files found in %s", s.rulesDir)
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

	s.logger.Printf("Loaded %d rules from %d files", totalRules, len(ruleFiles))
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
				s.logger.Printf("Auto-reload failed: %v", err)
			}
		}
	}
}
