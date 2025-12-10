package spocp

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// Benchmark comparison between indexed and non-indexed engines

func BenchmarkIndexed_SmallRuleset_100(b *testing.B) {
	benchmarkIndexComparison(b, 100, true)
}

func BenchmarkNonIndexed_SmallRuleset_100(b *testing.B) {
	benchmarkIndexComparison(b, 100, false)
}

func BenchmarkIndexed_MediumRuleset_1k(b *testing.B) {
	benchmarkIndexComparison(b, 1000, true)
}

func BenchmarkNonIndexed_MediumRuleset_1k(b *testing.B) {
	benchmarkIndexComparison(b, 1000, false)
}

func BenchmarkIndexed_LargeRuleset_10k(b *testing.B) {
	benchmarkIndexComparison(b, 10000, true)
}

func BenchmarkNonIndexed_LargeRuleset_10k(b *testing.B) {
	benchmarkIndexComparison(b, 10000, false)
}

func BenchmarkIndexed_VeryLargeRuleset_50k(b *testing.B) {
	benchmarkIndexComparison(b, 50000, true)
}

func BenchmarkNonIndexed_VeryLargeRuleset_50k(b *testing.B) {
	benchmarkIndexComparison(b, 50000, false)
}

// Helper function for index comparison benchmarks
func benchmarkIndexComparison(b *testing.B, numRules int, indexed bool) {
	engine := NewEngineWithIndexing(indexed)

	// Generate diverse rules with multiple tags
	tags := []string{"read", "write", "execute", "delete", "admin", "user", "system", "network"}
	paths := []string{"/home/*", "/var/*", "/tmp/*", "/etc/*", "/usr/*", "/opt/*"}

	for i := 0; i < numRules; i++ {
		tag := tags[rand.Intn(len(tags))]
		path := paths[rand.Intn(len(paths))]

		rule := sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom(path)),
			sexp.NewList("user", sexp.NewAtom("*")),
		)
		engine.AddRuleElement(rule)
	}

	// Generate test queries that will match various tags
	queries := make([]sexp.Element, 100)
	for i := range queries {
		tag := tags[rand.Intn(len(tags))]
		path := paths[rand.Intn(len(paths))]
		queries[i] = sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom(path)),
			sexp.NewList("user", sexp.NewAtom("alice")),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		engine.QueryElement(query)
	}
}

// Benchmark different tag distribution patterns

func BenchmarkIndexed_UniformDistribution_10k(b *testing.B) {
	benchmarkWithTagDistribution(b, 10000, 10, true) // 10 tags, uniform
}

func BenchmarkNonIndexed_UniformDistribution_10k(b *testing.B) {
	benchmarkWithTagDistribution(b, 10000, 10, false)
}

func BenchmarkIndexed_HighlySkewed_10k(b *testing.B) {
	// 90% of rules have one tag, rest distributed
	engine := NewEngineWithIndexing(true)

	for i := 0; i < 9000; i++ {
		rule := sexp.NewList("read",
			sexp.NewList("path", sexp.NewAtom("/home/*")),
		)
		engine.AddRuleElement(rule)
	}

	tags := []string{"write", "execute", "delete", "admin"}
	for i := 0; i < 1000; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom("/tmp/*")),
		)
		engine.AddRuleElement(rule)
	}

	// Query against minority tags (best case for indexing)
	query := sexp.NewList("write",
		sexp.NewList("path", sexp.NewAtom("/tmp/*")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.QueryElement(query)
	}
}

func BenchmarkNonIndexed_HighlySkewed_10k(b *testing.B) {
	engine := NewEngineWithIndexing(false)

	for i := 0; i < 9000; i++ {
		rule := sexp.NewList("read",
			sexp.NewList("path", sexp.NewAtom("/home/*")),
		)
		engine.AddRuleElement(rule)
	}

	tags := []string{"write", "execute", "delete", "admin"}
	for i := 0; i < 1000; i++ {
		tag := tags[i%len(tags)]
		rule := sexp.NewList(tag,
			sexp.NewList("path", sexp.NewAtom("/tmp/*")),
		)
		engine.AddRuleElement(rule)
	}

	query := sexp.NewList("write",
		sexp.NewList("path", sexp.NewAtom("/tmp/*")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.QueryElement(query)
	}
}

// Helper for tag distribution benchmarks
func benchmarkWithTagDistribution(b *testing.B, numRules, numTags int, indexed bool) {
	engine := NewEngineWithIndexing(indexed)

	tags := make([]string, numTags)
	for i := 0; i < numTags; i++ {
		tags[i] = fmt.Sprintf("tag%d", i)
	}

	for i := 0; i < numRules; i++ {
		tag := tags[i%numTags]
		rule := sexp.NewList(tag,
			sexp.NewList("resource", sexp.NewAtom(fmt.Sprintf("resource%d", i))),
		)
		engine.AddRuleElement(rule)
	}

	// Generate queries
	queries := make([]sexp.Element, 100)
	for i := range queries {
		tag := tags[rand.Intn(numTags)]
		queries[i] = sexp.NewList(tag,
			sexp.NewList("resource", sexp.NewAtom("resource123")),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		engine.QueryElement(query)
	}
}

// Benchmark index overhead during rule addition

func BenchmarkIndexed_AddRule(b *testing.B) {
	engine := NewEngineWithIndexing(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rule := sexp.NewList("read",
			sexp.NewList("path", sexp.NewAtom("/tmp/*")),
		)
		engine.AddRuleElement(rule)

		if i%1000 == 0 {
			engine.Clear() // Prevent unbounded growth
		}
	}
}

func BenchmarkNonIndexed_AddRule(b *testing.B) {
	engine := NewEngineWithIndexing(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rule := sexp.NewList("read",
			sexp.NewList("path", sexp.NewAtom("/tmp/*")),
		)
		engine.AddRuleElement(rule)

		if i%1000 == 0 {
			engine.Clear()
		}
	}
}

// Benchmark best vs worst case for indexed queries

func BenchmarkIndexed_BestCase_10k(b *testing.B) {
	// Best case: query tag matches first rule
	engine := NewEngineWithIndexing(true)

	// Add one matching rule first
	engine.AddRuleElement(sexp.NewList("target",
		sexp.NewList("path", sexp.NewAtom("/*")),
	))

	// Add many non-matching rules
	for i := 0; i < 9999; i++ {
		engine.AddRuleElement(sexp.NewList("other",
			sexp.NewList("path", sexp.NewAtom("/tmp/*")),
		))
	}

	query := sexp.NewList("target",
		sexp.NewList("path", sexp.NewAtom("/home/user/file")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.QueryElement(query)
	}
}

func BenchmarkIndexed_WorstCase_10k(b *testing.B) {
	// Worst case: no matching rules (checks all rules with that tag)
	engine := NewEngineWithIndexing(true)

	// All rules have same tag but won't match
	for i := 0; i < 10000; i++ {
		engine.AddRuleElement(sexp.NewList("read",
			sexp.NewList("path", sexp.NewAtom("/specific/path")),
		))
	}

	query := sexp.NewList("read",
		sexp.NewList("path", sexp.NewAtom("/different/path")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.QueryElement(query)
	}
}
