// Package authzen implements the AuthZen Authorization API 1.0 specification.package authzen

// See: https://openid.net/specs/authorization-api-1_0-01.html
package authzen

import (
	"encoding/json"
	"fmt"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// Subject represents the user or principal in an authorization request.
type Subject struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Resource represents the target of an access request.
type Resource struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Action represents the operation being performed.
type Action struct {
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Context represents environmental/contextual attributes.
type Context map[string]interface{}

// EvaluationRequest is the AuthZen access evaluation request.
type EvaluationRequest struct {
	Subject  Subject  `json:"subject"`
	Resource Resource `json:"resource"`
	Action   Action   `json:"action"`
	Context  Context  `json:"context,omitempty"`
}

// EvaluationResponse is the AuthZen access evaluation response.
type EvaluationResponse struct {
	Decision bool                   `json:"decision"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// ToSExpression converts an AuthZen evaluation request to a SPOCP S-expression query.
//
// The conversion follows this mapping:
//
//	AuthZen Request:
//	  {
//	    "subject": {"type": "user", "id": "alice@acmecorp.com"},
//	    "resource": {"type": "account", "id": "123"},
//	    "action": {"name": "can_read", "properties": {"method": "GET"}},
//	    "context": {"ip": "192.168.1.1"}
//	  }
//
//	SPOCP S-expression:
//	  (account
//	    (id 123)
//	    (action can_read (method GET))
//	    (subject (type user) (id alice@acmecorp.com))
//	    (context (ip 192.168.1.1)))
//
// Structure rules:
//   - Resource type becomes the outer tag
//   - Resource properties become top-level elements
//   - Action is wrapped in (action ...) with name as first atom
//   - Subject is wrapped in (subject ...) with type and id if present
//   - Context is wrapped in (context ...) and only included if non-empty
//   - Properties are converted recursively (strings, bools, numbers, arrays, nested objects)
//
// Returns an error if any property value cannot be converted to an S-expression.
func (r *EvaluationRequest) ToSExpression() (sexp.Element, error) {
	// Build the query starting with resource type as the root
	elements := []sexp.Element{sexp.NewAtom(r.Resource.Type)}

	// Add resource ID if present
	if r.Resource.ID != "" {
		elements = append(elements, sexp.NewList("id", sexp.NewAtom(r.Resource.ID)))
	}

	// Add resource properties
	for key, value := range r.Resource.Properties {
		elem, err := propertyToSExp(key, value)
		if err != nil {
			return nil, fmt.Errorf("resource property %s: %w", key, err)
		}
		elements = append(elements, elem)
	}

	// Add action
	actionElements := []sexp.Element{sexp.NewAtom(r.Action.Name)}
	for key, value := range r.Action.Properties {
		elem, err := propertyToSExp(key, value)
		if err != nil {
			return nil, fmt.Errorf("action property %s: %w", key, err)
		}
		actionElements = append(actionElements, elem)
	}
	elements = append(elements, buildList("action", actionElements))

	// Add subject
	var subjectElements []sexp.Element
	if r.Subject.Type != "" {
		subjectElements = append(subjectElements, sexp.NewList("type", sexp.NewAtom(r.Subject.Type)))
	}
	if r.Subject.ID != "" {
		subjectElements = append(subjectElements, sexp.NewList("id", sexp.NewAtom(r.Subject.ID)))
	}
	for key, value := range r.Subject.Properties {
		elem, err := propertyToSExp(key, value)
		if err != nil {
			return nil, fmt.Errorf("subject property %s: %w", key, err)
		}
		subjectElements = append(subjectElements, elem)
	}
	elements = append(elements, buildList("subject", subjectElements))

	// Add context if present
	if len(r.Context) > 0 {
		var contextElements []sexp.Element
		for key, value := range r.Context {
			elem, err := propertyToSExp(key, value)
			if err != nil {
				return nil, fmt.Errorf("context property %s: %w", key, err)
			}
			contextElements = append(contextElements, elem)
		}
		elements = append(elements, buildList("context", contextElements))
	}

	return buildList(r.Resource.Type, elements[1:]), nil
}

// buildList creates a List from a tag and slice of elements.
//
// This is a helper to work around sexp.NewList requiring variadic parameters
// rather than a slice.
//
// Example:
//
//	elements := []sexp.Element{sexp.NewAtom("alice"), sexp.NewAtom("bob")}
//	list := buildList("users", elements)
//	// Result: (users alice bob)
func buildList(tag string, elements []sexp.Element) *sexp.List {
	return sexp.NewList(tag, elements...)
}

// propertyToSExp converts a property key-value pair to an S-expression.
//
// Conversion rules:
//   - string: (key "value")
//   - bool: (key true) or (key false)
//   - number: (key 123) or (key 45.67)
//   - array: (key (item1) (item2) ...)
//   - object: (key (subkey1 value1) (subkey2 value2) ...)
//
// Examples:
//
//	propertyToSExp("name", "alice")           → (name alice)
//	propertyToSExp("active", true)            → (active true)
//	propertyToSExp("count", 42)               → (count 42)
//	propertyToSExp("tags", []any{"foo","bar"}) → (tags (foo) (bar))
//	propertyToSExp("meta", map[string]any{"key": "val"}) → (meta (key val))
//
// Returns an error if the value type is not supported.
func propertyToSExp(key string, value interface{}) (sexp.Element, error) {
	switch v := value.(type) {
	case string:
		return sexp.NewList(key, sexp.NewAtom(v)), nil
	case bool:
		return sexp.NewList(key, sexp.NewAtom(fmt.Sprintf("%t", v))), nil
	case float64:
		return sexp.NewList(key, sexp.NewAtom(fmt.Sprintf("%g", v))), nil
	case int:
		return sexp.NewList(key, sexp.NewAtom(fmt.Sprintf("%d", v))), nil
	case []interface{}:
		// Array of values - create nested list
		var elements []sexp.Element
		for _, item := range v {
			itemStr, err := valueToString(item)
			if err != nil {
				return nil, err
			}
			elements = append(elements, sexp.NewAtom(itemStr))
		}
		return buildList(key, elements), nil
	case map[string]interface{}:
		// Nested object - create nested list
		var elements []sexp.Element
		for nestedKey, nestedValue := range v {
			elem, err := propertyToSExp(nestedKey, nestedValue)
			if err != nil {
				return nil, err
			}
			elements = append(elements, elem)
		}
		return buildList(key, elements), nil
	default:
		// Fallback to JSON encoding for complex types
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("cannot convert value to s-expression: %w", err)
		}
		return sexp.NewList(key, sexp.NewAtom(string(data))), nil
	}
}

// valueToString converts a value to its string representation.
func valueToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case float64:
		return fmt.Sprintf("%g", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	default:
		// Fallback to JSON encoding
		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("cannot convert value to string: %w", err)
		}
		return string(data), nil
	}
}
