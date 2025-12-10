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
		tcpEnabled = flag.Bool("tcp", true, "Enable TCP server")
		tcpAddress = flag.String("addr", ":6000", "TCP address to listen on (host:port)")

		// HTTP/AuthZen options
		httpEnabled = flag.Bool("http", false, "Enable HTTP/AuthZen server")
		httpAddress = flag.String("http-addr", ":8000", "HTTP address to listen on (host:port)")

		// Common options
		rulesDir       = flag.String("rules", "", "Directory containing .spoc rule files (required)")
		tlsCert        = flag.String("tls-cert", "", "Path to TLS certificate file (optional)")
		tlsKey         = flag.String("tls-key", "", "Path to TLS private key file (optional)")
		reloadInterval = flag.Duration("reload", 0, "Auto-reload interval (e.g., 5m, 1h) - 0 to disable")
		pidFile        = flag.String("pid", "", "PID file path (optional)")
		healthAddr     = flag.String("health", "", "Health check address (e.g., :8080, optional)")
		logLevel       = flag.String("log", "error", "Log level: silent, error, warn, info, debug")
	)

	flag.Parse()

	// Validate required arguments
	if *rulesDir == "" {
		fmt.Fprintf(os.Stderr, "Error: -rules directory is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// At least one server must be enabled
	if !*tcpEnabled && !*httpEnabled {
		fmt.Fprintf(os.Stderr, "Error: at least one of -tcp or -http must be enabled\n\n")
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
			logger.Println("[INFO] TLS enabled")
		}
	} else if *tlsCert != "" || *tlsKey != "" {
		log.Fatal("Both -tls-cert and -tls-key must be specified for TLS")
	}

	// Create server
	config := &server.Config{
		Address:        *tcpAddress,
		RulesDir:       *rulesDir,
		TLSConfig:      tlsConfig,
		ReloadInterval: *reloadInterval,
		PidFile:        *pidFile,
		HealthAddr:     *healthAddr,
		Logger:         logger,
		LogLevel:       level,
	}

	var srv *server.Server
	var httpSrv *httpserver.HTTPServer

	// Create TCP server if enabled
	if *tcpEnabled {
		var err error
		srv, err = server.NewServer(config)
		if err != nil {
			log.Fatalf("Failed to create TCP server: %v", err)
		}
	}

	// Create HTTP server if enabled
	if *httpEnabled {
		httpConfig := &httpserver.Config{
			Address:  *httpAddress,
			Logger:   logger,
			LogLevel: level,
		}

		// Share engine from TCP server if available
		if srv != nil {
			httpConfig.Engine = srv.GetEngine()
			httpConfig.EngineMutex = srv.GetEngineMutex()
		} else {
			// HTTP-only mode: need to configure standalone
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
		}
		if *httpEnabled {
			logger.Printf("[INFO]   HTTP/AuthZen server: %s", *httpAddress)
		}
		if *reloadInterval > 0 {
			logger.Printf("[INFO]   Auto-reload: every %v", *reloadInterval)
		}
		if *pidFile != "" {
			logger.Printf("[INFO]   PID file: %s", *pidFile)
		}
		if *healthAddr != "" {
			logger.Printf("[INFO]   Health check: %s", *healthAddr)
		}
		logger.Printf("[INFO]   Log level: %s", *logLevel)
	}

	// Start HTTP server in background if enabled
	if *httpEnabled && httpSrv != nil {
		if err := httpSrv.Start(); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}

	// Start TCP server (blocking) if enabled, or wait for shutdown
	if *tcpEnabled && srv != nil {
		if err := srv.Serve(); err != nil {
			log.Fatalf("TCP server error: %v", err)
		}
	} else {
		// If only HTTP is enabled, wait for shutdown signal
		<-shutdownComplete
	}
}
