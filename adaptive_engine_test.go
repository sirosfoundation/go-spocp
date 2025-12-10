package spocp

import (
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestNew(t *testing.T) {
	// Test that New() returns an AdaptiveEngine
	engine := New()
	if engine == nil {
		t.Fatal("New() returned nil")
	}

	// Should start with no rules
	if engine.RuleCount() != 0 {
		t.Errorf("Expected 0 rules, got %d", engine.RuleCount())
	}

	// Should start with indexing disabled
	stats := engine.Stats()
	if stats.IndexingEnabled {
		t.Error("Expected indexing to be disabled initially")
	}

	// Should work like a normal engine
	err := engine.AddRule("(4:test)")
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	allowed, err := engine.Query("(4:test)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !allowed {
		t.Error("Expected query to be allowed")
	}
}

func TestAdaptiveEngine_SmallRuleset(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add only 10 rules - should NOT enable indexing
	for i := 0; i < 10; i++ {
		err := engine.AddRule("(4:test)")
		if err != nil {
			t.Fatalf("Failed to add rule: %v", err)
		}
	}

	stats := engine.Stats()
	if stats.IndexingEnabled {
		t.Error("Expected indexing to be disabled for small ruleset")
	}
	if stats.TotalRules != 10 {
		t.Errorf("Expected 10 rules, got %d", stats.TotalRules)
	}
}

func TestAdaptiveEngine_LargeRulesetWithDiverseTags(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add 100 rules with 20 different tags - should enable indexing
	tags := []string{"read", "write", "delete", "update", "create",
		"list", "get", "post", "put", "patch",
		"admin", "user", "guest", "owner", "member",
		"view", "edit", "approve", "reject", "submit"}

	for i := 0; i < 100; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("resource"))
		engine.AddRuleElement(rule)
	}

	stats := engine.Stats()
	if !stats.IndexingEnabled {
		t.Error("Expected indexing to be enabled for large ruleset with diverse tags")
	}
	if stats.TotalRules != 100 {
		t.Errorf("Expected 100 rules, got %d", stats.TotalRules)
	}
	if stats.UniqueTags != 20 {
		t.Errorf("Expected 20 unique tags, got %d", stats.UniqueTags)
	}
	if stats.AvgTagFanout != 5.0 {
		t.Errorf("Expected average fanout of 5.0, got %.2f", stats.AvgTagFanout)
	}
}

func TestAdaptiveEngine_LargeRulesetFewTags(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add 100 rules with only 2 tags - should NOT enable indexing
	// (not enough tag diversity)
	for i := 0; i < 100; i++ {
		tag := "read"
		if i%2 == 0 {
			tag = "write"
		}
		rule := sexp.NewList(tag, sexp.NewAtom("resource"))
		engine.AddRuleElement(rule)
	}

	stats := engine.Stats()
	if stats.IndexingEnabled {
		t.Error("Expected indexing to be disabled - not enough unique tags")
	}
	if stats.UniqueTags != 2 {
		t.Errorf("Expected 2 unique tags, got %d", stats.UniqueTags)
	}
}

func TestAdaptiveEngine_LargeRulesetHighFanout(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add 600 rules with 5 tags - high fanout (120 rules per tag)
	// Should NOT enable indexing (fanout too high)
	tags := []string{"http", "file", "db", "api", "service"}

	for i := 0; i < 600; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("resource"))
		engine.AddRuleElement(rule)
	}

	stats := engine.Stats()
	if stats.IndexingEnabled {
		t.Error("Expected indexing to be disabled - average fanout too high")
	}
	if stats.AvgTagFanout != 120.0 {
		t.Errorf("Expected average fanout of 120.0, got %.2f", stats.AvgTagFanout)
	}
}

func TestAdaptiveEngine_Query(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add enough rules to trigger indexing
	for i := 0; i < 60; i++ {
		tag := "action"
		if i%2 == 0 {
			tag = "read"
		} else if i%3 == 0 {
			tag = "write"
		}
		rule := sexp.NewList(tag, sexp.NewAtom("file"))
		engine.AddRuleElement(rule)
	}

	// Query should work regardless of indexing state
	allowed, err := engine.Query("(4:read4:file)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !allowed {
		t.Error("Expected query to be allowed")
	}

	// Query that shouldn't match
	allowed, err = engine.Query("(6:delete4:file)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if allowed {
		t.Error("Expected query to be denied")
	}
}

func TestAdaptiveEngine_QueryElement(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add rules
	rule1 := sexp.NewList("http", sexp.NewAtom("GET"))
	rule2 := sexp.NewList("http", sexp.NewAtom("POST"))
	engine.AddRuleElement(rule1)
	engine.AddRuleElement(rule2)

	// Test query
	query := sexp.NewList("http", sexp.NewAtom("GET"))
	if !engine.QueryElement(query) {
		t.Error("Expected query to be allowed")
	}

	query2 := sexp.NewList("http", sexp.NewAtom("DELETE"))
	if engine.QueryElement(query2) {
		t.Error("Expected query to be denied")
	}
}

func TestAdaptiveEngine_FindMatchingRules(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add diverse rules to trigger indexing
	for i := 0; i < 60; i++ {
		tags := []string{"read", "write", "execute", "delete", "update", "list"}
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("file"))
		engine.AddRuleElement(rule)
	}

	// Find matching rules
	matches, err := engine.FindMatchingRules("(4:read4:file)")
	if err != nil {
		t.Fatalf("FindMatchingRules failed: %v", err)
	}

	if len(matches) != 10 {
		t.Errorf("Expected 10 matching rules, got %d", len(matches))
	}
}

func TestAdaptiveEngine_Clear(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add rules
	for i := 0; i < 60; i++ {
		engine.AddRule("(4:test)")
	}

	if engine.RuleCount() != 60 {
		t.Errorf("Expected 60 rules, got %d", engine.RuleCount())
	}

	// Clear
	engine.Clear()

	if engine.RuleCount() != 0 {
		t.Errorf("Expected 0 rules after clear, got %d", engine.RuleCount())
	}

	stats := engine.Stats()
	if stats.TotalRules != 0 {
		t.Errorf("Expected stats to be reset, got %d rules", stats.TotalRules)
	}
	if stats.IndexingEnabled {
		t.Error("Expected indexing to be disabled after clear")
	}
}

func TestAdaptiveEngine_ForceIndexing(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add only 10 rules - normally wouldn't enable indexing
	for i := 0; i < 10; i++ {
		engine.AddRule("(4:test)")
	}

	stats := engine.Stats()
	if stats.IndexingEnabled {
		t.Error("Expected indexing to be disabled initially")
	}

	// Force enable indexing
	engine.ForceIndexing(true)

	stats = engine.Stats()
	if !stats.IndexingEnabled {
		t.Error("Expected indexing to be enabled after force")
	}

	// Query should still work
	allowed, err := engine.Query("(4:test)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !allowed {
		t.Error("Expected query to be allowed")
	}
}

func TestAdaptiveEngine_GetIndexStats(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add rules
	for i := 0; i < 60; i++ {
		tags := []string{"read", "write", "execute", "delete", "update"}
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("file"))
		engine.AddRuleElement(rule)
	}

	stats := engine.GetIndexStats()

	// Check adaptive-specific stats
	if stats["adaptive_total_rules"] != 60 {
		t.Errorf("Expected 60 total rules, got %v", stats["adaptive_total_rules"])
	}

	if stats["adaptive_unique_tags"] != 5 {
		t.Errorf("Expected 5 unique tags, got %v", stats["adaptive_unique_tags"])
	}

	if stats["adaptive_avg_fanout"] != 12.0 {
		t.Errorf("Expected avg fanout of 12.0, got %v", stats["adaptive_avg_fanout"])
	}

	if enabled, ok := stats["adaptive_indexing_enabled"].(bool); !ok || !enabled {
		t.Error("Expected indexing to be enabled")
	}
}

func TestAdaptiveEngine_MixedRules(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Add mix of list and atom rules
	for i := 0; i < 40; i++ {
		rule := sexp.NewList("action", sexp.NewAtom("read"))
		engine.AddRuleElement(rule)
	}

	for i := 0; i < 20; i++ {
		rule := sexp.NewAtom("wildcard")
		engine.AddRuleElement(rule)
	}

	stats := engine.Stats()
	if stats.ListRules != 40 {
		t.Errorf("Expected 40 list rules, got %d", stats.ListRules)
	}
	if stats.AtomRules != 20 {
		t.Errorf("Expected 20 atom rules, got %d", stats.AtomRules)
	}
}

func TestAdaptiveEngine_GradualIndexingTransition(t *testing.T) {
	engine := NewAdaptiveEngine()

	// Start adding rules and watch indexing get enabled
	tags := []string{"read", "write", "delete", "update", "create", "list"}

	// Add 30 rules - should not enable indexing yet
	for i := 0; i < 30; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("resource"))
		engine.AddRuleElement(rule)
	}

	if engine.Stats().IndexingEnabled {
		t.Error("Expected indexing disabled with 30 rules")
	}

	// Add 30 more rules - should enable indexing (60 total, 6 tags)
	for i := 0; i < 30; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("resource"))
		engine.AddRuleElement(rule)
	}

	if !engine.Stats().IndexingEnabled {
		t.Error("Expected indexing enabled with 60 rules and 6 tags")
	}

	// Verify queries work correctly
	allowed, err := engine.Query("(4:read8:resource)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !allowed {
		t.Error("Expected query to be allowed")
	}
}

func TestAdaptiveEngine_EdgeCaseThresholds(t *testing.T) {
	// Test exact threshold boundaries
	engine := NewAdaptiveEngine()

	tags := []string{"a", "b", "c", "d", "e"}

	// Exactly minRulesForIndexing (50) with exactly minTagCountForIndexing (5)
	for i := 0; i < 50; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag, sexp.NewAtom("x"))
		engine.AddRuleElement(rule)
	}

	stats := engine.Stats()
	// Should enable: 50 rules >= 50, 5 tags >= 5, avg fanout 10 <= 100
	if !stats.IndexingEnabled {
		t.Error("Expected indexing enabled at exact threshold")
	}
}
