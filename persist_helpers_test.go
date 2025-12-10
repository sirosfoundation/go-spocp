package spocp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/persist"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestEngineLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules.txt")

	// Create test file
	content := `(4:http3:GET)
(4:http4:POST)
(5:admin)`

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load into engine
	engine := NewEngine()
	if err := engine.LoadRulesFromFile(filename); err != nil {
		t.Fatalf("LoadRulesFromFile failed: %v", err)
	}

	// Verify rules were loaded
	if engine.RuleCount() != 3 {
		t.Errorf("Expected 3 rules, got %d", engine.RuleCount())
	}

	// Test query
	allowed, err := engine.Query("(4:http3:GET)")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !allowed {
		t.Error("Expected query to be allowed")
	}
}

func TestEngineSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "saved.txt")

	// Create engine with rules
	engine := NewEngine()
	engine.AddRule("(4:http3:GET)")
	engine.AddRule("(4:http4:POST)")

	// Save to file
	if err := engine.SaveRulesToFile(filename, persist.FormatCanonical); err != nil {
		t.Fatalf("SaveRulesToFile failed: %v", err)
	}

	// Load and verify
	loaded, err := persist.LoadFileToSlice(filename)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("Expected 2 rules in saved file, got %d", len(loaded))
	}
}

func TestEngineExportImport(t *testing.T) {
	// Create engine with rules
	engine1 := NewEngine()
	engine1.AddRule("(4:http3:GET)")
	engine1.AddRule("(4:http4:POST)")
	engine1.AddRule("(5:admin)")

	// Export rules
	exported := engine1.ExportRules()
	if len(exported) != 3 {
		t.Errorf("Expected 3 exported rules, got %d", len(exported))
	}

	// Import to new engine
	engine2 := NewEngine()
	engine2.ImportRules(exported)

	if engine2.RuleCount() != 3 {
		t.Errorf("Expected 3 imported rules, got %d", engine2.RuleCount())
	}

	// Verify rules work the same
	query := "(4:http3:GET)"
	allowed1, _ := engine1.Query(query)
	allowed2, _ := engine2.Query(query)

	if allowed1 != allowed2 {
		t.Error("Imported engine behaves differently from original")
	}
}

func TestEngineBinarySaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules.spocp")

	// Create engine with complex rules
	engine1 := NewEngine()
	engine1.AddRule("(4:http3:GET)")
	engine1.AddRuleElement(sexp.NewList("file",
		sexp.NewList("path", sexp.NewAtom("/etc/passwd")),
		sexp.NewList("action", sexp.NewAtom("read")),
	))

	// Save in binary format
	if err := engine1.SaveRulesToFile(filename, persist.FormatBinary); err != nil {
		t.Fatalf("SaveRulesToFile failed: %v", err)
	}

	// Load into new engine
	engine2 := NewEngine()
	opts := persist.LoadOptions{Format: persist.FormatBinary}
	if err := engine2.LoadRulesFromFileWithOptions(filename, opts); err != nil {
		t.Fatalf("LoadRulesFromFileWithOptions failed: %v", err)
	}

	// Verify rule count
	if engine2.RuleCount() != 2 {
		t.Errorf("Expected 2 rules, got %d", engine2.RuleCount())
	}

	// Verify queries work
	allowed, _ := engine2.Query("(4:http3:GET)")
	if !allowed {
		t.Error("Expected query to be allowed after binary load")
	}
}

func TestAdaptiveEngineLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules.txt")

	// Create test file with enough rules to trigger indexing
	content := ""
	actions := []string{"read", "write", "execute", "delete", "list", "create"}
	for i := 0; i < 60; i++ {
		action := actions[i%len(actions)]
		// Create proper canonical S-expression
		rule := sexp.NewList(action, sexp.NewAtom("file"))
		content += rule.String() + "\n"
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load into adaptive engine
	engine := NewAdaptiveEngine()
	if err := engine.LoadRulesFromFile(filename); err != nil {
		t.Fatalf("LoadRulesFromFile failed: %v", err)
	}

	// Verify rules were loaded
	if engine.RuleCount() != 60 {
		t.Errorf("Expected 60 rules, got %d", engine.RuleCount())
	}

	// Check adaptive behavior
	stats := engine.Stats()
	t.Logf("Stats: TotalRules=%d, UniqueTags=%d, AvgFanout=%.2f, IndexingEnabled=%v",
		stats.TotalRules, stats.UniqueTags, stats.AvgTagFanout, stats.IndexingEnabled)
	if !stats.IndexingEnabled {
		t.Error("Expected indexing to be enabled for 60 rules with 6 diverse tags")
	}
}

func TestEngineLoadWithSkipInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "mixed.txt")

	// Create file with valid and invalid rules
	content := `(4:http3:GET)
invalid rule here
(4:http4:POST)
another bad one
(5:admin)`

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load with SkipInvalid
	engine := NewEngine()
	opts := persist.LoadOptions{
		SkipInvalid: true,
	}
	if err := engine.LoadRulesFromFileWithOptions(filename, opts); err != nil {
		t.Fatalf("LoadRulesFromFileWithOptions failed: %v", err)
	}

	// Should have 3 valid rules (2 invalid skipped)
	if engine.RuleCount() != 3 {
		t.Errorf("Expected 3 rules (invalid skipped), got %d", engine.RuleCount())
	}
}

func TestExportRulesReturnsCopy(t *testing.T) {
	engine := NewEngine()
	engine.AddRule("(4:http3:GET)")

	// Export rules
	exported := engine.ExportRules()

	// Modify exported slice
	exported[0] = sexp.NewAtom("modified")

	// Verify engine's rules are unchanged
	rules := engine.ExportRules()
	if rules[0].String() == "8:modified" {
		t.Error("ExportRules should return a copy, not the original slice")
	}
}
