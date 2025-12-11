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

// TestSaveLoadAdvanced tests the advanced/human-readable format
func TestSaveLoadAdvanced(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rules_advanced.txt")

	rules := []sexp.Element{
		sexp.NewList("http", sexp.NewAtom("GET")),
		sexp.NewList("file",
			sexp.NewList("path", sexp.NewAtom("test.txt")),
		),
	}

	// Save in advanced format
	if err := SaveFile(filename, rules, FormatAdvanced); err != nil {
		t.Fatalf("SaveFile (advanced) failed: %v", err)
	}

	// Verify file content is human-readable
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Advanced format should have readable structure
	if len(content) == 0 {
		t.Error("Expected non-empty file")
	}
}

// TestAdvancedToCanonical tests conversion from advanced to canonical
func TestAdvancedToCanonical(t *testing.T) {
	tests := []struct {
		name     string
		advanced string
		want     string
	}{
		{
			name:     "simple atom",
			advanced: "hello",
			want:     "5:hello",
		},
		{
			name:     "simple list",
			advanced: "(http GET)",
			want:     "(4:http3:GET)",
		},
		{
			name:     "nested list",
			advanced: "(http (action GET) (path index.html))",
			want:     "(4:http(6:action3:GET)(4:path10:index.html))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := advancedToCanonical(tt.advanced)
			if got != tt.want {
				t.Errorf("advancedToCanonical(%q) = %q, want %q", tt.advanced, got, tt.want)
			}
		})
	}
}

// TestTokenize tests the tokenizer
func TestTokenize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []string
	}{
		{
			name:   "simple atoms",
			input:  "http GET POST",
			tokens: []string{"http", "GET", "POST"},
		},
		{
			name:   "nested list",
			input:  "http (action GET)",
			tokens: []string{"http", "(action GET)"},
		},
		{
			name:   "quoted string",
			input:  `http "hello world"`,
			tokens: []string{"http", `"hello world"`},
		},
		{
			name:   "empty input",
			input:  "",
			tokens: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.tokens) {
				t.Errorf("tokenize(%q) returned %d tokens, want %d", tt.input, len(got), len(tt.tokens))
				return
			}
			for i, token := range tt.tokens {
				if got[i] != token {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], token)
				}
			}
		})
	}
}

// TestLoadFileErrors tests error handling in LoadFile
func TestLoadFileErrors(t *testing.T) {
	// Non-existent file
	_, err := LoadFile("/nonexistent/file.spoc", DefaultLoadOptions())
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestSaveFileErrors tests error handling in SaveFile
func TestSaveFileErrors(t *testing.T) {
	rules := []sexp.Element{sexp.NewAtom("test")}

	// Write to non-existent directory
	err := SaveFile("/nonexistent/dir/rules.spoc", rules, FormatCanonical)
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	// Unknown format defaults to canonical, so this should work
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.spoc")
	err = SaveFile(filename, rules, FileFormat(99))
	if err != nil {
		t.Errorf("Unknown format should default to canonical: %v", err)
	}
}

// TestSerializeDeserializeRuleAdditional tests more rule serialization cases
func TestSerializeDeserializeRuleAdditional(t *testing.T) {
	rules := []sexp.Element{
		sexp.NewAtom("simple"),
		sexp.NewList("http", sexp.NewAtom("GET")),
		sexp.NewList("complex",
			sexp.NewList("nested", sexp.NewAtom("value")),
			sexp.NewList("another", sexp.NewAtom("value2")),
		),
	}

	for _, rule := range rules {
		data, err := SerializeRule(rule)
		if err != nil {
			t.Fatalf("SerializeRule failed: %v", err)
		}

		loaded, err := DeserializeRule(data)
		if err != nil {
			t.Fatalf("DeserializeRule failed: %v", err)
		}

		if rule.String() != loaded.String() {
			t.Errorf("Round-trip mismatch: %s vs %s", rule.String(), loaded.String())
		}
	}
}

// TestDeserializeRuleErrors tests error handling in DeserializeRule
func TestDeserializeRuleErrors(t *testing.T) {
	// Empty data
	_, err := DeserializeRule([]byte{})
	if err == nil {
		t.Error("Expected error for empty data")
	}

	// Invalid magic
	_, err = DeserializeRule([]byte{0x00, 0x00, 0x00, 0x00})
	if err == nil {
		t.Error("Expected error for invalid magic")
	}
}

// TestLoadBinaryErrors tests error handling in loadBinary
func TestLoadBinaryErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Invalid magic
	badMagic := filepath.Join(tmpDir, "bad_magic.spocp")
	os.WriteFile(badMagic, []byte("WRONG"), 0644)
	_, err := LoadFile(badMagic, LoadOptions{Format: FormatBinary})
	if err == nil {
		t.Error("Expected error for bad magic")
	}

	// Truncated file
	truncated := filepath.Join(tmpDir, "truncated.spocp")
	os.WriteFile(truncated, []byte("SPOCP\x01"), 0644) // Just magic and version
	_, err = LoadFile(truncated, LoadOptions{Format: FormatBinary})
	if err == nil {
		t.Error("Expected error for truncated file")
	}
}
