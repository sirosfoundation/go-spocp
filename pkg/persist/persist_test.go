package persist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func TestSaveLoadCanonical(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules.txt")

	// Create test rules
	rules := []sexp.Element{
		sexp.NewList("http", sexp.NewAtom("GET")),
		sexp.NewList("http", sexp.NewAtom("POST")),
		sexp.NewAtom("admin"),
	}

	// Save
	if err := SaveFile(filename, rules, FormatCanonical); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Load
	loaded, err := LoadFile(filename, DefaultLoadOptions())
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// Verify
	if len(loaded) != len(rules) {
		t.Fatalf("Expected %d rules, got %d", len(rules), len(loaded))
	}

	for i, rule := range rules {
		if rule.String() != loaded[i].String() {
			t.Errorf("Rule %d mismatch: expected %s, got %s",
				i, rule.String(), loaded[i].String())
		}
	}
}

func TestSaveLoadBinary(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules.spocp")

	// Create test rules with various types
	rules := []sexp.Element{
		sexp.NewList("http", sexp.NewAtom("GET")),
		sexp.NewList("file",
			sexp.NewList("path", sexp.NewAtom("/etc/passwd")),
			sexp.NewList("action", sexp.NewAtom("read")),
		),
		sexp.NewAtom("wildcard"),
	}

	// Save in binary format
	if err := SaveFile(filename, rules, FormatBinary); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Load from binary format
	loaded, err := LoadFile(filename, LoadOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// Verify
	if len(loaded) != len(rules) {
		t.Fatalf("Expected %d rules, got %d", len(rules), len(loaded))
	}

	for i, rule := range rules {
		if rule.String() != loaded[i].String() {
			t.Errorf("Rule %d mismatch: expected %s, got %s",
				i, rule.String(), loaded[i].String())
		}
	}
}

func TestLoadWithComments(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules_with_comments.txt")

	// Create file with comments
	content := `# This is a comment
(4:http3:GET)
// Another comment
(4:http4:POST)

; Semicolon comment
(5:admin)
`

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load
	loaded, err := LoadFile(filename, DefaultLoadOptions())
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// Should have 3 rules (comments filtered out)
	if len(loaded) != 3 {
		t.Fatalf("Expected 3 rules, got %d", len(loaded))
	}
}

func TestLoadWithInvalidRules(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "invalid_rules.txt")

	// Create file with some invalid rules
	content := `(4:http3:GET)
invalid rule without proper format
(4:http4:POST)
`

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load with SkipInvalid = false (should fail)
	opts := DefaultLoadOptions()
	opts.SkipInvalid = false
	_, err := LoadFile(filename, opts)
	if err == nil {
		t.Error("Expected error when loading invalid rules with SkipInvalid=false")
	}

	// Load with SkipInvalid = true (should succeed)
	opts.SkipInvalid = true
	loaded, err := LoadFile(filename, opts)
	if err != nil {
		t.Fatalf("LoadFile failed with SkipInvalid=true: %v", err)
	}

	// Should have 2 valid rules
	if len(loaded) != 2 {
		t.Fatalf("Expected 2 rules (invalid skipped), got %d", len(loaded))
	}
}

func TestLoadWithMaxRules(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "many_rules.txt")

	// Create file with 10 rules
	content := ""
	for i := 0; i < 10; i++ {
		content += "(4:http3:GET)\n"
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load with limit of 5 rules
	opts := DefaultLoadOptions()
	opts.MaxRules = 5
	loaded, err := LoadFile(filename, opts)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// Should have only 5 rules
	if len(loaded) != 5 {
		t.Fatalf("Expected 5 rules (limited), got %d", len(loaded))
	}
}

func TestSerializeDeserializeRule(t *testing.T) {
	// Test simple rule
	rule := sexp.NewList("http", sexp.NewAtom("GET"))

	// Serialize
	data, err := SerializeRule(rule)
	if err != nil {
		t.Fatalf("SerializeRule failed: %v", err)
	}

	// Deserialize
	restored, err := DeserializeRule(data)
	if err != nil {
		t.Fatalf("DeserializeRule failed: %v", err)
	}

	// Verify
	if rule.String() != restored.String() {
		t.Errorf("Rule mismatch: expected %s, got %s",
			rule.String(), restored.String())
	}
}

func TestBinaryFormatVersion(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules.spocp")

	rules := []sexp.Element{
		sexp.NewAtom("test"),
	}

	// Save
	if err := SaveFile(filename, rules, FormatBinary); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Read raw bytes to verify format
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Check magic number
	if string(data[0:5]) != "SPOCP" {
		t.Errorf("Invalid magic number: %s", string(data[0:5]))
	}

	// Check version
	if data[5] != 1 {
		t.Errorf("Invalid version: %d", data[5])
	}
}

func TestEmptyRuleset(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "empty.txt")

	rules := []sexp.Element{}

	// Save empty ruleset
	if err := SaveFile(filename, rules, FormatCanonical); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Load empty ruleset
	loaded, err := LoadFile(filename, DefaultLoadOptions())
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("Expected empty ruleset, got %d rules", len(loaded))
	}
}

func TestBinaryRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "roundtrip.spocp")

	// Create complex rules
	rules := []sexp.Element{
		sexp.NewList("http",
			sexp.NewList("method", sexp.NewAtom("GET")),
			sexp.NewList("path", sexp.NewAtom("/api/v1/users")),
		),
		sexp.NewList("file",
			sexp.NewList("path", sexp.NewAtom("/etc/passwd")),
			sexp.NewList("action", sexp.NewAtom("read")),
			sexp.NewList("user", sexp.NewAtom("admin")),
		),
		sexp.NewAtom("simple-atom"),
	}

	// Save
	if err := SaveFile(filename, rules, FormatBinary); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Load
	loaded, err := LoadFile(filename, LoadOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	// Verify all rules match exactly
	if len(loaded) != len(rules) {
		t.Fatalf("Expected %d rules, got %d", len(rules), len(loaded))
	}

	for i, rule := range rules {
		if rule.String() != loaded[i].String() {
			t.Errorf("Rule %d mismatch:\n  expected: %s\n  got:      %s",
				i, rule.String(), loaded[i].String())
		}
	}
}

func TestLoadFileToSlice(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "simple.txt")

	// Create simple file
	content := "(4:http3:GET)\n(4:http4:POST)\n"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load using convenience function
	rules, err := LoadFileToSlice(filename)
	if err != nil {
		t.Fatalf("LoadFileToSlice failed: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("Expected 2 rules, got %d", len(rules))
	}
}

func BenchmarkSaveCanonical(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.txt")

	// Create test rules
	rules := make([]sexp.Element, 100)
	for i := 0; i < 100; i++ {
		rules[i] = sexp.NewList("http", sexp.NewAtom("GET"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SaveFile(filename, rules, FormatCanonical)
	}
}

func BenchmarkSaveBinary(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.spocp")

	rules := make([]sexp.Element, 100)
	for i := 0; i < 100; i++ {
		rules[i] = sexp.NewList("http", sexp.NewAtom("GET"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SaveFile(filename, rules, FormatBinary)
	}
}

func BenchmarkLoadCanonical(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.txt")

	rules := make([]sexp.Element, 100)
	for i := 0; i < 100; i++ {
		rules[i] = sexp.NewList("http", sexp.NewAtom("GET"))
	}
	SaveFile(filename, rules, FormatCanonical)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadFile(filename, DefaultLoadOptions())
	}
}

func BenchmarkLoadBinary(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.spocp")

	rules := make([]sexp.Element, 100)
	for i := 0; i < 100; i++ {
		rules[i] = sexp.NewList("http", sexp.NewAtom("GET"))
	}
	SaveFile(filename, rules, FormatBinary)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadFile(filename, LoadOptions{Format: FormatBinary})
	}
}
