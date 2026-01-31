package scheduler

import (
	"context"
	"time"

	"gold-socket/internal/config"
	"gold-socket/internal/logger"
	"gold-socket/internal/parser"
	"gold-socket/internal/sftp"
	"gold-socket/internal/websocket"
)

// Scheduler handles periodic SFTP downloads
type Scheduler struct {
	config   *config.SFTPConfig
	interval time.Duration
	hub      *websocket.Hub
	ticker   *time.Ticker
	stopChan chan struct{}
}

// New creates a new scheduler
func New(cfg *config.SFTPConfig, interval time.Duration, hub *websocket.Hub) *Scheduler {
	return &Scheduler{
		config:   cfg,
		interval: interval,
		hub:      hub,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduled download process
func (s *Scheduler) Start(ctx context.Context) {
	logger.Printf("Starting scheduled SFTP downloads every %v", s.interval)

	// Perform initial download
	s.performDownload()

	// Set up ticker for periodic downloads
	s.ticker = time.NewTicker(s.interval)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.performDownload()
			case <-s.stopChan:
				s.ticker.Stop()
				logger.Println("Scheduled downloader stopped")
				return
			case <-ctx.Done():
				s.ticker.Stop()
				logger.Println("Scheduled downloader stopped (context cancelled)")
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		close(s.stopChan)
	}
}

// performDownload executes the SFTP download with validation
func (s *Scheduler) performDownload() {
	logger.Info("Checking for server updates...")

	// Download files from SFTP
	err := sftp.DownloadFilesWithConfig(s.config)
	if err != nil {
		logger.Error("Scheduled download failed: %v", err)
		// Mark source as disconnected
		if s.hub != nil {
			s.hub.SetSourceDisconnected(err.Error())
			go s.hub.BroadcastMarketData()
		}
		return
	}

	// Mark source as connected (SFTP succeeded)
	if s.hub != nil {
		s.hub.SetSourceConnected()
	}

	// Update USD rate JSON from exrate.txt
	err = parser.UpdateUSDRateFromExrate()
	if err != nil {
		logger.Error("Failed to update JSON from exrate: %v", err)
	}

	// Process market retail data to JSON
	err = parser.ProcessMarketRetailData()
	if err != nil {
		logger.Error("Failed to process market retail data: %v", err)
	}

	// Trigger WebSocket broadcast if hub is available
	if s.hub != nil {
		go s.hub.BroadcastData()
	}
}

// StartScheduledDownloads starts the scheduled download service
func StartScheduledDownloads(ctx context.Context, hub *websocket.Hub, interval time.Duration) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	scheduler := New(&cfg.SFTP, interval, hub)
	scheduler.Start(ctx)

	logger.Printf("Scheduled downloads started with %v interval", interval)
	return nil
}
