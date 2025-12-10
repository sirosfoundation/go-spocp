package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
	"github.com/sirosfoundation/go-spocp/pkg/starform"
)

// RuleGenerator generates random rules for performance testing
type RuleGenerator struct {
	rnd *rand.Rand
}

func NewRuleGenerator(seed int64) *RuleGenerator {
	return &RuleGenerator{
		rnd: rand.New(rand.NewSource(seed)),
	}
}

func (g *RuleGenerator) generateRandomAtom() string {
	length := g.rnd.Intn(10) + 3
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[g.rnd.Intn(len(chars))]
	}
	return string(result)
}

func (g *RuleGenerator) generateHTTPRule(useWildcard bool) sexp.Element {
	actions := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	resources := []string{"/api/users", "/api/posts", "/api/comments", "/admin", "/public"}

	var actionElem sexp.Element
	if useWildcard && g.rnd.Float32() < 0.2 {
		actionElem = &starform.Wildcard{}
	} else {
		actionElem = sexp.NewAtom(actions[g.rnd.Intn(len(actions))])
	}

	var resourceElem sexp.Element
	if useWildcard && g.rnd.Float32() < 0.3 {
		resourceElem = &starform.Prefix{Value: resources[g.rnd.Intn(len(resources))]}
	} else {
		resourceElem = sexp.NewAtom(resources[g.rnd.Intn(len(resources))] + "/" + g.generateRandomAtom())
	}

	var userElem sexp.Element
	if useWildcard && g.rnd.Float32() < 0.5 {
		userElem = sexp.NewList("user")
	} else {
		userElem = sexp.NewList("user", sexp.NewAtom("user"+fmt.Sprintf("%d", g.rnd.Intn(100))))
	}

	return sexp.NewList("http",
		sexp.NewList("resource", resourceElem),
		sexp.NewList("action", actionElem),
		userElem,
	)
}

func (g *RuleGenerator) generateFileRule(useWildcard bool) sexp.Element {
	if useWildcard && g.rnd.Float32() < 0.4 {
		dirs := []string{"/etc/", "/var/", "/home/", "/tmp/", "/opt/"}
		return sexp.NewList("file", &starform.Prefix{Value: dirs[g.rnd.Intn(len(dirs))]})
	} else if useWildcard && g.rnd.Float32() < 0.3 {
		exts := []string{".pdf", ".txt", ".log", ".conf", ".json"}
		return sexp.NewList("file", &starform.Suffix{Value: exts[g.rnd.Intn(len(exts))]})
	} else {
		path := "/" + g.generateRandomAtom() + "/" + g.generateRandomAtom()
		return sexp.NewList("file", sexp.NewAtom(path))
	}
}

func (g *RuleGenerator) generateRoleRule(useWildcard bool) sexp.Element {
	roles := []string{"admin", "user", "guest", "moderator", "editor"}
	actions := []string{"read", "write", "delete", "execute", "create"}

	var actionElem sexp.Element
	if useWildcard && g.rnd.Float32() < 0.3 {
		actionElem = &starform.Wildcard{}
	} else if useWildcard && g.rnd.Float32() < 0.4 {
		numActions := g.rnd.Intn(3) + 2
		setElems := make([]sexp.Element, numActions)
		for i := range setElems {
			setElems[i] = sexp.NewAtom(actions[g.rnd.Intn(len(actions))])
		}
		actionElem = &starform.Set{Elements: setElems}
	} else {
		actionElem = sexp.NewAtom(actions[g.rnd.Intn(len(actions))])
	}

	return sexp.NewList("permission",
		sexp.NewList("role", sexp.NewAtom(roles[g.rnd.Intn(len(roles))])),
		sexp.NewList("action", actionElem),
	)
}

func (g *RuleGenerator) generateTimeRule() sexp.Element {
	hours := []string{"08:00:00", "09:00:00", "10:00:00", "17:00:00", "18:00:00", "20:00:00"}
	startHour := hours[g.rnd.Intn(3)]
	endHour := hours[3+g.rnd.Intn(3)]

	return sexp.NewList("access", &starform.Range{
		RangeType: starform.RangeTime,
		LowerBound: &starform.RangeBound{
			Op:    starform.OpGE,
			Value: startHour,
		},
		UpperBound: &starform.RangeBound{
			Op:    starform.OpLE,
			Value: endHour,
		},
	})
}

func (g *RuleGenerator) GenerateRuleset(size int, useWildcards bool) []sexp.Element {
	rules := make([]sexp.Element, size)
	for i := 0; i < size; i++ {
		ruleType := g.rnd.Intn(4)
		switch ruleType {
		case 0:
			rules[i] = g.generateHTTPRule(useWildcards)
		case 1:
			rules[i] = g.generateFileRule(useWildcards)
		case 2:
			rules[i] = g.generateRoleRule(useWildcards)
		case 3:
			rules[i] = g.generateTimeRule()
		}
	}
	return rules
}

func runPerformanceTest(ruleCount int, queryCount int, useWildcards bool) {
	fmt.Printf("\n=== Performance Test: %d rules, %d queries ===\n", ruleCount, queryCount)

	gen := NewRuleGenerator(42)
	engine := spocp.NewEngine()

	// Measure rule addition time
	fmt.Print("Generating rules... ")
	startGen := time.Now()
	rules := gen.GenerateRuleset(ruleCount, useWildcards)
	genDuration := time.Since(startGen)
	fmt.Printf("done in %v\n", genDuration)

	fmt.Print("Adding rules to engine... ")
	startAdd := time.Now()
	for _, rule := range rules {
		engine.AddRuleElement(rule)
	}
	addDuration := time.Since(startAdd)
	fmt.Printf("done in %v\n", addDuration)
	fmt.Printf("  Rules/sec: %.2f\n", float64(ruleCount)/addDuration.Seconds())

	// Generate queries
	fmt.Print("Generating queries... ")
	startQueryGen := time.Now()
	queries := gen.GenerateRuleset(queryCount, false)
	queryGenDuration := time.Since(startQueryGen)
	fmt.Printf("done in %v\n", queryGenDuration)

	// Measure query performance
	fmt.Printf("Running %d queries... ", queryCount)
	startQuery := time.Now()
	matchCount := 0
	for _, query := range queries {
		if engine.QueryElement(query) {
			matchCount++
		}
	}
	queryDuration := time.Since(startQuery)
	fmt.Printf("done in %v\n", queryDuration)

	// Statistics
	avgQueryTime := queryDuration / time.Duration(queryCount)
	queriesPerSec := float64(queryCount) / queryDuration.Seconds()

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Total rules:        %d\n", ruleCount)
	fmt.Printf("  Total queries:      %d\n", queryCount)
	fmt.Printf("  Matches found:      %d (%.1f%%)\n", matchCount, float64(matchCount)*100/float64(queryCount))
	fmt.Printf("  Avg query time:     %v\n", avgQueryTime)
	fmt.Printf("  Queries/sec:        %.2f\n", queriesPerSec)
	fmt.Printf("  Total time:         %v\n", queryDuration)
}

func main() {
	// Command line flags
	rules := flag.Int("rules", 1000, "Number of rules to generate")
	queries := flag.Int("queries", 1000, "Number of queries to run")
	wildcards := flag.Bool("wildcards", true, "Use wildcards in rules")
	sizes := flag.Bool("sizes", false, "Run tests with different ruleset sizes")

	flag.Parse()

	fmt.Println("SPOCP Performance Testing Tool")
	fmt.Println("===============================")

	if *sizes {
		// Run tests with different sizes
		sizes := []int{100, 500, 1000, 5000, 10000, 25000, 50000}
		for _, size := range sizes {
			runPerformanceTest(size, 1000, *wildcards)
		}
	} else {
		// Run single test with specified parameters
		runPerformanceTest(*rules, *queries, *wildcards)
	}

	fmt.Println("\n=== Performance Summary ===")
	fmt.Println("Engine characteristics:")
	fmt.Println("  - Linear search through rules")
	fmt.Println("  - Early termination on first match")
	fmt.Println("  - No caching or indexing")
	fmt.Println("\nFor better performance with large rulesets, consider:")
	fmt.Println("  - Indexing rules by tag")
	fmt.Println("  - Caching query results")
	fmt.Println("  - Parallel query evaluation")
	fmt.Println("  - Rule priority/ordering optimization")
}
