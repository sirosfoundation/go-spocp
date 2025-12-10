package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

func main() {
	numRules := flag.Int("rules", 10000, "Number of rules to generate")
	numQueries := flag.Int("queries", 10000, "Number of queries to run")
	numTags := flag.Int("tags", 8, "Number of unique tags")
	showStats := flag.Bool("stats", false, "Show index statistics")
	flag.Parse()

	fmt.Println("=== SPOCP Tag-Based Indexing Performance Comparison ===")
	fmt.Printf("Rules: %d, Queries: %d, Tags: %d\n\n", *numRules, *numQueries, *numTags)

	// Generate tag names
	tags := make([]string, *numTags)
	for i := 0; i < *numTags; i++ {
		tags[i] = fmt.Sprintf("action%d", i)
	}

	// Common paths for rules
	paths := []string{"/home/*", "/var/*", "/tmp/*", "/etc/*", "/usr/*", "/opt/*", "/root/*", "/srv/*"}

	// Test with INDEXED engine
	fmt.Println("🚀 Testing INDEXED engine...")
	indexedEngine := spocp.NewEngineWithIndexing(true)

	// Add rules
	addStart := time.Now()
	for i := 0; i < *numRules; i++ {
		tag := tags[rand.Intn(len(tags))]
		path := paths[rand.Intn(len(paths))]

		rule := sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom(path)),
			sexp.NewList("user", sexp.NewAtom("*")),
			sexp.NewList("time", sexp.NewAtom("*")),
		)
		indexedEngine.AddRuleElement(rule)
	}
	addDuration := time.Since(addStart)

	// Show index stats
	if *showStats {
		stats := indexedEngine.GetIndexStats()
		fmt.Printf("\n  Index Statistics:\n")
		fmt.Printf("    Total rules: %v\n", stats["total_rules"])
		fmt.Printf("    Unique tags: %v\n", stats["unique_tags"])
		fmt.Printf("    Avg rules per tag: %.2f\n", stats["avg_rules_per_tag"])
		if tag, ok := stats["most_common_tag"]; ok {
			fmt.Printf("    Most common tag: %v (%v rules)\n", tag, stats["most_common_tag_count"])
		}
		fmt.Println()
	}

	// Generate queries
	queries := make([]sexp.Element, *numQueries)
	for i := 0; i < *numQueries; i++ {
		tag := tags[rand.Intn(len(tags))]
		path := paths[rand.Intn(len(paths))]
		queries[i] = sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom(path)),
			sexp.NewList("user", sexp.NewAtom("alice")),
			sexp.NewList("time", sexp.NewAtom("12:00:00")),
		)
	}

	// Run queries
	queryStart := time.Now()
	matchCount := 0
	for _, query := range queries {
		if indexedEngine.QueryElement(query) {
			matchCount++
		}
	}
	queryDuration := time.Since(queryStart)

	fmt.Printf("  Rule addition: %v (%.2f µs/rule)\n", addDuration, float64(addDuration.Microseconds())/float64(*numRules))
	fmt.Printf("  Query time: %v (%.2f µs/query)\n", queryDuration, float64(queryDuration.Microseconds())/float64(*numQueries))
	fmt.Printf("  Throughput: %.0f queries/sec\n", float64(*numQueries)/queryDuration.Seconds())
	fmt.Printf("  Match rate: %.1f%%\n", float64(matchCount)*100/float64(*numQueries))

	// Test with NON-INDEXED engine
	fmt.Println("\n🐌 Testing NON-INDEXED engine...")
	nonIndexedEngine := spocp.NewEngineWithIndexing(false)

	// Add same rules
	addStart = time.Now()
	for i := 0; i < *numRules; i++ {
		tag := tags[rand.Intn(len(tags))]
		path := paths[rand.Intn(len(paths))]

		rule := sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom(path)),
			sexp.NewList("user", sexp.NewAtom("*")),
			sexp.NewList("time", sexp.NewAtom("*")),
		)
		nonIndexedEngine.AddRuleElement(rule)
	}
	addDuration2 := time.Since(addStart)

	// Run same queries
	queryStart = time.Now()
	matchCount2 := 0
	for _, query := range queries {
		if nonIndexedEngine.QueryElement(query) {
			matchCount2++
		}
	}
	queryDuration2 := time.Since(queryStart)

	fmt.Printf("  Rule addition: %v (%.2f µs/rule)\n", addDuration2, float64(addDuration2.Microseconds())/float64(*numRules))
	fmt.Printf("  Query time: %v (%.2f µs/query)\n", queryDuration2, float64(queryDuration2.Microseconds())/float64(*numQueries))
	fmt.Printf("  Throughput: %.0f queries/sec\n", float64(*numQueries)/queryDuration2.Seconds())
	fmt.Printf("  Match rate: %.1f%%\n", float64(matchCount2)*100/float64(*numQueries))

	// Calculate improvement
	fmt.Println("\n📊 COMPARISON:")
	fmt.Println("  ==========================================")

	addSpeedup := float64(addDuration2) / float64(addDuration)
	if addSpeedup >= 1.0 {
		fmt.Printf("  Rule addition: %.2fx SLOWER with index\n", addSpeedup)
	} else {
		fmt.Printf("  Rule addition: %.2fx FASTER with index\n", 1.0/addSpeedup)
	}

	querySpeedup := float64(queryDuration2) / float64(queryDuration)
	fmt.Printf("  Query execution: %.2fx FASTER with index\n", querySpeedup)
	fmt.Printf("  Speedup ratio: %.1fx\n", querySpeedup)

	fmt.Println("  ==========================================")

	// Calculate effective improvement considering both operations
	if querySpeedup > 2.0 {
		fmt.Println("  ✅ EXCELLENT - Indexing provides significant benefit")
	} else if querySpeedup > 1.5 {
		fmt.Println("  ✅ GOOD - Indexing provides moderate benefit")
	} else if querySpeedup > 1.1 {
		fmt.Println("  ⚠️  MARGINAL - Indexing provides small benefit")
	} else {
		fmt.Println("  ❌ NOT BENEFICIAL - Consider disabling indexing")
	}

	// Recommendations
	fmt.Println("\n💡 RECOMMENDATIONS:")
	if *numRules < 1000 {
		fmt.Println("  • Ruleset is small (<1k) - indexing overhead may not be worth it")
	} else if *numRules < 5000 {
		fmt.Println("  • Ruleset is medium (1k-5k) - indexing provides good benefits")
	} else {
		fmt.Println("  • Ruleset is large (>5k) - indexing is highly recommended")
	}

	if *numTags < 5 {
		fmt.Println("  • Few unique tags (<5) - consider adding more tag diversity")
		fmt.Println("  • More tags = better index selectivity = faster queries")
	} else if *numTags > 20 {
		fmt.Println("  • Many unique tags (>20) - excellent index selectivity")
	}

	avgRulesPerTag := float64(*numRules) / float64(*numTags)
	if avgRulesPerTag > 2000 {
		fmt.Printf("  • High rules-per-tag ratio (%.0f) - index still checks many rules\n", avgRulesPerTag)
		fmt.Println("  • Consider using more granular tags or hierarchical indexing")
	}
}
