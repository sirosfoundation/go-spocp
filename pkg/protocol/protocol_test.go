package protocol

import (
	"bufio"
	"strings"
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestEncodeLV(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "foobar", "6:foobar"},
		{"empty", "", "0:"},
		{"QUERY", "QUERY", "5:QUERY"},
		{"code", "200", "3:200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeLV(tt.input)
			if got != tt.want {
				t.Errorf("encodeLV(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadLV(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"simple", "6:foobar", "foobar", false},
		{"empty", "0:", "", false},
		{"QUERY", "5:QUERY", "QUERY", false},
		{"with remainder", "3:foobar", "foo", false},
		{"invalid length", "abc:foo", "", true},
		{"negative length", "-5:hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := readLV(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readLV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("readLV() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEncodeDecodeMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "QUERY with arguments",
			msg: &Message{
				Operation: "QUERY",
				Arguments: []string{"(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))"},
			},
		},
		{
			name: "ADD with rule",
			msg: &Message{
				Operation: "ADD",
				Arguments: []string{"(4:http(4:page)(6:action3:GET)(6:userid))"},
			},
		},
		{
			name: "LOGOUT no arguments",
			msg: &Message{
				Operation: "LOGOUT",
				Arguments: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := EncodeMessage(tt.msg)

			// Decode
			r := bufio.NewReader(strings.NewReader(encoded))
			decoded, err := DecodeMessage(r)
			if err != nil {
				t.Fatalf("DecodeMessage() error = %v", err)
			}

			// Compare
			if decoded.Operation != tt.msg.Operation {
				t.Errorf("Operation = %q, want %q", decoded.Operation, tt.msg.Operation)
			}
			if len(decoded.Arguments) != len(tt.msg.Arguments) {
				t.Errorf("Arguments length = %d, want %d", len(decoded.Arguments), len(tt.msg.Arguments))
			}
			for i := range decoded.Arguments {
				if decoded.Arguments[i] != tt.msg.Arguments[i] {
					t.Errorf("Argument[%d] = %q, want %q", i, decoded.Arguments[i], tt.msg.Arguments[i])
				}
			}
		})
	}
}

func TestEncodeDecodeResponse(t *testing.T) {
	tests := []struct {
		name string
		resp *Response
	}{
		{
			name: "OK response",
			resp: &Response{Code: CodeOK, Message: "Ok"},
		},
		{
			name: "Bye response",
			resp: &Response{Code: CodeBye, Message: "Bye"},
		},
		{
			name: "Error response",
			resp: &Response{Code: CodeError, Message: "Internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := EncodeResponse(tt.resp)

			// Decode
			r := bufio.NewReader(strings.NewReader(encoded))
			decoded, err := DecodeResponse(r)
			if err != nil {
				t.Fatalf("DecodeResponse() error = %v", err)
			}

			// Compare
			if decoded.Code != tt.resp.Code {
				t.Errorf("Code = %q, want %q", decoded.Code, tt.resp.Code)
			}
			if decoded.Message != tt.resp.Message {
				t.Errorf("Message = %q, want %q", decoded.Message, tt.resp.Message)
			}
		})
	}
}

func TestProtocolExample(t *testing.T) {
	// Example from the spec:
	// C: 70:5:QUERY60:(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))
	// S: 9:3:2002:Ok

	msg := &Message{
		Operation: "QUERY",
		Arguments: []string{"(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))"},
	}

	encoded := EncodeMessage(msg)
	expected := "70:5:QUERY60:(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))"

	if encoded != expected {
		t.Errorf("EncodeMessage() = %q, want %q", encoded, expected)
	}

	// Test response - the format is L(code:message) where code is "200" and message is "Ok"
	// So "200:Ok" is 6 bytes, giving us "6:200:Ok"
	resp := &Response{Code: CodeOK, Message: "Ok"}
	encodedResp := EncodeResponse(resp)
	expectedResp := "6:200:Ok"

	if encodedResp != expectedResp {
		t.Errorf("EncodeResponse() = %q, want %q", encodedResp, expectedResp)
	}
}

func TestParseQuery(t *testing.T) {
	queryStr := "(4:http(4:page10:index.html)(6:action3:GET)(6:userid4:olav))"
	elem, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}
	if elem == nil {
		t.Fatal("ParseQuery() returned nil element")
	}

	// Should be a list
	list, ok := elem.(*sexp.List)
	if !ok {
		t.Fatalf("ParseQuery() returned %T, want *sexp.List", elem)
	}
	if len(list.Elements) == 0 {
		t.Error("ParseQuery() returned empty list")
	}
}
