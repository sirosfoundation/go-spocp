// SPOCP server - TCP and HTTP/AuthZen server with dynamic rule loading
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirosfoundation/go-spocp/pkg/httpserver"
	"github.com/sirosfoundation/go-spocp/pkg/server"
)

func main() {
	var (
		// TCP options
		tcpEnabled = flag.Bool("tcp", false, "Enable TCP server")
		tcpAddress = flag.String("tcp-addr", ":6000", "TCP server address (host:port)")

		// HTTP options
		httpAddress    = flag.String("http-addr", ":8000", "HTTP server address for health/stats/metrics (and optionally AuthZen)")
		authzenEnabled = flag.Bool("authzen", false, "Enable AuthZen API endpoint on HTTP server")

		// Common options
		rulesDir       = flag.String("rules", "", "Directory containing .spoc rule files (required)")
		tlsCert        = flag.String("tls-cert", "", "Path to TLS certificate file for TCP server (optional)")
		tlsKey         = flag.String("tls-key", "", "Path to TLS private key file for TCP server (optional)")
		reloadInterval = flag.Duration("reload", 0, "Auto-reload interval (e.g., 5m, 1h) - 0 to disable")
		pidFile        = flag.String("pid", "", "PID file path (optional)")
		logLevel       = flag.String("log", "error", "Log level: silent, error, warn, info, debug")
	)

	flag.Parse()

	// Validate required arguments
	if *rulesDir == "" {
		fmt.Fprintf(os.Stderr, "Error: -rules directory is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// HTTP server is always started (for monitoring), but we need at least one protocol
	if !*tcpEnabled && !*authzenEnabled {
		fmt.Fprintf(os.Stderr, "Error: at least one of -tcp or -authzen must be enabled\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse log level
	var level server.LogLevel
	switch *logLevel {
	case "silent":
		level = server.LogLevelSilent
	case "error":
		level = server.LogLevelError
	case "warn":
		level = server.LogLevelWarn
	case "info":
		level = server.LogLevelInfo
	case "debug":
		level = server.LogLevelDebug
	default:
		log.Fatalf("Invalid log level: %s (must be: silent, error, warn, info, debug)", *logLevel)
	}

	// Setup logger
	logger := log.New(os.Stdout, "[SPOCP] ", log.LstdFlags)

	// Setup TLS if certificates are provided
	var tlsConfig *tls.Config
	if *tlsCert != "" && *tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			log.Fatalf("Failed to load TLS certificates: %v", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		if level >= server.LogLevelInfo {
			logger.Println("[INFO] TLS enabled for TCP server")
		}
	} else if *tlsCert != "" || *tlsKey != "" {
		log.Fatal("Both -tls-cert and -tls-key must be specified for TLS")
	}

	var srv *server.Server
	var httpSrv *httpserver.HTTPServer

	// Create TCP server if enabled
	if *tcpEnabled {
		config := &server.Config{
			Address:        *tcpAddress,
			RulesDir:       *rulesDir,
			TLSConfig:      tlsConfig,
			ReloadInterval: *reloadInterval,
			PidFile:        *pidFile,
			Logger:         logger,
			LogLevel:       level,
		}

		var err error
		srv, err = server.NewServer(config)
		if err != nil {
			log.Fatalf("Failed to create TCP server: %v", err)
		}
	}

	// Always create HTTP server (for monitoring)
	httpConfig := &httpserver.Config{
		Address:       *httpAddress,
		EnableAuthZen: *authzenEnabled,
		Logger:        logger,
		LogLevel:      level,
	}

	// Share engine from TCP server if available, otherwise create standalone
	if srv != nil {
		httpConfig.Engine = srv.GetEngine()
		httpConfig.EngineMutex = srv.GetEngineMutex()
	} else {
		// No TCP server: HTTP server manages its own engine
		httpConfig.RulesDir = *rulesDir
		httpConfig.ReloadInterval = *reloadInterval
		httpConfig.PidFile = *pidFile
	}

	var err error
	httpSrv, err = httpserver.NewHTTPServer(httpConfig)
	if err != nil {
		if srv != nil {
			srv.Close()
		}
		log.Fatalf("Failed to create HTTP server: %v", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Shutdown handler
	shutdownComplete := make(chan struct{})
	go func() {
		<-sigChan
		if level >= server.LogLevelInfo {
			logger.Println("[INFO] Received shutdown signal")
		}
		if srv != nil {
			srv.Close()
		}
		if httpSrv != nil {
			httpSrv.Close()
		}
		close(shutdownComplete)
	}()

	// Start servers
	if level >= server.LogLevelInfo {
		logger.Printf("[INFO] SPOCP Server starting...")
		logger.Printf("[INFO]   Rules directory: %s", *rulesDir)
		if *tcpEnabled {
			logger.Printf("[INFO]   TCP server: %s", *tcpAddress)
			if tlsConfig != nil {
				logger.Printf("[INFO]     TLS: enabled")
			}
		}
		logger.Printf("[INFO]   HTTP server: %s", *httpAddress)
		if *authzenEnabled {
			logger.Printf("[INFO]     AuthZen API: enabled")
		}
		logger.Printf("[INFO]     Health/Stats: always enabled")
		if *reloadInterval > 0 {
			logger.Printf("[INFO]   Auto-reload: every %v", *reloadInterval)
		}
		if *pidFile != "" {
			logger.Printf("[INFO]   PID file: %s", *pidFile)
		}
		logger.Printf("[INFO]   Log level: %s", *logLevel)
	}

	// Always start HTTP server in background
	if err := httpSrv.Start(); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}

	// Start TCP server (blocking) if enabled, otherwise wait for shutdown
	if *tcpEnabled && srv != nil {
		if err := srv.Serve(); err != nil {
			log.Fatalf("TCP server error: %v", err)
		}
	} else {
		// No TCP server: wait for shutdown signal
		<-shutdownComplete
	}
}
