// SPOCP client - Interactive TCP clientpackage spocpclient

package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirosfoundation/go-spocp/pkg/client"
)

func main() {
	var (
		address    = flag.String("addr", "localhost:6000", "Server address (host:port)")
		useTLS     = flag.Bool("tls", false, "Use TLS")
		skipVerify = flag.Bool("insecure", false, "Skip TLS certificate verification")
		query      = flag.String("query", "", "Execute single query and exit")
		addRule    = flag.String("add", "", "Add single rule and exit")
	)

	flag.Parse()

	// Setup TLS if requested
	var tlsConfig *tls.Config
	if *useTLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: *skipVerify,
		}
	}

	// Create client
	config := &client.Config{
		Address:   *address,
		TLSConfig: tlsConfig,
	}

	c, err := client.NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	fmt.Printf("Connected to %s\n", *address)

	// Single command mode
	if *query != "" {
		result, err := c.QueryString(*query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
			os.Exit(1)
		}
		if result {
			fmt.Println("OK - Query matched")
		} else {
			fmt.Println("DENIED - Query did not match")
		}
		return
	}

	if *addRule != "" {
		err := c.AddString(*addRule)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Add failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rule added successfully")
		return
	}

	// Interactive mode
	fmt.Println("SPOCP Client - Interactive Mode")
	fmt.Println("Commands:")
	fmt.Println("  query <s-expression>  - Query a rule")
	fmt.Println("  add <s-expression>    - Add a rule")
	fmt.Println("  reload                - Reload server rules")
	fmt.Println("  quit                  - Exit")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		cmd := strings.ToLower(parts[0])

		switch cmd {
		case "quit", "exit", "q":
			fmt.Println("Goodbye!")
			return

		case "query":
			if len(parts) < 2 {
				fmt.Println("Error: query requires an S-expression argument")
				continue
			}
			result, err := c.QueryString(parts[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			if result {
				fmt.Println("✓ OK - Query matched")
			} else {
				fmt.Println("✗ DENIED - Query did not match")
			}

		case "add":
			if len(parts) < 2 {
				fmt.Println("Error: add requires an S-expression argument")
				continue
			}
			err := c.AddString(parts[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Println("✓ Rule added successfully")

		case "reload":
			err := c.Reload()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Println("✓ Server rules reloaded")

		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			fmt.Println("Use: query, add, reload, or quit")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}
