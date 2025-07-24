package main

import (
	"log"
	"time"
)

// ScheduledDownloader handles periodic SFTP downloads
type ScheduledDownloader struct {
	config   *SFTPConfig
	interval time.Duration
	hub      *Hub
	ticker   *time.Ticker
	stopChan chan bool
}

// NewScheduledDownloader creates a new scheduled downloader
func NewScheduledDownloader(config *SFTPConfig, interval time.Duration, hub *Hub) *ScheduledDownloader {
	return &ScheduledDownloader{
		config:   config,
		interval: interval,
		hub:      hub,
		stopChan: make(chan bool),
	}
}

// Start begins the scheduled download process
func (sd *ScheduledDownloader) Start() {
	log.Printf("Starting scheduled SFTP downloads every %v", sd.interval)
	
	// Perform initial download
	sd.performDownload()
	
	// Set up ticker for periodic downloads
	sd.ticker = time.NewTicker(sd.interval)
	
	go func() {
		for {
			select {
			case <-sd.ticker.C:
				sd.performDownload()
			case <-sd.stopChan:
				sd.ticker.Stop()
				log.Println("Scheduled downloader stopped")
				return
			}
		}
	}()
}

// Stop stops the scheduled downloader
func (sd *ScheduledDownloader) Stop() {
	if sd.ticker != nil {
		close(sd.stopChan)
	}
}

// performDownload executes the SFTP download with smart validation
func (sd *ScheduledDownloader) performDownload() {
	log.Printf("Checking for server updates...")
	
	// Always download to check for server updates (but validate before overwriting)
	err := downloadMultipleFiles(sd.config)
	if err != nil {
		log.Printf("Scheduled download failed: %v", err)
		return
	}
	
	// Update JSON file from exrate.txt
	err = UpdateUSDRateFromExrate()
	if err != nil {
		log.Printf("Failed to update JSON from exrate: %v", err)
	}
	
	// Trigger WebSocket broadcast if hub is available
	if sd.hub != nil {
		go sd.hub.BroadcastData()
	}
}

// StartScheduledDownloads starts the scheduled download service
func StartScheduledDownloads(hub *Hub, interval time.Duration) {
	config, err := loadConfig()
	if err != nil {
		log.Printf("Configuration error for scheduled downloads: %v", err)
		return
	}
	
	downloader := NewScheduledDownloader(config, interval, hub)
	downloader.Start()
	
	log.Printf("Scheduled downloads started with %v interval", interval)
}