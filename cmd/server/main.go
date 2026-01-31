package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gold-socket/internal/api"
	"gold-socket/internal/config"
	"gold-socket/internal/logger"
	"gold-socket/internal/redis"
	"gold-socket/internal/scheduler"
	"gold-socket/internal/sftp"
	"gold-socket/internal/websocket"
)

var logLevel string

func init() {
	flag.StringVar(&logLevel, "log-level", "info", "Log level: error, warn, info")
}

func main() {
	flag.Parse()

	// Set log level from flag
	if err := logger.SetLevelFromString(logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level: %v\n", err)
		os.Exit(1)
	}

	// Check command line arguments (after flags)
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "download":
			runDownload()
			return
		case "continuous":
			runContinuous()
			return
		case "help":
			printUsage()
			return
		}
	}

	// Default: run WebSocket server
	runServer()
}

// runServer starts the WebSocket server with scheduled downloads
func runServer() {
	logger.Println("Starting Gold Socket WebSocket Server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Warn("Could not load full config: %v", err)
		cfg = config.LoadWithoutValidation()
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis (optional)
	var redisClient *redis.Client
	if cfg.Redis.Enabled {
		redisClient, err = redis.NewClient(redis.Config{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err != nil {
			logger.Warn("Redis connection failed: %v (continuing without Redis)", err)
			redisClient = nil
		} else {
			logger.Info("Connected to Redis at %s", cfg.Redis.Addr)
			defer redisClient.Close()
		}
	}

	// Create WebSocket hub
	hub := websocket.NewHub(redisClient)
	go hub.Run(ctx)

	// Start scheduler for SFTP downloads
	if cfg.SFTP.Host != "" {
		sched := scheduler.New(&cfg.SFTP, cfg.Schedule.DownloadInterval, hub)
		sched.Start(ctx)
		logger.Info("Scheduled downloads enabled every %v", cfg.Schedule.DownloadInterval)
	} else {
		logger.Info("SFTP not configured, scheduled downloads disabled")
	}

	// Create and start HTTP server
	server := api.NewServer(&cfg.Server, hub)

	// Start server in goroutine
	go func() {
		logger.Printf("WebSocket server starting on :%s", cfg.Server.Port)
		logger.Printf("WebSocket endpoint: ws://localhost:%s/ws", cfg.Server.Port)
		logger.Printf("USD Rate API: http://localhost:%s/api/data", cfg.Server.Port)
		logger.Printf("Market Data API: http://localhost:%s/api/market-data", cfg.Server.Port)
		logger.Printf("Health Check: http://localhost:%s/health", cfg.Server.Port)
		logger.Printf("Web interface: http://localhost:%s", cfg.Server.Port)

		if err := server.Start(); err != nil {
			logger.Fatal("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Println("Shutting down...")
	cancel()
}

// runDownload performs a one-time SFTP download
func runDownload() {
	logger.Println("Running one-time SFTP download")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Configuration error: %v", err)
	}

	logger.Printf("Connecting to SFTP server: %s:%d", cfg.SFTP.Host, cfg.SFTP.Port)
	logger.Printf("Remote path 1: %s", cfg.SFTP.RemotePath)
	if cfg.SFTP.RemotePath2 != "" {
		logger.Printf("Remote path 2: %s", cfg.SFTP.RemotePath2)
	}
	logger.Printf("Local path: %s", cfg.SFTP.LocalPath)

	err = sftp.DownloadFilesWithConfig(&cfg.SFTP)
	if err != nil {
		logger.Fatal("SFTP download failed: %v", err)
	}

	logger.Println("SFTP download completed successfully!")
}

// runContinuous runs continuous downloads without WebSocket server
func runContinuous() {
	logger.Println("Starting continuous SFTP downloads (no WebSocket server)")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Configuration error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched := scheduler.New(&cfg.SFTP, cfg.Schedule.DownloadInterval, nil)
	sched.Start(ctx)

	logger.Printf("Continuous downloads running every %v", cfg.Schedule.DownloadInterval)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Println("Shutting down...")
	cancel()
}

// printUsage prints command usage
func printUsage() {
	fmt.Printf("Gold Socket - Real-time Exchange Rate Monitor\n\n")
	fmt.Printf("Usage: %s [options] [command]\n\n", os.Args[0])
	fmt.Println("Options:")
	fmt.Println("  -log-level string    Log level: error, warn, info (default \"info\")")
	fmt.Println("\nCommands:")
	fmt.Println("  (none)      Start WebSocket server with scheduled downloads (default)")
	fmt.Println("  download    One-time SFTP download")
	fmt.Println("  continuous  Continuous SFTP downloads (no WebSocket)")
	fmt.Println("  help        Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  gold-socket                    # Start server with info logs")
	fmt.Println("  gold-socket -log-level=warn    # Start server with warn+ logs")
	fmt.Println("  gold-socket -log-level=error download  # One-time download, errors only")
}
