package sexp

import (
	"testing"
)

func TestParseAtom(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple atom", "5:spocp", "spocp"},
		{"numeric atom", "2:42", "42"},
		{"empty would be invalid", "0:", ""},
		{"special chars", "7:ab:c/de", "ab:c/de"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			elem, err := parser.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			atom, ok := elem.(*Atom)
			if !ok {
				t.Fatalf("expected Atom, got %T", elem)
			}
			if atom.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, atom.Value)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedTag string
		numElements int
	}{
		{
			"simple list",
			"(5:spocp)",
			"spocp",
			0,
		},
		{
			"list with atom",
			"(5:spocp8:Resource)",
			"spocp",
			1,
		},
		{
			"nested list - example from spec",
			"(5:spocp(8:Resource6:mailer))",
			"spocp",
			1,
		},
		{
			"list with multiple elements",
			"(5:fruit5:apple5:large3:red)",
			"fruit",
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			elem, err := parser.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			list, ok := elem.(*List)
			if !ok {
				t.Fatalf("expected List, got %T", elem)
			}
			if list.Tag != tt.expectedTag {
				t.Errorf("expected tag %q, got %q", tt.expectedTag, list.Tag)
			}
			if len(list.Elements) != tt.numElements {
				t.Errorf("expected %d elements, got %d", tt.numElements, len(list.Elements))
			}
		})
	}
}

// Example from spec section 5.2:
// (http (page index.html)(action GET)(user olav))
func TestSpecExample1(t *testing.T) {
	input := "(4:http(4:page10:index.html)(6:action3:GET)(4:user4:olav))"
	parser := NewParser(input)
	elem, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := elem.(*List)
	if list.Tag != "http" {
		t.Errorf("expected tag 'http', got %q", list.Tag)
	}
	if len(list.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(list.Elements))
	}

	// Verify first element is (page index.html)
	page := list.Elements[0].(*List)
	if page.Tag != "page" {
		t.Errorf("expected tag 'page', got %q", page.Tag)
	}
	pageVal := page.Elements[0].(*Atom)
	if pageVal.Value != "index.html" {
		t.Errorf("expected 'index.html', got %q", pageVal.Value)
	}
}

func TestAdvancedForm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple atom",
			"5:spocp",
			"spocp",
		},
		{
			"simple list",
			"(5:spocp8:Resource)",
			"(spocp Resource)",
		},
		{
			"nested list from spec",
			"(5:spocp(8:Resource6:mailer))",
			"(spocp (Resource mailer))",
		},
		{
			"complex example",
			"(5:fruit5:apple5:large3:red)",
			"(fruit apple large red)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			elem, err := parser.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			result := AdvancedForm(elem)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCanonicalFormRoundTrip(t *testing.T) {
	tests := []string{
		"5:hello",
		"(4:test)",
		"(4:test5:value)",
		"(4:http(4:page10:index.html))",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			parser := NewParser(input)
			elem, err := parser.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			output := elem.String()
			if output != input {
				t.Errorf("roundtrip failed: input=%q, output=%q", input, output)
			}
		})
	}
}

func TestParseAll(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			"single expression",
			"(4:http3:GET)",
			1,
		},
		{
			"two expressions same line",
			"(4:http3:GET)(4:http4:POST)",
			2,
		},
		{
			"two expressions with whitespace",
			"(4:http3:GET)  (4:http4:POST)",
			2,
		},
		{
			"expressions on separate lines",
			"(4:http3:GET)\n(4:http4:POST)",
			2,
		},
		{
			"expression with newlines and spaces",
			"  (4:http3:GET)  \n\n  (4:http4:POST)  \n",
			2,
		},
		{
			"three expressions",
			"(4:http3:GET)\n(4:http4:POST)\n(4:http6:DELETE)",
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			elements, err := parser.ParseAll()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(elements) != tt.expected {
				t.Errorf("expected %d elements, got %d", tt.expected, len(elements))
			}
		})
	}
}

func TestParseAllMultiLine(t *testing.T) {
	// Test a complex multi-line s-expression that couldn't be parsed line-by-line
	input := `(7:authzen
(6:tenant)
(6:action5:check)
(8:resource
  (4:type3:jwk)
  (2:id))
(7:subject
  (4:type3:key)
  (2:id)))`

	parser := NewParser(input)
	elements, err := parser.ParseAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elements))
	}

	list, ok := elements[0].(*List)
	if !ok {
		t.Fatalf("expected List, got %T", elements[0])
	}
	if list.Tag != "authzen" {
		t.Errorf("expected tag 'authzen', got %q", list.Tag)
	}
	// Should have 4 sub-elements: tenant, action, resource, subject
	if len(list.Elements) != 4 {
		t.Errorf("expected 4 elements, got %d", len(list.Elements))
	}
}

func TestParseAllFromString(t *testing.T) {
	input := "(4:http3:GET)\n(4:http4:POST)"
	elements, err := ParseAllFromString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(elements))
	}
}

func TestSkipToNextExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		startPos int
		found    bool
		endPos   int
	}{
		{
			"finds opening paren",
			"xxx(4:test)",
			0,
			true,
			3, // position of '('
		},
		{
			"finds digit",
			"xxx5:hello",
			0,
			true,
			3, // position of '5'
		},
		{
			"no expression found",
			"xxxyyy",
			0,
			false,
			6, // end of input
		},
		{
			"already at expression start",
			"(4:test)",
			0,
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			parser.pos = tt.startPos
			found := parser.SkipToNextExpression()
			if found != tt.found {
				t.Errorf("expected found=%v, got %v", tt.found, found)
			}
			if parser.pos != tt.endPos {
				t.Errorf("expected pos=%d, got %d", tt.endPos, parser.pos)
			}
		})
	}
}
