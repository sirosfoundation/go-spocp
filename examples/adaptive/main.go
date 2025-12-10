package main

import (
	"fmt"
	"log"

	"github.com/sirosfoundation/go-spocp"
)

func main() {
	fmt.Println("=== SPOCP Adaptive Engine Demo ===\n")

	// Create an adaptive engine - it automatically decides when to use indexing
	engine := spocp.NewAdaptiveEngine()

	// Scenario 1: Small ruleset (indexing not beneficial)
	fmt.Println("Scenario 1: Small ruleset (10 rules)")
	for i := 0; i < 10; i++ {
		err := engine.AddRule("(4:read4:file)")
		if err != nil {
			log.Fatal(err)
		}
	}

	stats := engine.Stats()
	fmt.Printf("  Rules: %d\n", stats.TotalRules)
	fmt.Printf("  Indexing enabled: %v\n", stats.IndexingEnabled)
	fmt.Printf("  Reason: Not enough rules (< 50)\n\n")

	// Clear for next scenario
	engine.Clear()

	// Scenario 2: Large ruleset with diverse tags (indexing beneficial)
	fmt.Println("Scenario 2: Large ruleset with diverse tags (100 rules, 10 tags)")

	tags := []string{"read", "write", "delete", "update", "create",
		"list", "get", "post", "put", "patch"}

	for i := 0; i < 100; i++ {
		tag := tags[i%len(tags)]
		rule := fmt.Sprintf("(%d:%s4:file)", len(tag), tag)
		err := engine.AddRule(rule)
		if err != nil {
			log.Fatal(err)
		}
	}

	stats = engine.Stats()
	fmt.Printf("  Rules: %d\n", stats.TotalRules)
	fmt.Printf("  Unique tags: %d\n", stats.UniqueTags)
	fmt.Printf("  Average fanout: %.1f rules/tag\n", stats.AvgTagFanout)
	fmt.Printf("  Indexing enabled: %v\n", stats.IndexingEnabled)
	fmt.Printf("  Reason: Many rules with selective tags\n\n")

	// Perform queries
	fmt.Println("Querying...")
	allowed, err := engine.Query("(4:read4:file)")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Query '(read file)': %v\n", allowed)

	allowed, err = engine.Query("(6:delete4:file)")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Query '(delete file)': %v\n\n", allowed)

	// Find all matching rules
	matches, err := engine.FindMatchingRules("(4:read4:file)")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Found %d matching rules for '(read file)'\n\n", len(matches))

	// Scenario 3: Large ruleset with few tags (indexing not beneficial)
	engine.Clear()
	fmt.Println("Scenario 3: Large ruleset with few tags (100 rules, 2 tags)")

	for i := 0; i < 100; i++ {
		tag := "read"
		if i%2 == 0 {
			tag = "write"
		}
		rule := fmt.Sprintf("(%d:%s4:file)", len(tag), tag)
		err := engine.AddRule(rule)
		if err != nil {
			log.Fatal(err)
		}
	}

	stats = engine.Stats()
	fmt.Printf("  Rules: %d\n", stats.TotalRules)
	fmt.Printf("  Unique tags: %d\n", stats.UniqueTags)
	fmt.Printf("  Indexing enabled: %v\n", stats.IndexingEnabled)
	fmt.Printf("  Reason: Not enough unique tags (< 5)\n\n")

	// Scenario 4: Manual override
	engine.Clear()
	fmt.Println("Scenario 4: Manual override for testing")

	for i := 0; i < 10; i++ {
		err := engine.AddRule("(4:test)")
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("  Before override - Indexing: %v\n", engine.Stats().IndexingEnabled)

	engine.ForceIndexing(true)
	fmt.Printf("  After override - Indexing: %v\n", engine.Stats().IndexingEnabled)
	fmt.Printf("  Use case: Testing indexing behavior with small datasets\n\n")

	// Show detailed stats
	fmt.Println("Detailed Index Statistics:")
	indexStats := engine.GetIndexStats()
	for key, value := range indexStats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	fmt.Println("\n=== Demo Complete ===")
}
