// SPOCP server - TCP server with dynamic rule loading
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirosfoundation/go-spocp/pkg/server"
)

func main() {
	var (
		address        = flag.String("addr", ":6000", "Address to listen on (host:port)")
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
		Address:        *address,
		RulesDir:       *rulesDir,
		TLSConfig:      tlsConfig,
		ReloadInterval: *reloadInterval,
		PidFile:        *pidFile,
		HealthAddr:     *healthAddr,
		Logger:         logger,
		LogLevel:       level,
	}

	srv, err := server.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		if level >= server.LogLevelInfo {
			logger.Println("[INFO] Received shutdown signal")
		}
		srv.Close()
	}()

	// Start server
	if level >= server.LogLevelInfo {
		logger.Printf("[INFO] SPOCP Server starting...")
		logger.Printf("[INFO]   Address: %s", *address)
		logger.Printf("[INFO]   Rules directory: %s", *rulesDir)
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

	if err := srv.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
