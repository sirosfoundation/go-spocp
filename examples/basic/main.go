package main

import (
	"fmt"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
	"github.com/sirosfoundation/go-spocp/pkg/starform"
)

func main() {
	fmt.Println("=== SPOCP Authorization Engine Examples ===")
	fmt.Println()

	// Example 1: HTTP Access Control
	example1()

	// Example 2: File System Access
	example2()

	// Example 3: Time-Based Access
	example3()

	// Example 4: Role-Based Access
	example4()
}

func example1() {
	fmt.Println("Example 1: HTTP Access Control")
	fmt.Println("-------------------------------")

	engine := spocp.NewEngine()

	// Rule: Allow any user to GET index.html
	// (http (page index.html)(action GET)(user))
	rule := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("GET")),
		sexp.NewList("user"),
	)
	engine.AddRuleElement(rule)
	fmt.Printf("Added rule: %s\n", sexp.AdvancedForm(rule))

	// Query 1: Can alice GET index.html?
	query1 := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("GET")),
		sexp.NewList("user", sexp.NewAtom("alice")),
	)
	fmt.Printf("Query: %s\n", sexp.AdvancedForm(query1))
	fmt.Printf("Result: %v ✓\n", engine.QueryElement(query1))

	// Query 2: Can alice POST to index.html?
	query2 := sexp.NewList("http",
		sexp.NewList("page", sexp.NewAtom("index.html")),
		sexp.NewList("action", sexp.NewAtom("POST")),
		sexp.NewList("user", sexp.NewAtom("alice")),
	)
	fmt.Printf("Query: %s\n", sexp.AdvancedForm(query2))
	fmt.Printf("Result: %v ✗\n\n", engine.QueryElement(query2))
}

func example2() {
	fmt.Println("Example 2: File System Access")
	fmt.Println("------------------------------")

	engine := spocp.NewEngine()

	// Rule 1: Allow access to files under /etc/
	rule1 := sexp.NewList("file", &starform.Prefix{Value: "/etc/"})
	engine.AddRuleElement(rule1)
	fmt.Printf("Added rule: (file (* prefix /etc/))\n")

	// Rule 2: Allow access to PDF files
	rule2 := sexp.NewList("file", &starform.Suffix{Value: ".pdf"})
	engine.AddRuleElement(rule2)
	fmt.Printf("Added rule: (file (* suffix .pdf))\n")

	// Test queries
	queries := []string{
		"/etc/passwd",
		"/var/log/syslog",
		"document.pdf",
		"document.txt",
	}

	for _, path := range queries {
		query := sexp.NewList("file", sexp.NewAtom(path))
		result := engine.QueryElement(query)
		status := "✗"
		if result {
			status = "✓"
		}
		fmt.Printf("Access to %s: %v %s\n", path, result, status)
	}
	fmt.Println()
}

func example3() {
	fmt.Println("Example 3: Time-Based Access")
	fmt.Println("-----------------------------")

	engine := spocp.NewEngine()

	// Rule: Allow access during work hours (08:00:00 to 17:00:00)
	rule := sexp.NewList("access", &starform.Range{
		RangeType: starform.RangeTime,
		LowerBound: &starform.RangeBound{
			Op:    starform.OpGE,
			Value: "08:00:00",
		},
		UpperBound: &starform.RangeBound{
			Op:    starform.OpLE,
			Value: "17:00:00",
		},
	})
	engine.AddRuleElement(rule)
	fmt.Printf("Added rule: (access (* range time ge 08:00:00 le 17:00:00))\n")

	// Test different times
	times := []string{
		"07:00:00", // Before work
		"09:30:00", // During work
		"12:00:00", // Lunch time
		"16:45:00", // Near end of day
		"18:30:00", // After work
	}

	for _, t := range times {
		query := sexp.NewList("access", sexp.NewAtom(t))
		result := engine.QueryElement(query)
		status := "✗"
		if result {
			status = "✓"
		}
		fmt.Printf("Access at %s: %v %s\n", t, result, status)
	}
	fmt.Println()
}

func example4() {
	fmt.Println("Example 4: Role-Based Access Control")
	fmt.Println("-------------------------------------")

	engine := spocp.NewEngine()

	// Rule 1: Admins can perform any action
	rule1 := sexp.NewList("permission",
		sexp.NewList("role", sexp.NewAtom("admin")),
		sexp.NewList("action", &starform.Wildcard{}),
	)
	engine.AddRuleElement(rule1)
	fmt.Printf("Added rule: %s\n", sexp.AdvancedForm(rule1))

	// Rule 2: Users can read or write (but not delete)
	rule2 := sexp.NewList("permission",
		sexp.NewList("role", sexp.NewAtom("user")),
		sexp.NewList("action", &starform.Set{
			Elements: []sexp.Element{
				sexp.NewAtom("read"),
				sexp.NewAtom("write"),
			},
		}),
	)
	engine.AddRuleElement(rule2)
	fmt.Printf("Added rule: (permission (role user) (action (* set read write)))\n")

	// Test admin permissions
	fmt.Println("\nAdmin permissions:")
	adminActions := []string{"read", "write", "delete", "execute"}
	for _, action := range adminActions {
		query := sexp.NewList("permission",
			sexp.NewList("role", sexp.NewAtom("admin")),
			sexp.NewList("action", sexp.NewAtom(action)),
		)
		result := engine.QueryElement(query)
		status := "✓"
		if !result {
			status = "✗"
		}
		fmt.Printf("  %s: %v %s\n", action, result, status)
	}

	// Test user permissions
	fmt.Println("\nUser permissions:")
	for _, action := range adminActions {
		query := sexp.NewList("permission",
			sexp.NewList("role", sexp.NewAtom("user")),
			sexp.NewList("action", sexp.NewAtom(action)),
		)
		result := engine.QueryElement(query)
		status := "✗"
		if result {
			status = "✓"
		}
		fmt.Printf("  %s: %v %s\n", action, result, status)
	}
	fmt.Println()
}
