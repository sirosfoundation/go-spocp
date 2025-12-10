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
