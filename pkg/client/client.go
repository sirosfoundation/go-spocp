// Package client implements a SPOCP TCP client with TLS support.
package client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/sirosfoundation/go-spocp/pkg/protocol"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// Client represents a SPOCP TCP client
type Client struct {
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	tlsConfig *tls.Config
}

// Config contains client configuration
type Config struct {
	// Server address (e.g., "localhost:6000")
	Address string

	// TLS configuration (optional, nil for plain TCP)
	TLSConfig *tls.Config

	// Connection timeout
	Timeout time.Duration
}

// NewClient creates a new SPOCP client and connects to the server
func NewClient(config *Config) (*Client, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	var conn net.Conn
	var err error

	if config.TLSConfig != nil {
		dialer := &net.Dialer{Timeout: timeout}
		conn, err = tls.DialWithDialer(dialer, "tcp", config.Address, config.TLSConfig)
	} else {
		conn, err = net.DialTimeout("tcp", config.Address, timeout)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Address, err)
	}

	return &Client{
		conn:      conn,
		reader:    bufio.NewReader(conn),
		writer:    bufio.NewWriter(conn),
		tlsConfig: config.TLSConfig,
	}, nil
}

// Close closes the connection to the server
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	// Try to send LOGOUT, but don't fail if it doesn't work
	// (connection might already be closed)
	defer func() {
		c.conn.Close()
		c.conn = nil
	}()

	// Try to logout gracefully
	msg := &protocol.Message{
		Operation: "LOGOUT",
		Arguments: []string{},
	}

	// Set a short timeout for logout
	_ = c.conn.SetWriteDeadline(time.Now().Add(1 * time.Second)) //nolint:errcheck // best-effort logout
	encoded := protocol.EncodeMessage(msg)
	_, _ = c.writer.WriteString(encoded) //nolint:errcheck // best-effort logout
	_ = c.writer.Flush()                 //nolint:errcheck // best-effort logout

	return nil
}

// Query sends a QUERY operation to the server
func (c *Client) Query(query sexp.Element) (bool, error) {
	msg := &protocol.Message{
		Operation: "QUERY",
		Arguments: []string{query.String()},
	}

	resp, err := c.sendMessage(msg)
	if err != nil {
		return false, err
	}

	switch resp.Code {
	case protocol.CodeOK:
		return true, nil
	case protocol.CodeDenied:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected response: %s %s", resp.Code, resp.Message)
	}
}

// QueryString sends a QUERY operation using a canonical S-expression string
func (c *Client) QueryString(queryStr string) (bool, error) {
	query, err := protocol.ParseQuery(queryStr)
	if err != nil {
		return false, fmt.Errorf("invalid query: %w", err)
	}
	return c.Query(query)
}

// Add sends an ADD operation to the server
func (c *Client) Add(rule sexp.Element) error {
	msg := &protocol.Message{
		Operation: "ADD",
		Arguments: []string{rule.String()},
	}

	resp, err := c.sendMessage(msg)
	if err != nil {
		return err
	}

	if resp.Code != protocol.CodeOK {
		return fmt.Errorf("add failed: %s %s", resp.Code, resp.Message)
	}

	return nil
}

// AddString sends an ADD operation using a canonical S-expression string
func (c *Client) AddString(ruleStr string) error {
	rule, err := protocol.ParseRule(ruleStr)
	if err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}
	return c.Add(rule)
}

// Reload sends a RELOAD operation to the server
func (c *Client) Reload() error {
	msg := &protocol.Message{
		Operation: "RELOAD",
		Arguments: []string{},
	}

	resp, err := c.sendMessage(msg)
	if err != nil {
		return err
	}

	if resp.Code != protocol.CodeOK {
		return fmt.Errorf("reload failed: %s %s", resp.Code, resp.Message)
	}

	return nil
}

// Logout sends a LOGOUT operation to the server
func (c *Client) Logout() error {
	msg := &protocol.Message{
		Operation: "LOGOUT",
		Arguments: []string{},
	}

	resp, err := c.sendMessage(msg)
	if err != nil {
		return err
	}

	if resp.Code != protocol.CodeBye {
		return fmt.Errorf("unexpected logout response: %s %s", resp.Code, resp.Message)
	}

	return nil
}

// sendMessage sends a message and receives a response
func (c *Client) sendMessage(msg *protocol.Message) (*protocol.Response, error) {
	// Encode and send message
	encoded := protocol.EncodeMessage(msg)
	if _, err := c.writer.WriteString(encoded); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush: %w", err)
	}

	// Set read deadline
	_ = c.conn.SetReadDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck // non-critical timeout setting

	// Read response
	resp, err := protocol.DecodeResponse(c.reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, nil
}
