package spocp

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/sirosfoundation/go-spocp/pkg/sexp"
	"github.com/sirosfoundation/go-spocp/pkg/starform"
)

// RuleGenerator generates random rules for benchmarking
type RuleGenerator struct {
	rnd *rand.Rand
}

func NewRuleGenerator(seed int64) *RuleGenerator {
	return &RuleGenerator{
		rnd: rand.New(rand.NewSource(seed)),
	}
}

// generateRandomAtom creates a random atom value
func (g *RuleGenerator) generateRandomAtom() string {
	length := g.rnd.Intn(10) + 3 // 3-12 characters
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[g.rnd.Intn(len(chars))]
	}
	return string(result)
}

// generateRandomResource creates a random resource path
func (g *RuleGenerator) generateRandomResource() string {
	depth := g.rnd.Intn(4) + 1 // 1-4 levels deep
	parts := make([]string, depth)
	for i := range parts {
		parts[i] = g.generateRandomAtom()
	}
	path := ""
	for i, part := range parts {
		if i == 0 {
			path = "/" + part
		} else {
			path += "/" + part
		}
	}
	return path
}

// generateRandomElement creates a random S-expression element
func (g *RuleGenerator) generateRandomElement(depth int) sexp.Element {
	if depth > 3 || g.rnd.Float32() < 0.3 {
		return sexp.NewAtom(g.generateRandomAtom())
	}

	// Create a list with random elements
	tag := g.generateRandomAtom()
	numElements := g.rnd.Intn(3) + 1 // 1-3 elements
	elements := make([]sexp.Element, numElements)
	for i := range elements {
		elements[i] = g.generateRandomElement(depth + 1)
	}
	return sexp.NewList(tag, elements...)
}

// generateHTTPRule generates a realistic HTTP-style rule
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

// generateFileRule generates a file access rule
func (g *RuleGenerator) generateFileRule(useWildcard bool) sexp.Element {
	if useWildcard && g.rnd.Float32() < 0.4 {
		// Use prefix for directory access
		dirs := []string{"/etc/", "/var/", "/home/", "/tmp/", "/opt/"}
		return sexp.NewList("file", &starform.Prefix{Value: dirs[g.rnd.Intn(len(dirs))]})
	} else if useWildcard && g.rnd.Float32() < 0.3 {
		// Use suffix for file types
		exts := []string{".pdf", ".txt", ".log", ".conf", ".json"}
		return sexp.NewList("file", &starform.Suffix{Value: exts[g.rnd.Intn(len(exts))]})
	} else {
		return sexp.NewList("file", sexp.NewAtom(g.generateRandomResource()))
	}
}

// generateRoleRule generates a role-based access rule
func (g *RuleGenerator) generateRoleRule(useWildcard bool) sexp.Element {
	roles := []string{"admin", "user", "guest", "moderator", "editor"}
	actions := []string{"read", "write", "delete", "execute", "create"}

	var actionElem sexp.Element
	if useWildcard && g.rnd.Float32() < 0.3 {
		actionElem = &starform.Wildcard{}
	} else if useWildcard && g.rnd.Float32() < 0.4 {
		// Use set for multiple actions
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

// generateTimeRule generates a time-based access rule
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

// GenerateRuleset creates a ruleset with mixed rule types
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

// Benchmark small ruleset (100 rules)
func BenchmarkEngine_SmallRuleset(b *testing.B) {
	gen := NewRuleGenerator(42)
	engine := NewEngine()

	// Generate 100 rules
	rules := gen.GenerateRuleset(100, true)
	for _, rule := range rules {
		engine.AddRuleElement(rule)
	}

	// Generate queries
	queries := gen.GenerateRuleset(100, false)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		engine.QueryElement(query)
	}
}

// Benchmark medium ruleset (1,000 rules)
func BenchmarkEngine_MediumRuleset(b *testing.B) {
	gen := NewRuleGenerator(42)
	engine := NewEngine()

	// Generate 1,000 rules
	rules := gen.GenerateRuleset(1000, true)
	for _, rule := range rules {
		engine.AddRuleElement(rule)
	}

	// Generate queries
	queries := gen.GenerateRuleset(100, false)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		engine.QueryElement(query)
	}
}

// Benchmark large ruleset (10,000 rules)
func BenchmarkEngine_LargeRuleset(b *testing.B) {
	gen := NewRuleGenerator(42)
	engine := NewEngine()

	// Generate 10,000 rules
	rules := gen.GenerateRuleset(10000, true)
	for _, rule := range rules {
		engine.AddRuleElement(rule)
	}

	// Generate queries
	queries := gen.GenerateRuleset(100, false)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		engine.QueryElement(query)
	}
}

// Benchmark very large ruleset (50,000 rules)
func BenchmarkEngine_VeryLargeRuleset(b *testing.B) {
	gen := NewRuleGenerator(42)
	engine := NewEngine()

	// Generate 50,000 rules
	rules := gen.GenerateRuleset(50000, true)
	for _, rule := range rules {
		engine.AddRuleElement(rule)
	}

	// Generate queries
	queries := gen.GenerateRuleset(100, false)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		engine.QueryElement(query)
	}
}

// Benchmark query matching (best case - matches first rule)
func BenchmarkEngine_QueryBestCase(b *testing.B) {
	engine := NewEngine()

	// Add a wildcard rule first (matches everything)
	rule := sexp.NewList("resource", &starform.Wildcard{})
	engine.AddRuleElement(rule)

	// Add more specific rules
	gen := NewRuleGenerator(42)
	rules := gen.GenerateRuleset(1000, false)
	for _, r := range rules {
		engine.AddRuleElement(r)
	}

	query := sexp.NewList("resource", sexp.NewAtom("anything"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		engine.QueryElement(query)
	}
}

// Benchmark query matching (worst case - no match)
func BenchmarkEngine_QueryWorstCase(b *testing.B) {
	gen := NewRuleGenerator(42)
	engine := NewEngine()

	// Generate rules
	rules := gen.GenerateRuleset(1000, false)
	for _, rule := range rules {
		engine.AddRuleElement(rule)
	}

	// Query that won't match anything
	query := sexp.NewList("nonexistent",
		sexp.NewAtom("this"),
		sexp.NewAtom("will"),
		sexp.NewAtom("not"),
		sexp.NewAtom("match"),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		engine.QueryElement(query)
	}
}

// Benchmark rule addition
func BenchmarkEngine_AddRule(b *testing.B) {
	gen := NewRuleGenerator(42)
	rules := gen.GenerateRuleset(b.N, true)

	b.ResetTimer()
	b.ReportAllocs()

	engine := NewEngine()
	for i := 0; i < b.N; i++ {
		engine.AddRuleElement(rules[i])
	}
}

// Benchmark different star forms
func BenchmarkStarForm_Wildcard(b *testing.B) {
	wildcard := &starform.Wildcard{}
	atom := sexp.NewAtom("test")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wildcard.Match(atom)
	}
}

func BenchmarkStarForm_Prefix(b *testing.B) {
	prefix := &starform.Prefix{Value: "/etc/"}
	atom := sexp.NewAtom("/etc/passwd")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		prefix.Match(atom)
	}
}

func BenchmarkStarForm_Suffix(b *testing.B) {
	suffix := &starform.Suffix{Value: ".pdf"}
	atom := sexp.NewAtom("document.pdf")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		suffix.Match(atom)
	}
}

func BenchmarkStarForm_Set(b *testing.B) {
	set := &starform.Set{
		Elements: []sexp.Element{
			sexp.NewAtom("read"),
			sexp.NewAtom("write"),
			sexp.NewAtom("execute"),
			sexp.NewAtom("delete"),
			sexp.NewAtom("create"),
		},
	}
	atom := sexp.NewAtom("read")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		set.Match(atom)
	}
}

func BenchmarkStarForm_Range(b *testing.B) {
	timeRange := &starform.Range{
		RangeType: starform.RangeTime,
		LowerBound: &starform.RangeBound{
			Op:    starform.OpGE,
			Value: "08:00:00",
		},
		UpperBound: &starform.RangeBound{
			Op:    starform.OpLE,
			Value: "17:00:00",
		},
	}
	atom := sexp.NewAtom("12:00:00")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		timeRange.Match(atom)
	}
}

// Benchmark S-expression parsing
func BenchmarkParser_SimpleAtom(b *testing.B) {
	input := "5:hello"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parser := sexp.NewParser(input)
		parser.Parse()
	}
}

func BenchmarkParser_SimpleList(b *testing.B) {
	input := "(4:http3:GET)"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parser := sexp.NewParser(input)
		parser.Parse()
	}
}

func BenchmarkParser_ComplexNested(b *testing.B) {
	input := "(4:http(8:resource10:/api/users)(6:action3:GET)(4:user5:alice))"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parser := sexp.NewParser(input)
		parser.Parse()
	}
}
