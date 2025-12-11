package authzen

import (
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestEvaluationRequestToSExpression(t *testing.T) {
	tests := []struct {
		name    string
		request EvaluationRequest
		want    string
	}{
		{
			name: "simple request",
			request: EvaluationRequest{
				Subject: Subject{
					Type: "user",
					ID:   "alice@acmecorp.com",
				},
				Resource: Resource{
					Type: "account",
					ID:   "123",
				},
				Action: Action{
					Name: "can_read",
				},
			},
			want: "(7:account(2:id3:123)(6:action8:can_read)(7:subject(4:type4:user)(2:id18:alice@acmecorp.com)))",
		},
		{
			name: "request with properties",
			request: EvaluationRequest{
				Subject: Subject{
					Type: "user",
					ID:   "alice@acmecorp.com",
					Properties: map[string]interface{}{
						"department": "Sales",
					},
				},
				Resource: Resource{
					Type: "document",
					ID:   "report.pdf",
				},
				Action: Action{
					Name: "can_update",
					Properties: map[string]interface{}{
						"method": "PUT",
					},
				},
			},
			want: "(8:document(2:id10:report.pdf)(6:action10:can_update(6:method3:PUT))(7:subject(4:type4:user)(2:id18:alice@acmecorp.com)(10:department5:Sales)))",
		},
		{
			name: "request with context",
			request: EvaluationRequest{
				Subject: Subject{
					Type: "user",
					ID:   "bob",
				},
				Resource: Resource{
					Type: "file",
					ID:   "config.txt",
				},
				Action: Action{
					Name: "can_delete",
				},
				Context: Context{
					"time": "1985-10-26T01:22-07:00",
				},
			},
			want: "(4:file(2:id10:config.txt)(6:action10:can_delete)(7:subject(4:type4:user)(2:id3:bob))(7:context(4:time22:1985-10-26T01:22-07:00)))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.request.ToSExpression()
			if err != nil {
				t.Fatalf("ToSExpression() error = %v", err)
			}
			gotStr := got.String()
			if gotStr != tt.want {
				t.Errorf("ToSExpression() = %v, want %v", gotStr, tt.want)
			}

			// Verify it parses back correctly
			parser := sexp.NewParser(gotStr)
			parsed, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse generated S-expression: %v", err)
			}
			if parsed.String() != gotStr {
				t.Errorf("Round-trip failed: %v != %v", parsed.String(), gotStr)
			}
		})
	}
}

func TestPropertyToSExp(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "string value",
			key:   "name",
			value: "Alice",
			want:  "(4:name5:Alice)",
		},
		{
			name:  "boolean true",
			key:   "enabled",
			value: true,
			want:  "(7:enabled4:true)",
		},
		{
			name:  "boolean false",
			key:   "disabled",
			value: false,
			want:  "(8:disabled5:false)",
		},
		{
			name:  "number",
			key:   "count",
			value: float64(42),
			want:  "(5:count2:42)",
		},
		{
			name:  "array of strings",
			key:   "tags",
			value: []interface{}{"admin", "user"},
			want:  "(4:tags5:admin4:user)",
		},
		{
			name: "nested object",
			key:  "metadata",
			value: map[string]interface{}{
				"version": "1.0",
			},
			want: "(8:metadata(7:version3:1.0))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := propertyToSExp(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("propertyToSExp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			gotStr := got.String()
			if gotStr != tt.want {
				t.Errorf("propertyToSExp() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func TestPropertyToSExpIntValue(t *testing.T) {
	got, err := propertyToSExp("count", 42)
	if err != nil {
		t.Fatalf("propertyToSExp() error = %v", err)
	}
	want := "(5:count2:42)"
	if got.String() != want {
		t.Errorf("propertyToSExp() = %v, want %v", got.String(), want)
	}
}

func TestPropertyToSExpFallbackToJSON(t *testing.T) {
	// Test with a type that will trigger JSON marshalling fallback
	type customType struct {
		Value string `json:"value"`
	}
	got, err := propertyToSExp("custom", customType{Value: "test"})
	if err != nil {
		t.Fatalf("propertyToSExp() error = %v", err)
	}
	// Should contain JSON-encoded version
	gotStr := got.String()
	if gotStr == "" {
		t.Error("propertyToSExp() returned empty string")
	}
}

func TestValueToString(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "string",
			value: "hello",
			want:  "hello",
		},
		{
			name:  "bool true",
			value: true,
			want:  "true",
		},
		{
			name:  "bool false",
			value: false,
			want:  "false",
		},
		{
			name:  "float64",
			value: float64(3.14),
			want:  "3.14",
		},
		{
			name:  "int",
			value: 42,
			want:  "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueToString(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("valueToString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("valueToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueToStringFallback(t *testing.T) {
	// Test with a type that triggers JSON marshalling
	type customType struct {
		Value string `json:"value"`
	}
	got, err := valueToString(customType{Value: "test"})
	if err != nil {
		t.Fatalf("valueToString() error = %v", err)
	}
	want := `{"value":"test"}`
	if got != want {
		t.Errorf("valueToString() = %v, want %v", got, want)
	}
}

func TestPropertyToSExpArrayWithNonString(t *testing.T) {
	// Test array with non-string elements
	got, err := propertyToSExp("numbers", []interface{}{float64(1), float64(2), float64(3)})
	if err != nil {
		t.Fatalf("propertyToSExp() error = %v", err)
	}
	want := "(7:numbers1:11:21:3)"
	if got.String() != want {
		t.Errorf("propertyToSExp() = %v, want %v", got.String(), want)
	}
}

func TestToSExpressionWithResourceProperties(t *testing.T) {
	request := EvaluationRequest{
		Subject: Subject{
			Type: "user",
			ID:   "alice",
		},
		Resource: Resource{
			Type: "file",
			ID:   "doc.pdf",
			Properties: map[string]interface{}{
				"owner": "bob",
				"size":  float64(1024),
			},
		},
		Action: Action{
			Name: "read",
		},
	}

	got, err := request.ToSExpression()
	if err != nil {
		t.Fatalf("ToSExpression() error = %v", err)
	}

	// Verify it's a valid S-expression
	parser := sexp.NewParser(got.String())
	parsed, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse generated S-expression: %v", err)
	}
	if parsed == nil {
		t.Error("Parsed S-expression is nil")
	}
}

func TestToSExpressionWithContextProperties(t *testing.T) {
	request := EvaluationRequest{
		Subject: Subject{
			Type: "user",
			ID:   "alice",
		},
		Resource: Resource{
			Type: "file",
			ID:   "doc.pdf",
		},
		Action: Action{
			Name: "read",
		},
		Context: Context{
			"ip":        "192.168.1.1",
			"timestamp": float64(1234567890),
		},
	}

	got, err := request.ToSExpression()
	if err != nil {
		t.Fatalf("ToSExpression() error = %v", err)
	}

	// Verify it's a valid S-expression
	parser := sexp.NewParser(got.String())
	parsed, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse generated S-expression: %v", err)
	}
	if parsed == nil {
		t.Error("Parsed S-expression is nil")
	}
}
