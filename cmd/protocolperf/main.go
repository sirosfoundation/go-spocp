// Package main provides protocol performance comparison: Direct Engine vs TCP vs HTTP/AuthZen.
//
// This tool compares the performance overhead of different access methods to the SPOCP engine:
//   - Direct: Library calls to the engine (baseline)
//   - TCP: Using the SPOCP TCP protocol via client library
//   - HTTP: Using the AuthZen HTTP endpoint
//
// All tests use the same rules and queries to ensure fair comparison.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirosfoundation/go-spocp"
	"github.com/sirosfoundation/go-spocp/pkg/authzen"
	"github.com/sirosfoundation/go-spocp/pkg/client"
	"github.com/sirosfoundation/go-spocp/pkg/httpserver"
	"github.com/sirosfoundation/go-spocp/pkg/server"
	"github.com/sirosfoundation/go-spocp/pkg/sexp"
)

// TestCase represents a single query test case
type TestCase struct {
	Query       sexp.Element
	AuthZenReq  authzen.EvaluationRequest
	QueryString string
}

// BenchmarkResult holds performance results for a single test
type BenchmarkResult struct {
	Name          string
	Duration      time.Duration
	Queries       int
	QueriesPerSec float64
	AvgLatency    time.Duration
	Matches       int
	MatchRate     float64
}

func main() {
	numRules := flag.Int("rules", 1000, "Number of rules to generate")
	numQueries := flag.Int("queries", 10000, "Number of queries to run")
	numConcurrent := flag.Int("concurrent", 1, "Number of concurrent clients (for TCP/HTTP)")
	warmup := flag.Int("warmup", 100, "Number of warmup queries")
	tcpPort := flag.Int("tcp-port", 16000, "TCP server port")
	httpPort := flag.Int("http-port", 18000, "HTTP server port")
	skipTCP := flag.Bool("skip-tcp", false, "Skip TCP benchmark")
	skipHTTP := flag.Bool("skip-http", false, "Skip HTTP benchmark")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     SPOCP Protocol Performance Comparison                     ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Rules: %d, Queries: %d, Concurrent: %d               \n", *numRules, *numQueries, *numConcurrent)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create shared engine with rules
	fmt.Println("📦 Generating rules and test cases...")
	engine := spocp.NewEngine()
	testCases := generateRulesAndTestCases(engine, *numRules, *numQueries+*warmup)
	fmt.Printf("   Generated %d rules, %d test cases\n\n", engine.RuleCount(), len(testCases))

	results := make([]BenchmarkResult, 0)

	// ═══════════════════════════════════════════════════════════════════════
	// Benchmark 1: Direct Engine Access (baseline)
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("🔧 Benchmark 1: Direct Engine Access (baseline)")
	fmt.Println("   No network, no serialization - pure engine performance")

	// Warmup
	for i := 0; i < *warmup; i++ {
		engine.QueryElement(testCases[i].Query)
	}

	queries := testCases[*warmup:]
	start := time.Now()
	matches := 0
	for _, tc := range queries {
		if engine.QueryElement(tc.Query) {
			matches++
		}
	}
	duration := time.Since(start)

	directResult := BenchmarkResult{
		Name:          "Direct Engine",
		Duration:      duration,
		Queries:       len(queries),
		QueriesPerSec: float64(len(queries)) / duration.Seconds(),
		AvgLatency:    duration / time.Duration(len(queries)),
		Matches:       matches,
		MatchRate:     float64(matches) * 100 / float64(len(queries)),
	}
	results = append(results, directResult)
	printResult(directResult, *verbose)

	// ═══════════════════════════════════════════════════════════════════════
	// Benchmark 2: TCP Protocol
	// ═══════════════════════════════════════════════════════════════════════
	if !*skipTCP {
		fmt.Println("\n🔌 Benchmark 2: TCP Protocol")
		fmt.Println("   SPOCP binary protocol over TCP socket")

		tcpResult := runTCPBenchmark(engine, testCases, *tcpPort, *warmup, *numConcurrent, *verbose)
		if tcpResult != nil {
			results = append(results, *tcpResult)
			printResult(*tcpResult, *verbose)
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// Benchmark 3: HTTP/AuthZen Protocol
	// ═══════════════════════════════════════════════════════════════════════
	if !*skipHTTP {
		fmt.Println("\n🌐 Benchmark 3: HTTP/AuthZen Protocol")
		fmt.Println("   JSON over HTTP (AuthZen API 1.0)")

		httpResult := runHTTPBenchmark(engine, testCases, *httpPort, *warmup, *numConcurrent, *verbose)
		if httpResult != nil {
			results = append(results, *httpResult)
			printResult(*httpResult, *verbose)
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// Summary
	// ═══════════════════════════════════════════════════════════════════════
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     PERFORMANCE SUMMARY                       ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")

	baseline := results[0].QueriesPerSec
	for _, r := range results {
		overhead := ((baseline / r.QueriesPerSec) - 1) * 100
		if r.Name == "Direct Engine" {
			fmt.Printf("║  %-16s: %10.0f q/s  %8s latency  (baseline)\n",
				r.Name, r.QueriesPerSec, r.AvgLatency.Round(time.Microsecond))
		} else {
			fmt.Printf("║  %-16s: %10.0f q/s  %8s latency  (+%.1f%% overhead)\n",
				r.Name, r.QueriesPerSec, r.AvgLatency.Round(time.Microsecond), overhead)
		}
	}
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

	// Detailed overhead analysis
	fmt.Println("\n📊 Overhead Analysis:")
	for i := 1; i < len(results); i++ {
		overhead := results[0].AvgLatency
		protocolLatency := results[i].AvgLatency - overhead
		fmt.Printf("   %s: +%s per query (%.1fx slower than direct)\n",
			results[i].Name, protocolLatency.Round(time.Microsecond),
			float64(results[i].AvgLatency)/float64(results[0].AvgLatency))
	}
}

// ruleData holds rule components for reuse as queries
type ruleData struct {
	resType, resID, action, userType, userID string
}

// generateRulesAndTestCases creates rules in the engine and generates test cases
func generateRulesAndTestCases(engine *spocp.Engine, numRules, numTestCases int) []TestCase {
	rnd := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility

	// Resource types and actions for AuthZen-compatible rules
	resourceTypes := []string{"account", "document", "file", "user", "project"}
	actions := []string{"can_read", "can_write", "can_delete", "can_update", "can_create"}
	userTypes := []string{"user", "service", "admin"}

	// Store rules for later use as queries (to ensure matches)
	rules := make([]ruleData, 0, numRules)

	// Generate rules
	for i := 0; i < numRules; i++ {
		resType := resourceTypes[rnd.Intn(len(resourceTypes))]
		resID := fmt.Sprintf("%d", rnd.Intn(1000))
		action := actions[rnd.Intn(len(actions))]
		userType := userTypes[rnd.Intn(len(userTypes))]
		userID := fmt.Sprintf("user%d", rnd.Intn(100))

		// Create rule that matches AuthZen structure
		rule := sexp.NewList(resType,
			sexp.NewList("id", sexp.NewAtom(resID)),
			sexp.NewList("action", sexp.NewAtom(action)),
			sexp.NewList("subject",
				sexp.NewList("type", sexp.NewAtom(userType)),
				sexp.NewList("id", sexp.NewAtom(userID)),
			),
		)
		engine.AddRuleElement(rule)

		// Store rules for later use as queries (to ensure matches)
		rules = append(rules, ruleData{resType, resID, action, userType, userID})
	}

	// Generate test cases - 50% from rules (guaranteed matches) + 50% random (likely non-matches)
	testCases := make([]TestCase, numTestCases)
	for i := 0; i < numTestCases; i++ {
		var resType, resID, action, userType, userID string

		if rnd.Float32() < 0.5 && len(rules) > 0 {
			// Use an existing rule (guaranteed match)
			r := rules[rnd.Intn(len(rules))]
			resType, resID, action, userType, userID = r.resType, r.resID, r.action, r.userType, r.userID
		} else {
			// Random query (likely no match)
			resType = resourceTypes[rnd.Intn(len(resourceTypes))]
			resID = fmt.Sprintf("%d", rnd.Intn(1000))
			action = actions[rnd.Intn(len(actions))]
			userType = userTypes[rnd.Intn(len(userTypes))]
			userID = fmt.Sprintf("user%d", rnd.Intn(100))
		}

		// SPOCP query
		query := sexp.NewList(resType,
			sexp.NewList("id", sexp.NewAtom(resID)),
			sexp.NewList("action", sexp.NewAtom(action)),
			sexp.NewList("subject",
				sexp.NewList("type", sexp.NewAtom(userType)),
				sexp.NewList("id", sexp.NewAtom(userID)),
			),
		)

		// AuthZen request
		authzenReq := authzen.EvaluationRequest{
			Subject: authzen.Subject{
				Type: userType,
				ID:   userID,
			},
			Resource: authzen.Resource{
				Type: resType,
				ID:   resID,
			},
			Action: authzen.Action{
				Name: action,
			},
		}

		testCases[i] = TestCase{
			Query:       query,
			AuthZenReq:  authzenReq,
			QueryString: query.String(),
		}
	}

	return testCases
}

// runTCPBenchmark runs the TCP protocol benchmark
func runTCPBenchmark(engine *spocp.Engine, testCases []TestCase, port, warmup, concurrent int, verbose bool) *BenchmarkResult {
	tcpAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Check if port is available
	ln, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		fmt.Printf("   ❌ Port %d not available: %v\n", port, err)
		return nil
	}
	ln.Close()

	// Start TCP server
	tcpConfig := &server.Config{
		Address:  tcpAddr,
		Engine:   engine,
		LogLevel: server.LogLevelSilent,
	}

	tcpServer, err := server.NewServer(tcpConfig)
	if err != nil {
		fmt.Printf("   ❌ Failed to create TCP server: %v\n", err)
		return nil
	}

	// Start server in background
	serverReady := make(chan struct{})
	go func() {
		close(serverReady)
		_ = tcpServer.Serve() //nolint:errcheck // benchmark server
	}()
	<-serverReady
	time.Sleep(50 * time.Millisecond) // Give server time to start

	defer tcpServer.Close()

	queries := testCases[warmup:]

	if concurrent == 1 {
		// Single client benchmark
		cli, err := client.NewClient(&client.Config{
			Address: tcpAddr,
			Timeout: 5 * time.Second,
		})
		if err != nil {
			fmt.Printf("   ❌ Failed to connect: %v\n", err)
			return nil
		}
		defer cli.Close()

		// Warmup
		for i := 0; i < warmup; i++ {
			_, _ = cli.Query(testCases[i].Query) //nolint:errcheck // warmup
		}

		start := time.Now()
		matches := 0
		for _, tc := range queries {
			result, err := cli.Query(tc.Query)
			if err != nil {
				if verbose {
					fmt.Printf("   ⚠️  Query error: %v\n", err)
				}
				continue
			}
			if result {
				matches++
			}
		}
		duration := time.Since(start)

		return &BenchmarkResult{
			Name:          "TCP (1 client)",
			Duration:      duration,
			Queries:       len(queries),
			QueriesPerSec: float64(len(queries)) / duration.Seconds(),
			AvgLatency:    duration / time.Duration(len(queries)),
			Matches:       matches,
			MatchRate:     float64(matches) * 100 / float64(len(queries)),
		}
	}

	// Multi-client benchmark
	var wg sync.WaitGroup
	var totalMatches int64
	var totalErrors int64

	queriesPerClient := len(queries) / concurrent
	start := time.Now()

	for c := 0; c < concurrent; c++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			cli, err := client.NewClient(&client.Config{
				Address: tcpAddr,
				Timeout: 5 * time.Second,
			})
			if err != nil {
				atomic.AddInt64(&totalErrors, int64(queriesPerClient))
				return
			}
			defer cli.Close()

			startIdx := clientID * queriesPerClient
			endIdx := startIdx + queriesPerClient
			if clientID == concurrent-1 {
				endIdx = len(queries)
			}

			for i := startIdx; i < endIdx; i++ {
				result, err := cli.Query(queries[i].Query)
				if err != nil {
					atomic.AddInt64(&totalErrors, 1)
					continue
				}
				if result {
					atomic.AddInt64(&totalMatches, 1)
				}
			}
		}(c)
	}

	wg.Wait()
	duration := time.Since(start)

	return &BenchmarkResult{
		Name:          fmt.Sprintf("TCP (%d clients)", concurrent),
		Duration:      duration,
		Queries:       len(queries),
		QueriesPerSec: float64(len(queries)) / duration.Seconds(),
		AvgLatency:    duration / time.Duration(len(queries)),
		Matches:       int(totalMatches),
		MatchRate:     float64(totalMatches) * 100 / float64(len(queries)),
	}
}

// runHTTPBenchmark runs the HTTP/AuthZen protocol benchmark
func runHTTPBenchmark(engine *spocp.Engine, testCases []TestCase, port, warmup, concurrent int, verbose bool) *BenchmarkResult {
	httpAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Check if port is available
	ln, err := net.Listen("tcp", httpAddr)
	if err != nil {
		fmt.Printf("   ❌ Port %d not available: %v\n", port, err)
		return nil
	}
	ln.Close()

	// Create mutex for engine access
	var mu sync.RWMutex

	// Start HTTP server
	httpConfig := &httpserver.Config{
		Address:       httpAddr,
		EnableAuthZen: true,
		Engine:        engine,
		EngineMutex:   &mu,
		LogLevel:      server.LogLevelSilent,
	}

	httpServer, err := httpserver.NewHTTPServer(httpConfig)
	if err != nil {
		fmt.Printf("   ❌ Failed to create HTTP server: %v\n", err)
		return nil
	}

	if err := httpServer.Start(); err != nil {
		fmt.Printf("   ❌ Failed to start HTTP server: %v\n", err)
		return nil
	}
	time.Sleep(50 * time.Millisecond) // Give server time to start

	defer httpServer.Close()

	// Create HTTP client with connection pooling
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        concurrent * 2,
			MaxIdleConnsPerHost: concurrent * 2,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 5 * time.Second,
	}

	endpoint := fmt.Sprintf("http://%s/access/v1/evaluation", httpAddr)
	queries := testCases[warmup:]

	if concurrent == 1 {
		// Single client benchmark

		// Warmup
		for i := 0; i < warmup; i++ {
			_, _ = sendAuthZenRequest(httpClient, endpoint, testCases[i].AuthZenReq) //nolint:errcheck // warmup
		}

		start := time.Now()
		matches := 0
		for _, tc := range queries {
			result, err := sendAuthZenRequest(httpClient, endpoint, tc.AuthZenReq)
			if err != nil {
				if verbose {
					fmt.Printf("   ⚠️  HTTP error: %v\n", err)
				}
				continue
			}
			if result {
				matches++
			}
		}
		duration := time.Since(start)

		return &BenchmarkResult{
			Name:          "HTTP (1 client)",
			Duration:      duration,
			Queries:       len(queries),
			QueriesPerSec: float64(len(queries)) / duration.Seconds(),
			AvgLatency:    duration / time.Duration(len(queries)),
			Matches:       matches,
			MatchRate:     float64(matches) * 100 / float64(len(queries)),
		}
	}

	// Multi-client benchmark
	var wg sync.WaitGroup
	var totalMatches int64
	var totalErrors int64

	queriesPerClient := len(queries) / concurrent
	start := time.Now()

	for c := 0; c < concurrent; c++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			startIdx := clientID * queriesPerClient
			endIdx := startIdx + queriesPerClient
			if clientID == concurrent-1 {
				endIdx = len(queries)
			}

			for i := startIdx; i < endIdx; i++ {
				result, err := sendAuthZenRequest(httpClient, endpoint, queries[i].AuthZenReq)
				if err != nil {
					atomic.AddInt64(&totalErrors, 1)
					continue
				}
				if result {
					atomic.AddInt64(&totalMatches, 1)
				}
			}
		}(c)
	}

	wg.Wait()
	duration := time.Since(start)

	return &BenchmarkResult{
		Name:          fmt.Sprintf("HTTP (%d clients)", concurrent),
		Duration:      duration,
		Queries:       len(queries),
		QueriesPerSec: float64(len(queries)) / duration.Seconds(),
		AvgLatency:    duration / time.Duration(len(queries)),
		Matches:       int(totalMatches),
		MatchRate:     float64(totalMatches) * 100 / float64(len(queries)),
	}
}

// sendAuthZenRequest sends an AuthZen evaluation request
func sendAuthZenRequest(client *http.Client, endpoint string, req authzen.EvaluationRequest) (bool, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return false, err
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result authzen.EvaluationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, err
	}

	return result.Decision, nil
}

// printResult prints a benchmark result
func printResult(r BenchmarkResult, verbose bool) {
	fmt.Printf("   ✓ %s:\n", r.Name)
	fmt.Printf("     Queries/sec: %.0f\n", r.QueriesPerSec)
	fmt.Printf("     Avg latency: %s\n", r.AvgLatency.Round(time.Microsecond))
	fmt.Printf("     Match rate:  %.1f%% (%d/%d)\n", r.MatchRate, r.Matches, r.Queries)
	if verbose {
		fmt.Printf("     Total time:  %s\n", r.Duration.Round(time.Millisecond))
	}
}

func init() {
	// Suppress stdout unless running
	if len(os.Args) > 1 && os.Args[1] == "-h" {
		return
	}
}
