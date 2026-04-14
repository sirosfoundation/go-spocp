// SPOCP server - TCP and HTTP/AuthZen server with dynamic rule loading
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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

	// Build Zap logger
	var logger *zap.Logger
	if *logLevel == "silent" {
		logger = zap.NewNop()
	} else {
		var zapLevel zapcore.Level
		switch *logLevel {
		case "error":
			zapLevel = zapcore.ErrorLevel
		case "warn":
			zapLevel = zapcore.WarnLevel
		case "info":
			zapLevel = zapcore.InfoLevel
		case "debug":
			zapLevel = zapcore.DebugLevel
		default:
			fmt.Fprintf(os.Stderr, "Invalid log level: %s (must be: silent, error, warn, info, debug)\n", *logLevel)
			os.Exit(1)
		}

		zapCfg := zap.NewProductionConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(zapLevel)
		zapCfg.EncoderConfig.TimeKey = "ts"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		var err error
		logger, err = zapCfg.Build()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Sync() //nolint:errcheck
	}

	// Setup TLS if certificates are provided
	var tlsConfig *tls.Config
	if *tlsCert != "" && *tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
		if err != nil {
			logger.Fatal("Failed to load TLS certificates", zap.Error(err))
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		logger.Info("TLS enabled for TCP server")
	} else if *tlsCert != "" || *tlsKey != "" {
		logger.Fatal("Both -tls-cert and -tls-key must be specified for TLS")
	}

	var srv *server.Server
	var httpSrv *httpserver.HTTPServer
	var err error

	// Create TCP server if enabled
	if *tcpEnabled {
		config := &server.Config{
			Address:        *tcpAddress,
			RulesDir:       *rulesDir,
			TLSConfig:      tlsConfig,
			ReloadInterval: *reloadInterval,
			PidFile:        *pidFile,
			Logger:         logger.Named("tcp"),
		}

		srv, err = server.NewServer(config)
		if err != nil {
			logger.Fatal("Failed to create TCP server", zap.Error(err))
		}
	}

	// Always create HTTP server (for monitoring)
	httpConfig := &httpserver.Config{
		Address:       *httpAddress,
		EnableAuthZen: *authzenEnabled,
		Logger:        logger.Named("http"),
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

	httpSrv, err = httpserver.NewHTTPServer(httpConfig)
	if err != nil {
		if srv != nil {
			srv.Close()
		}
		logger.Fatal("Failed to create HTTP server", zap.Error(err))
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Shutdown handler
	shutdownComplete := make(chan struct{})
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		if srv != nil {
			srv.Close()
		}
		if httpSrv != nil {
			httpSrv.Close()
		}
		close(shutdownComplete)
	}()

	// Start servers
	logger.Info("SPOCP Server starting",
		zap.String("rules_dir", *rulesDir),
		zap.Bool("tcp", *tcpEnabled),
		zap.String("tcp_addr", *tcpAddress),
		zap.String("http_addr", *httpAddress),
		zap.Bool("authzen", *authzenEnabled),
		zap.Duration("reload_interval", *reloadInterval),
		zap.String("log_level", *logLevel),
	)

	// Always start HTTP server in background
	if err := httpSrv.Start(); err != nil {
		logger.Fatal("HTTP server error", zap.Error(err))
	}

	// Start TCP server (blocking) if enabled, otherwise wait for shutdown
	if *tcpEnabled && srv != nil {
		if err := srv.Serve(); err != nil {
			logger.Fatal("TCP server error", zap.Error(err))
		}
	} else {
		// No TCP server: wait for shutdown signal
		<-shutdownComplete
	}
}
