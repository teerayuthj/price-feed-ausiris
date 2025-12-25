package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gold-socket/internal/api"
	"gold-socket/internal/config"
	"gold-socket/internal/redis"
	"gold-socket/internal/scheduler"
	"gold-socket/internal/sftp"
	"gold-socket/internal/websocket"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Check command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
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
	log.Println("Starting Gold Socket WebSocket Server")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Could not load full config: %v", err)
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
			log.Printf("Warning: Redis connection failed: %v (continuing without Redis)", err)
			redisClient = nil
		} else {
			log.Printf("Connected to Redis at %s", cfg.Redis.Addr)
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
		log.Printf("Scheduled downloads enabled every %v", cfg.Schedule.DownloadInterval)
	} else {
		log.Println("SFTP not configured, scheduled downloads disabled")
	}

	// Create and start HTTP server
	server := api.NewServer(&cfg.Server, hub)

	// Start server in goroutine
	go func() {
		log.Printf("WebSocket server starting on :%s", cfg.Server.Port)
		log.Printf("WebSocket endpoint: ws://localhost:%s/ws", cfg.Server.Port)
		log.Printf("USD Rate API: http://localhost:%s/api/data", cfg.Server.Port)
		log.Printf("Market Data API: http://localhost:%s/api/market-data", cfg.Server.Port)
		log.Printf("Health Check: http://localhost:%s/health", cfg.Server.Port)
		log.Printf("Web interface: http://localhost:%s", cfg.Server.Port)

		if err := server.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
}

// runDownload performs a one-time SFTP download
func runDownload() {
	log.Println("Running one-time SFTP download")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Connecting to SFTP server: %s:%d", cfg.SFTP.Host, cfg.SFTP.Port)
	log.Printf("Remote path 1: %s", cfg.SFTP.RemotePath)
	if cfg.SFTP.RemotePath2 != "" {
		log.Printf("Remote path 2: %s", cfg.SFTP.RemotePath2)
	}
	log.Printf("Local path: %s", cfg.SFTP.LocalPath)

	err = sftp.DownloadFilesWithConfig(&cfg.SFTP)
	if err != nil {
		log.Fatalf("SFTP download failed: %v", err)
	}

	log.Println("SFTP download completed successfully!")
}

// runContinuous runs continuous downloads without WebSocket server
func runContinuous() {
	log.Println("Starting continuous SFTP downloads (no WebSocket server)")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched := scheduler.New(&cfg.SFTP, cfg.Schedule.DownloadInterval, nil)
	sched.Start(ctx)

	log.Printf("Continuous downloads running every %v", cfg.Schedule.DownloadInterval)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
}

// printUsage prints command usage
func printUsage() {
	fmt.Printf("Gold Socket - Real-time Exchange Rate Monitor\n\n")
	fmt.Printf("Usage: %s [command]\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Println("  (none)      Start WebSocket server with scheduled downloads (default)")
	fmt.Println("  download    One-time SFTP download")
	fmt.Println("  continuous  Continuous SFTP downloads (no WebSocket)")
	fmt.Println("  help        Show this help message")
}
