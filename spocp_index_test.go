package spocp

import (
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestNewEngineWithIndexing(t *testing.T) {
	// Test with indexing enabled
	indexedEngine := NewEngineWithIndexing(true)
	if !indexedEngine.indexEnabled {
		t.Error("Expected indexing to be enabled")
	}

	// Test with indexing disabled
	nonIndexedEngine := NewEngineWithIndexing(false)
	if nonIndexedEngine.indexEnabled {
		t.Error("Expected indexing to be disabled")
	}
}

func TestQueryByString(t *testing.T) {
	engine := NewEngine()

	// Add a simple rule that matches the query
	// Rule: (read file) should match query: (read file)
	err := engine.AddRule("(4:read4:file)")
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Test matching query - exact match
	allowed, err := engine.Query("(4:read4:file)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !allowed {
		t.Error("Expected query to be allowed")
	}

	// Test non-matching query
	allowed, err = engine.Query("(5:write4:file)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if allowed {
		t.Error("Expected query to be denied")
	}

	// Test invalid query
	_, err = engine.Query("invalid")
	if err == nil {
		t.Error("Expected error for invalid query")
	}
}

func TestQueryLinear(t *testing.T) {
	// Create engine with indexing disabled
	engine := NewEngineWithIndexing(false)

	// Add rules - query must be <= rule, so rule needs to be more general
	engine.AddRuleElement(sexp.NewList("read"))
	engine.AddRuleElement(sexp.NewList("write", sexp.NewAtom("admin")))

	// Test queries
	query1 := sexp.NewList("read", sexp.NewAtom("user"))
	if !engine.QueryElement(query1) {
		t.Error("Expected read query to match")
	}

	query2 := sexp.NewList("write", sexp.NewAtom("admin"))
	if !engine.QueryElement(query2) {
		t.Error("Expected write query to match")
	}

	query3 := sexp.NewList("delete", sexp.NewAtom("user"))
	if engine.QueryElement(query3) {
		t.Error("Expected delete query to not match")
	}
}

func TestQueryIndexedWithAtoms(t *testing.T) {
	engine := NewEngine() // indexed by default

	// Add atom rules
	engine.AddRuleElement(sexp.NewAtom("allow"))
	engine.AddRuleElement(sexp.NewAtom("permit"))

	// Query with atom
	if !engine.QueryElement(sexp.NewAtom("allow")) {
		t.Error("Expected atom query to match")
	}

	if engine.QueryElement(sexp.NewAtom("deny")) {
		t.Error("Expected non-matching atom query to fail")
	}
}

func TestGetIndexStats(t *testing.T) {
	engine := NewEngine()

	// Add rules with different tags
	engine.AddRuleElement(sexp.NewList("read", sexp.NewAtom("file1")))
	engine.AddRuleElement(sexp.NewList("read", sexp.NewAtom("file2")))
	engine.AddRuleElement(sexp.NewList("write", sexp.NewAtom("file3")))
	engine.AddRuleElement(sexp.NewAtom("atom1"))

	stats := engine.GetIndexStats()

	// Check stats
	if totalRules, ok := stats["total_rules"].(int); !ok || totalRules != 4 {
		t.Errorf("Expected total_rules to be 4, got %v", stats["total_rules"])
	}

	if uniqueTags, ok := stats["unique_tags"].(int); !ok || uniqueTags != 2 {
		t.Errorf("Expected unique_tags to be 2, got %v", stats["unique_tags"])
	}

	if atomRules, ok := stats["atom_rules"].(int); !ok || atomRules != 1 {
		t.Errorf("Expected atom_rules to be 1, got %v", stats["atom_rules"])
	}

	if avgRulesPerTag, ok := stats["avg_rules_per_tag"].(float64); !ok || avgRulesPerTag != 1.5 {
		t.Errorf("Expected avg_rules_per_tag to be 1.5, got %v", stats["avg_rules_per_tag"])
	}

	if mostCommonTag, ok := stats["most_common_tag"].(string); !ok || mostCommonTag != "read" {
		t.Errorf("Expected most_common_tag to be 'read', got %v", stats["most_common_tag"])
	}

	if mostCommonTagCount, ok := stats["most_common_tag_count"].(int); !ok || mostCommonTagCount != 2 {
		t.Errorf("Expected most_common_tag_count to be 2, got %v", stats["most_common_tag_count"])
	}
}

func TestGetIndexStatsNonIndexed(t *testing.T) {
	engine := NewEngineWithIndexing(false)
	engine.AddRuleElement(sexp.NewList("read", sexp.NewAtom("file")))

	stats := engine.GetIndexStats()

	if indexEnabled, ok := stats["index_enabled"].(bool); !ok || indexEnabled {
		t.Error("Expected index_enabled to be false")
	}

	// Should not have index-specific stats
	if _, exists := stats["unique_tags"]; exists {
		t.Error("Non-indexed engine should not have unique_tags stat")
	}
}

func TestFindMatchingRulesIndexed(t *testing.T) {
	engine := NewEngine()

	// Add multiple rules - rules need to be more general than queries
	engine.AddRuleElement(sexp.NewList("read"))
	engine.AddRuleElement(sexp.NewList("write"))

	// Find matches for read query
	matches, err := engine.FindMatchingRules("(4:read4:file)")
	if err != nil {
		t.Fatalf("FindMatchingRules failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}
}

func TestFindMatchingRulesNonIndexed(t *testing.T) {
	engine := NewEngineWithIndexing(false)

	// Add multiple rules
	engine.AddRuleElement(sexp.NewList("read"))
	engine.AddRuleElement(sexp.NewList("write"))

	// Find matches
	matches, err := engine.FindMatchingRules("(4:read4:file)")
	if err != nil {
		t.Fatalf("FindMatchingRules failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}
}

func TestFindMatchingRulesInvalidQuery(t *testing.T) {
	engine := NewEngine()

	_, err := engine.FindMatchingRules("invalid")
	if err == nil {
		t.Error("Expected error for invalid query")
	}
}

func TestClearIndexed(t *testing.T) {
	engine := NewEngine()

	// Add rules
	engine.AddRuleElement(sexp.NewList("read", sexp.NewAtom("file")))
	engine.AddRuleElement(sexp.NewAtom("atom"))

	if engine.RuleCount() != 2 {
		t.Error("Expected 2 rules before clear")
	}

	// Clear
	engine.Clear()

	if engine.RuleCount() != 0 {
		t.Error("Expected 0 rules after clear")
	}

	// Check index is cleared
	stats := engine.GetIndexStats()
	if uniqueTags, ok := stats["unique_tags"].(int); ok && uniqueTags != 0 {
		t.Error("Expected index to be cleared")
	}
}

func TestAddRuleError(t *testing.T) {
	engine := NewEngine()

	// Try to add invalid rule
	err := engine.AddRule("invalid s-expression")
	if err == nil {
		t.Error("Expected error when adding invalid rule")
	}
}
