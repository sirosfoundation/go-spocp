// Package spocp provides an example implementation of tag-based indexing optimization.
// This is a demonstration of the highest-impact performance improvement.
package main

import (
	"fmt"

	"github.com/sirosfoundation/go-spocp/pkg/compare"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// IndexedEngine extends the basic SPOCP engine with tag-based indexing
type IndexedEngine struct {
	rules     []sexp.Element
	tagIndex  map[string][]int // tag -> slice of rule indices
	atomRules []int            // indices of non-list rules
}

// NewIndexedEngine creates a new indexed SPOCP engine
func NewIndexedEngine() *IndexedEngine {
	return &IndexedEngine{
		rules:     make([]sexp.Element, 0),
		tagIndex:  make(map[string][]int),
		atomRules: make([]int, 0),
	}
}

// AddRule adds a policy rule and updates the index
func (e *IndexedEngine) AddRule(rule string) error {
	parser := sexp.NewParser(rule)
	elem, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse rule: %v", err)
	}
	e.AddRuleElement(elem)
	return nil
}

// AddRuleElement adds a parsed rule element and updates the index
func (e *IndexedEngine) AddRuleElement(rule sexp.Element) {
	idx := len(e.rules)
	e.rules = append(e.rules, rule)

	// Update index based on rule type
	if list, ok := rule.(*sexp.List); ok {
		// List rule - index by tag
		e.tagIndex[list.Tag] = append(e.tagIndex[list.Tag], idx)
	} else {
		// Atom or star form - keep in separate list
		e.atomRules = append(e.atomRules, idx)
	}
}

// Query checks if a query is authorized using indexed lookup
func (e *IndexedEngine) Query(query string) (bool, error) {
	parser := sexp.NewParser(query)
	queryElem, err := parser.Parse()
	if err != nil {
		return false, fmt.Errorf("failed to parse query: %v", err)
	}
	return e.QueryElement(queryElem), nil
}

// QueryElement performs indexed query lookup
func (e *IndexedEngine) QueryElement(query sexp.Element) bool {
	// Fast path: if query is a list, only check rules with matching tag
	if list, ok := query.(*sexp.List); ok {
		return e.queryByTag(query, list.Tag)
	}

	// For atoms and star forms, check all non-list rules plus tag-less lists
	return e.queryNonList(query)
}

// queryByTag checks only rules with matching tag (the optimization!)
func (e *IndexedEngine) queryByTag(query sexp.Element, tag string) bool {
	// Check rules with matching tag
	if indices, exists := e.tagIndex[tag]; exists {
		for _, idx := range indices {
			if compare.LessPermissive(query, e.rules[idx]) {
				return true
			}
		}
	}

	// Also check wildcard rules (if they exist)
	// This could be further optimized by tracking wildcard rules separately
	if indices, exists := e.tagIndex["*"]; exists {
		for _, idx := range indices {
			if compare.LessPermissive(query, e.rules[idx]) {
				return true
			}
		}
	}

	return false
}

// queryNonList checks atom and star form rules
func (e *IndexedEngine) queryNonList(query sexp.Element) bool {
	for _, idx := range e.atomRules {
		if compare.LessPermissive(query, e.rules[idx]) {
			return true
		}
	}
	return false
}

// FindMatchingRules returns all rules that authorize the query (indexed)
func (e *IndexedEngine) FindMatchingRules(query string) ([]sexp.Element, error) {
	parser := sexp.NewParser(query)
	queryElem, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %v", err)
	}

	var matches []sexp.Element

	if list, ok := queryElem.(*sexp.List); ok {
		// Query is a list - use tag index
		if indices, exists := e.tagIndex[list.Tag]; exists {
			for _, idx := range indices {
				if compare.LessPermissive(queryElem, e.rules[idx]) {
					matches = append(matches, e.rules[idx])
				}
			}
		}
		// Also check wildcard tags
		if indices, exists := e.tagIndex["*"]; exists {
			for _, idx := range indices {
				if compare.LessPermissive(queryElem, e.rules[idx]) {
					matches = append(matches, e.rules[idx])
				}
			}
		}
	} else {
		// Query is atom/star form - check atom rules
		for _, idx := range e.atomRules {
			if compare.LessPermissive(queryElem, e.rules[idx]) {
				matches = append(matches, e.rules[idx])
			}
		}
	}

	return matches, nil
}

// RuleCount returns the number of rules in the engine
func (e *IndexedEngine) RuleCount() int {
	return len(e.rules)
}

// GetIndexStats returns statistics about the index
func (e *IndexedEngine) GetIndexStats() map[string]any {
	stats := make(map[string]any)
	stats["total_rules"] = len(e.rules)
	stats["unique_tags"] = len(e.tagIndex)
	stats["atom_rules"] = len(e.atomRules)

	// Calculate average rules per tag
	if len(e.tagIndex) > 0 {
		total := 0
		for _, indices := range e.tagIndex {
			total += len(indices)
		}
		stats["avg_rules_per_tag"] = float64(total) / float64(len(e.tagIndex))
	}

	// Find most common tag
	maxCount := 0
	maxTag := ""
	for tag, indices := range e.tagIndex {
		if len(indices) > maxCount {
			maxCount = len(indices)
			maxTag = tag
		}
	}
	if maxTag != "" {
		stats["most_common_tag"] = maxTag
		stats["most_common_tag_count"] = maxCount
	}

	return stats
}

// Clear removes all rules and clears the index
func (e *IndexedEngine) Clear() {
	e.rules = make([]sexp.Element, 0)
	e.tagIndex = make(map[string][]int)
	e.atomRules = make([]int, 0)
}

// Example usage
func exampleIndexedEngine() {
	engine := NewIndexedEngine()

	// Add rules
	engine.AddRule("(read /home/user/docs/file.txt)")
	engine.AddRule("(read /home/user/docs/*)")
	engine.AddRule("(write /tmp/*)")
	engine.AddRule("(execute /usr/bin/python)")

	// Query - will use tag index
	allowed, _ := engine.Query("(read /home/user/docs/file.txt)")
	fmt.Printf("Query authorized: %v\n", allowed)

	// Get index stats
	stats := engine.GetIndexStats()
	fmt.Printf("Index stats: %+v\n", stats)

	// Performance comparison example
	fmt.Println("\n=== Performance Comparison ===")
	fmt.Println("Unindexed: Check all rules")
	fmt.Println("Indexed:   Check only rules with matching tag")
	fmt.Println()
	fmt.Println("Example with 10,000 rules:")
	fmt.Println("- 2,000 rules with tag 'read'")
	fmt.Println("- 2,000 rules with tag 'write'")
	fmt.Println("- 2,000 rules with tag 'execute'")
	fmt.Println("- 2,000 rules with tag 'delete'")
	fmt.Println("- 2,000 rules with tag 'admin'")
	fmt.Println()
	fmt.Println("Query: (read /path/to/file)")
	fmt.Println("Unindexed: Checks all 10,000 rules")
	fmt.Println("Indexed:   Checks only 2,000 'read' rules")
	fmt.Println("Speedup:   5x faster!")
	fmt.Println()
	fmt.Println("With more diverse tags, speedup can be 10-100x")
}
