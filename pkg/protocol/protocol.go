// Package protocol implements the SPOCP TCP protocol as defined in draft-hedberg-spocp-tcp-00.package protocol

// The protocol uses a simple length-value (LV) format where messages are encoded as:
//
//	L:value
//
// where L is the decimal length of the value, followed by ':', followed by the value bytes.
//
// Protocol operations use the format:
//
//	L(L'Operand' *L'arg')
//
// Example:
//
//	70:5:QUERY60:(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))
package protocol

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// Message represents a SPOCP protocol message
type Message struct {
	Operation string
	Arguments []string
}

// Response codes as defined in the SPOCP protocol
const (
	CodeOK      = "200"
	CodeBye     = "203"
	CodeDenied  = "400"
	CodeError   = "500"
	CodeUnknown = "501"
)

// Response represents a SPOCP protocol response
type Response struct {
	Code    string
	Message string
}

// EncodeMessage encodes a message into the SPOCP protocol format
// Format: L(L'Operand' *L'arg')
func EncodeMessage(msg *Message) string {
	var parts []string

	// Encode operation
	parts = append(parts, encodeLV(msg.Operation))

	// Encode arguments
	for _, arg := range msg.Arguments {
		parts = append(parts, encodeLV(arg))
	}

	// Join all parts
	inner := strings.Join(parts, "")

	// Wrap in final length-value encoding
	return encodeLV(inner)
}

// EncodeResponse encodes a response into the SPOCP protocol format
func EncodeResponse(resp *Response) string {
	return encodeLV(fmt.Sprintf("%s:%s", resp.Code, resp.Message))
}

// encodeLV encodes a string as length:value
func encodeLV(s string) string {
	return fmt.Sprintf("%d:%s", len(s), s)
}

// DecodeMessage decodes a SPOCP protocol message from a reader
func DecodeMessage(r *bufio.Reader) (*Message, error) {
	// Read the outer LV wrapper
	outer, err := readLV(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read outer message: %w", err)
	}

	// Parse the inner content
	innerReader := bufio.NewReader(strings.NewReader(outer))

	// Read operation
	operation, err := readLV(innerReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read operation: %w", err)
	}

	// Read arguments
	var arguments []string
	for {
		arg, err := readLV(innerReader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read argument: %w", err)
		}
		arguments = append(arguments, arg)
	}

	return &Message{
		Operation: operation,
		Arguments: arguments,
	}, nil
}

// DecodeResponse decodes a SPOCP protocol response from a reader
func DecodeResponse(r *bufio.Reader) (*Response, error) {
	content, err := readLV(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse code:message format
	parts := strings.SplitN(content, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid response format: %s", content)
	}

	return &Response{
		Code:    parts[0],
		Message: parts[1],
	}, nil
}

// readLV reads a length-value encoded string from a reader
// Format: <decimal-length>:<value>
func readLV(r *bufio.Reader) (string, error) {
	// Read length until ':'
	lengthStr, err := r.ReadString(':')
	if err != nil {
		return "", err
	}

	// Remove the ':'
	lengthStr = strings.TrimSuffix(lengthStr, ":")

	// Parse length
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", fmt.Errorf("invalid length '%s': %w", lengthStr, err)
	}

	if length < 0 {
		return "", fmt.Errorf("negative length not allowed: %d", length)
	}

	if length == 0 {
		return "", nil
	}

	// Read exact number of bytes
	value := make([]byte, length)
	_, err = io.ReadFull(r, value)
	if err != nil {
		return "", fmt.Errorf("failed to read value of length %d: %w", length, err)
	}

	return string(value), nil
}

// ParseQuery parses a query argument into an S-expression element
func ParseQuery(queryStr string) (sexp.Element, error) {
	parser := sexp.NewParser(queryStr)
	return parser.Parse()
}

// ParseRule parses a rule argument into an S-expression element
func ParseRule(ruleStr string) (sexp.Element, error) {
	parser := sexp.NewParser(ruleStr)
	return parser.Parse()
}
