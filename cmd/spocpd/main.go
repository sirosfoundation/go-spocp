// SPOCP server - TCP server with dynamic rule loadingpackage spocpd

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
	)

	flag.Parse()

	// Validate required arguments
	if *rulesDir == "" {
		fmt.Fprintf(os.Stderr, "Error: -rules directory is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

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
		log.Println("TLS enabled")
	} else if *tlsCert != "" || *tlsKey != "" {
		log.Fatal("Both -tls-cert and -tls-key must be specified for TLS")
	}

	// Create server
	config := &server.Config{
		Address:        *address,
		RulesDir:       *rulesDir,
		TLSConfig:      tlsConfig,
		ReloadInterval: *reloadInterval,
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
		log.Println("Received shutdown signal")
		srv.Close()
	}()

	// Start server
	log.Printf("SPOCP Server starting...")
	log.Printf("  Address: %s", *address)
	log.Printf("  Rules directory: %s", *rulesDir)
	if *reloadInterval > 0 {
		log.Printf("  Auto-reload: every %v", *reloadInterval)
	}

	if err := srv.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
