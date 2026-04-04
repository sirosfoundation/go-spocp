package persist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
	"github.com/sirosfoundation/go-spocp/pkg/starform"
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
	if err := SaveFile(filename, rules, FormatBinary); err != nil {
		b.Fatalf("SaveFile (binary) failed: %v", err)
	}

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
		// Regression: nested starform was previously flattened into atoms at
		// the wrong nesting level by the broken advancedToCanonical pipeline.
		{
			name:     "nested starform range (regression)",
			advanced: "(outer (inner (* range numeric ge 080)))",
			want:     "(5:outer(5:inner(1:*5:range7:numeric2:ge3:080)))",
		},
		{
			name:     "multiple nested starforms (facetec-scan pattern)",
			advanced: "(facetec-scan (liveness-score (* range numeric ge 080)) (face-match-level (* range numeric ge 06)) (doc-type passport) (mrz-verified true))",
			want:     "(12:facetec-scan(14:liveness-score(1:*5:range7:numeric2:ge3:080))(16:face-match-level(1:*5:range7:numeric2:ge2:06))(8:doc-type8:passport)(12:mrz-verified4:true))",
		},
		{
			name:     "nested wildcard starform",
			advanced: "(outer (inner (*)))",
			want:     "(5:outer(5:inner(1:*)))",
		},
		{
			name:     "flat starform range",
			advanced: "(resource (* range numeric ge 010))",
			want:     "(8:resource(1:*5:range7:numeric2:ge3:010))",
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

// TestParseAdvanced_NestedStarForms is a regression test for the bug where
// parseAdvanced (and advancedToCanonical) would flatten nested sub-expressions
// containing *-forms into atoms at the wrong depth.
//
// Before the fix, `(outer (inner (* range numeric ge 080)))` produced a list
// whose inner element was NOT a sub-list but a series of sibling atoms, causing
// range predicates in policy rules to be silently ignored during evaluation.
func TestParseAdvanced_NestedStarForms(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		checkInner  func(t *testing.T, elem sexp.Element)
	}{
		{
			name:  "single level nesting with range",
			input: "(outer (* range numeric ge 080))",
			checkInner: func(t *testing.T, elem sexp.Element) {
				list, ok := elem.(*sexp.List)
				if !ok {
					t.Fatalf("expected *sexp.List, got %T", elem)
				}
				if list.Tag != "outer" {
					t.Fatalf("expected tag 'outer', got %q", list.Tag)
				}
				if len(list.Elements) != 1 {
					t.Fatalf("expected 1 element, got %d", len(list.Elements))
				}
				r, ok := list.Elements[0].(*starform.Range)
				if !ok {
					t.Fatalf("expected *starform.Range as child, got %T", list.Elements[0])
				}
				if r.RangeType != starform.RangeNumeric {
					t.Errorf("expected RangeNumeric, got %v", r.RangeType)
				}
				if r.LowerBound == nil || r.LowerBound.Op != starform.OpGE || r.LowerBound.Value != "080" {
					t.Errorf("unexpected LowerBound: %+v", r.LowerBound)
				}
			},
		},
		{
			// Regression: this exact pattern was broken before the fix.
			name:  "two levels of nesting with range",
			input: "(outer (inner (* range numeric ge 080)))",
			checkInner: func(t *testing.T, elem sexp.Element) {
				outer, ok := elem.(*sexp.List)
				if !ok {
					t.Fatalf("expected *sexp.List for outer, got %T", elem)
				}
				if outer.Tag != "outer" {
					t.Fatalf("outer tag: got %q, want 'outer'", outer.Tag)
				}
				if len(outer.Elements) != 1 {
					t.Fatalf("outer should have 1 child, got %d; elements: %v",
						len(outer.Elements), outer.Elements)
				}
				inner, ok := outer.Elements[0].(*sexp.List)
				if !ok {
					t.Fatalf("outer.Elements[0]: expected *sexp.List (inner), got %T — "+
						"this is the regression: starform was flattened", outer.Elements[0])
				}
				if inner.Tag != "inner" {
					t.Fatalf("inner tag: got %q, want 'inner'", inner.Tag)
				}
				if len(inner.Elements) != 1 {
					t.Fatalf("inner should have 1 child, got %d", len(inner.Elements))
				}
				r, ok := inner.Elements[0].(*starform.Range)
				if !ok {
					t.Fatalf("inner.Elements[0]: expected *starform.Range, got %T", inner.Elements[0])
				}
				if r.RangeType != starform.RangeNumeric {
					t.Errorf("expected RangeNumeric, got %v", r.RangeType)
				}
				if r.LowerBound == nil || r.LowerBound.Value != "080" {
					t.Errorf("unexpected LowerBound: %+v", r.LowerBound)
				}
			},
		},
		{
			// The exact facetec-scan pattern from facetec-api/rules/default.spoc.
			name:  "facetec-scan rule with two range predicates",
			input: "(facetec-scan (liveness-score (* range numeric ge 080)) (face-match-level (* range numeric ge 06)) (doc-type passport) (mrz-verified true))",
			checkInner: func(t *testing.T, elem sexp.Element) {
				outer, ok := elem.(*sexp.List)
				if !ok {
					t.Fatalf("expected *sexp.List for facetec-scan, got %T", elem)
				}
				if outer.Tag != "facetec-scan" {
					t.Fatalf("tag: got %q, want 'facetec-scan'", outer.Tag)
				}
				if len(outer.Elements) != 4 {
					t.Fatalf("expected 4 children, got %d", len(outer.Elements))
				}

				// Check liveness-score sub-list.
				ls, ok := outer.Elements[0].(*sexp.List)
				if !ok {
					t.Fatalf("Elements[0]: expected *sexp.List (liveness-score), got %T", outer.Elements[0])
				}
				if ls.Tag != "liveness-score" {
					t.Errorf("liveness-score tag: got %q", ls.Tag)
				}
				if len(ls.Elements) != 1 {
					t.Fatalf("liveness-score should have 1 child, got %d", len(ls.Elements))
				}
				lr, ok := ls.Elements[0].(*starform.Range)
				if !ok {
					t.Fatalf("liveness-score child: expected *starform.Range, got %T", ls.Elements[0])
				}
				if lr.LowerBound == nil || lr.LowerBound.Value != "080" {
					t.Errorf("liveness-score range bound: %+v", lr.LowerBound)
				}

				// Check face-match-level sub-list.
				fm, ok := outer.Elements[1].(*sexp.List)
				if !ok {
					t.Fatalf("Elements[1]: expected *sexp.List (face-match-level), got %T", outer.Elements[1])
				}
				if fm.Tag != "face-match-level" {
					t.Errorf("face-match-level tag: got %q", fm.Tag)
				}
				fr, ok := fm.Elements[0].(*starform.Range)
				if !ok {
					t.Fatalf("face-match-level child: expected *starform.Range, got %T", fm.Elements[0])
				}
				if fr.LowerBound == nil || fr.LowerBound.Value != "06" {
					t.Errorf("face-match-level range bound: %+v", fr.LowerBound)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := parseAdvanced(tt.input)
			if err != nil {
				t.Fatalf("parseAdvanced(%q): %v", tt.input, err)
			}
			tt.checkInner(t, elem)
		})
	}
}

// TestLoadFile_AdvancedNestedRangeRules is an end-to-end regression test:
// write a .spoc file in advanced format containing rules with nested starform
// range predicates, load it, and verify the canonical representation matches.
func TestLoadFile_AdvancedNestedRangeRules(t *testing.T) {
	tmpDir := t.TempDir()
	rulesFile := filepath.Join(tmpDir, "test.spoc")

	content := `; accept passports
(facetec-scan (liveness-score (* range numeric ge 080)) (face-match-level (* range numeric ge 06)) (doc-type passport) (mrz-verified true))
; accept driving licences
(facetec-scan (liveness-score (* range numeric ge 080)) (face-match-level (* range numeric ge 06)) (doc-type dl) (barcode-verified true))
`
	if err := os.WriteFile(rulesFile, []byte(content), 0644); err != nil {
		t.Fatalf("write rules file: %v", err)
	}

	opts := LoadOptions{Format: FormatAdvanced, SkipInvalid: false, Comments: []string{"#", "//", ";"}}
	rules, err := LoadFile(rulesFile, opts)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	want := []string{
		"(12:facetec-scan(14:liveness-score(1:*5:range7:numeric2:ge3:080))(16:face-match-level(1:*5:range7:numeric2:ge2:06))(8:doc-type8:passport)(12:mrz-verified4:true))",
		"(12:facetec-scan(14:liveness-score(1:*5:range7:numeric2:ge3:080))(16:face-match-level(1:*5:range7:numeric2:ge2:06))(8:doc-type2:dl)(16:barcode-verified4:true))",
	}
	for i, rule := range rules {
		got := rule.String()
		if got != want[i] {
			t.Errorf("rule[%d]:\n  got:  %s\n  want: %s", i, got, want[i])
		}
	}
}

// TestAdvTokenize_NestedParens verifies that advTokenize preserves nested
// parenthesised groups as single tokens (this is the property that
// distinguishes it from the old broken tokenize implementation).
func TestAdvTokenize_NestedParens(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "http GET",
			want:  []string{"http", "GET"},
		},
		{
			input: "(action GET) (path /api)",
			want:  []string{"(action GET)", "(path /api)"},
		},
		{
			// The critical case: a sub-expression containing a further nested group.
			input: "(liveness-score (* range numeric ge 080)) (doc-type passport)",
			want:  []string{"(liveness-score (* range numeric ge 080))", "(doc-type passport)"},
		},
		{
			input: `"quoted string" plain`,
			want:  []string{`"quoted string"`, "plain"},
		},
	}

	for _, tt := range tests {
		got := advTokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("advTokenize(%q): got %d tokens %v, want %d tokens %v",
				tt.input, len(got), got, len(tt.want), tt.want)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("advTokenize(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
