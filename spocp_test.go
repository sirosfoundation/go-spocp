package spocp

import (
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
	"github.com/sirosfoundation/go-spocp/pkg/starform"
)

func TestEngineBasicOperations(t *testing.T) {
	engine := NewEngine()

	if engine.RuleCount() != 0 {
		t.Errorf("new engine should have 0 rules, got %d", engine.RuleCount())
	}

	// Add a simple rule
	err := engine.AddRule("(5:admin)")
	if err != nil {
		t.Fatalf("failed to add rule: %v", err)
	}

	if engine.RuleCount() != 1 {
		t.Errorf("expected 1 rule, got %d", engine.RuleCount())
	}

	engine.Clear()
	if engine.RuleCount() != 0 {
		t.Errorf("after clear, expected 0 rules, got %d", engine.RuleCount())
	}
}

func TestEngineHTTPExample(t *testing.T) {
	// Based on spec example from section 5.2
	engine := NewEngine()

	// Add rule: (http (page index.html)(action GET)(user))
	// This allows any user to GET index.html
	rule := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("GET")),
		sexp.NewList("user"),
	)
	engine.AddRuleElement(rule)

	// Query: (http (page index.html)(action GET)(user olav))
	// This should be authorized (specific user <= any user)
	query := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("GET")),
		sexp.NewList("user", sexp.NewAtom("olav")),
	)

	if !engine.QueryElement(query) {
		t.Error("Expected query to be authorized")
	}

	// Query: (http (page index.html)(action POST)(user olav))
	// This should NOT be authorized (POST != GET)
	query2 := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("POST")),
		sexp.NewList("user", sexp.NewAtom("olav")),
	)

	if engine.QueryElement(query2) {
		t.Error("Expected POST query to be denied")
	}
}

func TestEngineWithWildcard(t *testing.T) {
	engine := NewEngine()

	// Rule: (resource *)
	// This allows access to any resource
	rule := sexp.NewList("resource", &starform.Wildcard{})
	engine.AddRuleElement(rule)

	// Any resource query should match
	query := sexp.NewList("resource", sexp.NewAtom("database"))
	if !engine.QueryElement(query) {
		t.Error("Expected wildcard to match any resource")
	}

	query2 := sexp.NewList("resource", sexp.NewAtom("file"))
	if !engine.QueryElement(query2) {
		t.Error("Expected wildcard to match any resource")
	}
}

func TestEngineWithPrefix(t *testing.T) {
	engine := NewEngine()

	// Rule: (file (* prefix /etc/))
	// Allows access to files under /etc/
	rule := sexp.NewList("file", &starform.Prefix{Value: "/etc/"})
	engine.AddRuleElement(rule)

	// Query for /etc/passwd should match
	query := sexp.NewList("file", sexp.NewAtom("/etc/passwd"))
	if !engine.QueryElement(query) {
		t.Error("Expected /etc/passwd to match /etc/ prefix")
	}

	// Query for /var/log should NOT match
	query2 := sexp.NewList("file", sexp.NewAtom("/var/log"))
	if engine.QueryElement(query2) {
		t.Error("Expected /var/log to NOT match /etc/ prefix")
	}
}

func TestEngineWithSet(t *testing.T) {
	engine := NewEngine()

	// Rule: (action (* set read write))
	// Allows read or write actions
	rule := sexp.NewList("action", &starform.Set{
		Elements: []sexp.Element{
			sexp.NewAtom("read"),
			sexp.NewAtom("write"),
		},
	})
	engine.AddRuleElement(rule)

	// Query for read should match
	query := sexp.NewList("action", sexp.NewAtom("read"))
	if !engine.QueryElement(query) {
		t.Error("Expected 'read' to be in set")
	}

	// Query for write should match
	query2 := sexp.NewList("action", sexp.NewAtom("write"))
	if !engine.QueryElement(query2) {
		t.Error("Expected 'write' to be in set")
	}

	// Query for delete should NOT match
	query3 := sexp.NewList("action", sexp.NewAtom("delete"))
	if engine.QueryElement(query3) {
		t.Error("Expected 'delete' to NOT be in set")
	}
}

func TestEngineMultipleRules(t *testing.T) {
	engine := NewEngine()

	// Add multiple rules for different resources
	engine.AddRuleElement(sexp.NewList("resource", sexp.NewAtom("public")))
	engine.AddRuleElement(sexp.NewList("resource", sexp.NewAtom("shared")))

	// Query for public should match
	query1 := sexp.NewList("resource", sexp.NewAtom("public"))
	if !engine.QueryElement(query1) {
		t.Error("Expected 'public' to be authorized")
	}

	// Query for shared should match
	query2 := sexp.NewList("resource", sexp.NewAtom("shared"))
	if !engine.QueryElement(query2) {
		t.Error("Expected 'shared' to be authorized")
	}

	// Query for private should NOT match
	query3 := sexp.NewList("resource", sexp.NewAtom("private"))
	if engine.QueryElement(query3) {
		t.Error("Expected 'private' to be denied")
	}
}

func TestEngineFindMatchingRules(t *testing.T) {
	engine := NewEngine()

	// Add several rules
	rule1 := sexp.NewList("resource", sexp.NewAtom("file"))
	rule2 := sexp.NewList("resource", &starform.Wildcard{})
	rule3 := sexp.NewList("other", sexp.NewAtom("data"))

	engine.AddRuleElement(rule1)
	engine.AddRuleElement(rule2)
	engine.AddRuleElement(rule3)

	// Query (resource file) should match rule1 and rule2
	query := sexp.NewList("resource", sexp.NewAtom("file"))
	parser := sexp.NewParser(query.String())
	queryElem, _ := parser.Parse()

	matches, err := engine.FindMatchingRules(queryElem.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matching rules, got %d", len(matches))
	}
}

func TestEngineRangeExample(t *testing.T) {
	// From spec: (worktime (* range time ge 08:00:00 le 17:00:00))
	engine := NewEngine()

	// Add rule for work hours
	rule := sexp.NewList("worktime", &starform.Range{
		RangeType: starform.RangeTime,
		LowerBound: &starform.RangeBound{
			Op:    starform.OpGE,
			Value: "08:00:00",
		},
		UpperBound: &starform.RangeBound{
			Op:    starform.OpLE,
			Value: "17:00:00",
		},
	})
	engine.AddRuleElement(rule)

	// Query during work hours
	query := sexp.NewList("worktime", sexp.NewAtom("12:00:00"))
	if !engine.QueryElement(query) {
		t.Error("Expected 12:00:00 to be within work hours")
	}

	// Query outside work hours
	query2 := sexp.NewList("worktime", sexp.NewAtom("20:00:00"))
	if engine.QueryElement(query2) {
		t.Error("Expected 20:00:00 to be outside work hours")
	}
}
