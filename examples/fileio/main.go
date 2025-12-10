package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/persist"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func main() {
	fmt.Println("=== SPOCP File Loading and Serialization Demo ===")
	fmt.Println()

	// Create a temporary directory for our examples
	tmpDir, err := os.MkdirTemp("", "spocp-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Example 1: Save and load rules in canonical format
	example1(tmpDir)

	// Example 2: Binary serialization for efficiency
	example2(tmpDir)

	// Example 3: Loading with comments and filtering
	example3(tmpDir)

	// Example 4: Direct engine loading
	example4(tmpDir)

	// Example 5: Performance comparison
	example5(tmpDir)
}

func example1(tmpDir string) {
	fmt.Println("Example 1: Canonical Format Save/Load")
	fmt.Println("--------------------------------------")

	filename := filepath.Join(tmpDir, "rules.txt")

	// Create some rules
	rules := []sexp.Element{
		sexp.NewList("http", sexp.NewAtom("GET")),
		sexp.NewList("http", sexp.NewAtom("POST")),
		sexp.NewList("file", sexp.NewAtom("/etc/passwd")),
	}

	// Save to file
	if err := persist.SaveFile(filename, rules, persist.FormatCanonical); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Saved %d rules to %s\n", len(rules), filename)

	// Load from file
	loaded, err := persist.LoadFileToSlice(filename)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded %d rules from %s\n", len(loaded), filename)

	// Display file contents
	content, _ := os.ReadFile(filename)
	fmt.Println("\nFile contents:")
	fmt.Println(string(content))
}

func example2(tmpDir string) {
	fmt.Println("Example 2: Binary Serialization")
	fmt.Println("--------------------------------")

	textFile := filepath.Join(tmpDir, "rules_text.txt")
	binaryFile := filepath.Join(tmpDir, "rules.spocp")

	// Create a large ruleset
	rules := make([]sexp.Element, 1000)
	for i := 0; i < 1000; i++ {
		action := "read"
		if i%2 == 0 {
			action = "write"
		}
		rules[i] = sexp.NewList(action, sexp.NewAtom("/path/to/file"))
	}

	// Save in text format
	if err := persist.SaveFile(textFile, rules, persist.FormatCanonical); err != nil {
		log.Fatal(err)
	}

	// Save in binary format
	if err := persist.SaveFile(binaryFile, rules, persist.FormatBinary); err != nil {
		log.Fatal(err)
	}

	// Compare file sizes
	textInfo, _ := os.Stat(textFile)
	binaryInfo, _ := os.Stat(binaryFile)

	fmt.Printf("Text format:   %d bytes\n", textInfo.Size())
	fmt.Printf("Binary format: %d bytes\n", binaryInfo.Size())
	fmt.Printf("Compression:   %.1f%%\n",
		100.0*(1.0-float64(binaryInfo.Size())/float64(textInfo.Size())))
	fmt.Println()
}

func example3(tmpDir string) {
	fmt.Println("Example 3: Loading with Comments")
	fmt.Println("---------------------------------")

	filename := filepath.Join(tmpDir, "rules_with_comments.txt")

	// Create a file with comments and blank lines
	content := `# HTTP access control rules
# Updated: 2025-12-10

(4:http3:GET)  
(4:http4:POST)

// File access rules
(4:file11:/etc/passwd)
(4:file8:/var/log)

; Legacy format comments also supported
(5:admin)
`

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		log.Fatal(err)
	}

	// Load rules (comments automatically filtered)
	rules, err := persist.LoadFileToSlice(filename)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("File has %d lines with comments\n", len(content))
	fmt.Printf("Loaded %d actual rules\n", len(rules))
	fmt.Println("\nParsed rules:")
	for i, rule := range rules {
		fmt.Printf("  %d: %s\n", i+1, sexp.AdvancedForm(rule))
	}
	fmt.Println()
}

func example4(tmpDir string) {
	fmt.Println("Example 4: Direct Engine Loading")
	fmt.Println("---------------------------------")

	filename := filepath.Join(tmpDir, "policy.txt")

	// Create a policy file
	content := `# Web application policy
(4:http3:GET)
(4:http4:POST)
(4:http3:PUT)
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		log.Fatal(err)
	}

	// Create engine and load rules directly
	engine := spocp.NewAdaptiveEngine()
	if err := engine.LoadRulesFromFile(filename); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d rules into engine\n", engine.RuleCount())

	// Test queries
	queries := []string{
		"(4:http3:GET)",
		"(4:http6:DELETE)",
	}

	for _, q := range queries {
		allowed, _ := engine.Query(q)
		status := "✓ allowed"
		if !allowed {
			status = "✗ denied"
		}
		fmt.Printf("Query %s: %s\n", sexp.AdvancedForm(mustParse(q)), status)
	}

	// Save engine state
	saveFile := filepath.Join(tmpDir, "engine_state.spocp")
	if err := engine.SaveRulesToFile(saveFile, persist.FormatBinary); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nEngine state saved to %s\n", saveFile)

	// Restore to new engine
	engine2 := spocp.NewEngine()
	if err := engine2.LoadRulesFromFile(saveFile); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Restored %d rules to new engine\n\n", engine2.RuleCount())
}

func example5(tmpDir string) {
	fmt.Println("Example 5: Performance Comparison")
	fmt.Println("----------------------------------")

	// Create large ruleset
	ruleCount := 10000
	rules := make([]sexp.Element, ruleCount)
	for i := 0; i < ruleCount; i++ {
		action := []string{"read", "write", "execute", "delete"}[i%4]
		rules[i] = sexp.NewList(action, sexp.NewAtom("/file"))
	}

	textFile := filepath.Join(tmpDir, "perf_text.txt")
	binaryFile := filepath.Join(tmpDir, "perf_binary.spocp")

	// Save both formats
	persist.SaveFile(textFile, rules, persist.FormatCanonical)
	persist.SaveFile(binaryFile, rules, persist.FormatBinary)

	// Measure load times (simple timing)
	fmt.Printf("Loading %d rules...\n", ruleCount)

	// Text format
	_, err := persist.LoadFile(textFile, persist.DefaultLoadOptions())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Text format loaded")

	// Binary format
	_, err = persist.LoadFile(binaryFile, persist.LoadOptions{Format: persist.FormatBinary})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✓ Binary format loaded")

	// File size comparison
	textInfo, _ := os.Stat(textFile)
	binaryInfo, _ := os.Stat(binaryFile)

	fmt.Printf("\nFile sizes for %d rules:\n", ruleCount)
	fmt.Printf("  Text:   %d bytes (%.2f KB)\n", textInfo.Size(), float64(textInfo.Size())/1024)
	fmt.Printf("  Binary: %d bytes (%.2f KB)\n", binaryInfo.Size(), float64(binaryInfo.Size())/1024)
	fmt.Printf("  Savings: %.1f%%\n",
		100.0*(1.0-float64(binaryInfo.Size())/float64(textInfo.Size())))
}

func mustParse(s string) sexp.Element {
	parser := sexp.NewParser(s)
	elem, _ := parser.Parse()
	return elem
}
