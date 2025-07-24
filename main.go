package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	RemotePath  string
	RemotePath2 string
	LocalPath   string
}

func loadConfig() (*SFTPConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	port, err := strconv.Atoi(os.Getenv("SFTP_PORT"))
	if err != nil {
		port = 22 // default SSH/SFTP port
	}

	config := &SFTPConfig{
		Host:        os.Getenv("SFTP_HOST"),
		Port:        port,
		User:        os.Getenv("SFTP_USER"),
		Password:    os.Getenv("SFTP_PASSWORD"),
		RemotePath:  os.Getenv("SFTP_REMOTE_PATH"),
		RemotePath2: os.Getenv("SFTP_REMOTE_PATH2"),
		LocalPath:   os.Getenv("SFTP_LOCAL_PATH"),
	}

	if config.Host == "" || config.User == "" || config.Password == "" {
		return nil, fmt.Errorf("missing required SFTP configuration")
	}

	return config, nil
}

func connectSFTP(config *SFTPConfig) (*sftp.Client, error) {
	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         5 * time.Second,
	}

	// Connect to SSH server
	conn, err := ssh.Dial("tcp", net.JoinHostPort(config.Host, strconv.Itoa(config.Port)), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %v", err)
	}

	return sftpClient, nil
}

func downloadFile(c *sftp.Client, remotePath, localPath string) error {
	// Open remote file
	remoteFile, err := c.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %v", remotePath, err)
	}
	defer remoteFile.Close()

	// Create local directory if it doesn't exist
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %v", err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	// Copy file content
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	return nil
}

func downloadSingleFile(c *sftp.Client, remotePath, localPath, filename string) error {
	// Download to temporary file first
	tempPath := localPath + ".tmp"
	
	fmt.Printf("Downloading: %s -> %s\n", remotePath, tempPath)
	
	err := downloadFile(c, remotePath, tempPath)
	if err != nil {
		return fmt.Errorf("error downloading %s: %v", remotePath, err)
	}

	// Validate server data before overwriting local file
	shouldKeep, reason := ValidateServerData(tempPath, localPath)
	
	if shouldKeep {
		// Move temp file to final location
		err = os.Rename(tempPath, localPath)
		if err != nil {
			os.Remove(tempPath) // Clean up temp file
			return fmt.Errorf("error moving temp file: %v", err)
		}
		fmt.Printf("Successfully updated: %s (%s)\n", filename, reason)
	} else {
		// Remove temp file and keep existing local file
		os.Remove(tempPath)
		fmt.Printf("Kept existing file: %s (%s)\n", filename, reason)
	}
	
	return nil
}

func downloadMultipleFiles(config *SFTPConfig) error {
	return downloadMultipleFilesWithValidation(config, true, true)
}

func downloadMultipleFilesWithValidation(config *SFTPConfig, downloadFile1, downloadFile2 bool) error {
	c, err := connectSFTP(config)
	if err != nil {
		return err
	}
	defer c.Close()

	var filesToDownload []struct {
		remotePath string
		filename   string
	}

	// Add first file if needed and exists
	if downloadFile1 && config.RemotePath != "" {
		filesToDownload = append(filesToDownload, struct {
			remotePath string
			filename   string
		}{
			remotePath: config.RemotePath,
			filename:   filepath.Base(config.RemotePath),
		})
	}

	// Add second file if needed and exists
	if downloadFile2 && config.RemotePath2 != "" {
		filesToDownload = append(filesToDownload, struct {
			remotePath string
			filename   string
		}{
			remotePath: config.RemotePath2,
			filename:   filepath.Base(config.RemotePath2),
		})
	}

	if len(filesToDownload) == 0 {
		log.Printf("No files need to be downloaded")
		return nil
	}

	downloadCount := 0
	for _, file := range filesToDownload {
		localPath := filepath.Join(config.LocalPath, file.filename)
		
		err := downloadSingleFile(c, file.remotePath, localPath, file.filename)
		if err != nil {
			log.Printf("Error downloading %s: %v", file.remotePath, err)
			continue
		}
		
		// Validate the downloaded file
		hasValid, err := ValidateFileData(localPath)
		if err != nil {
			log.Printf("Error validating downloaded file %s: %v", file.filename, err)
		} else if !hasValid {
			log.Printf("Warning: Downloaded file %s contains invalid or zero data", file.filename)
		} else {
			log.Printf("Downloaded file %s validated successfully", file.filename)
		}
		
		downloadCount++
	}

	if downloadCount == 0 {
		return fmt.Errorf("failed to download any files")
	}

	fmt.Printf("Downloaded %d files to %s\n", downloadCount, config.LocalPath)
	return nil
}

func listAndDownloadFiles(config *SFTPConfig) error {
	c, err := connectSFTP(config)
	if err != nil {
		return err
	}
	defer c.Close()

	// Check if RemotePath is a file or directory
	fileInfo, err := c.Stat(config.RemotePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote path %s: %v", config.RemotePath, err)
	}

	if !fileInfo.IsDir() {
		// It's a single file, download it directly
		filename := filepath.Base(config.RemotePath)
		localPath := filepath.Join(config.LocalPath, filename)
		return downloadSingleFile(c, config.RemotePath, localPath, filename)
	}

	// It's a directory, list and download all files
	entries, err := c.ReadDir(config.RemotePath)
	if err != nil {
		return fmt.Errorf("failed to list remote directory: %v", err)
	}

	fmt.Printf("Found %d items in remote directory\n", len(entries))

	downloadCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			remotePath := filepath.Join(config.RemotePath, entry.Name())
			localPath := filepath.Join(config.LocalPath, entry.Name())

			fmt.Printf("Downloading: %s -> %s\n", remotePath, localPath)
			
			err := downloadFile(c, remotePath, localPath)
			if err != nil {
				log.Printf("Error downloading %s: %v", remotePath, err)
				continue
			}

			downloadCount++
			fmt.Printf("Successfully downloaded: %s\n", entry.Name())
		}
	}

	if downloadCount == 0 {
		return fmt.Errorf("no files found to download")
	}

	fmt.Printf("Downloaded %d files to %s\n", downloadCount, config.LocalPath)
	return nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "download":
			// One-time SFTP download mode
			config, err := loadConfig()
			if err != nil {
				log.Fatal("Configuration error:", err)
			}

			fmt.Printf("Connecting to SFTP server: %s:%d\n", config.Host, config.Port)
			fmt.Printf("Remote path 1: %s\n", config.RemotePath)
			if config.RemotePath2 != "" {
				fmt.Printf("Remote path 2: %s\n", config.RemotePath2)
			}
			fmt.Printf("Local path: %s\n", config.LocalPath)

			err = downloadMultipleFiles(config)
			if err != nil {
				log.Fatal("SFTP operation failed:", err)
			}

			fmt.Println("SFTP download completed successfully!")

		case "continuous":
			// Continuous download mode only (no WebSocket server)
			config, err := loadConfig()
			if err != nil {
				log.Fatal("Configuration error:", err)
			}

			// Get download interval from environment (default: 30 seconds)
			err = godotenv.Load()
			if err != nil {
				log.Printf("Warning: Could not load .env file: %v", err)
			}

			intervalSeconds := 30
			if envInterval := os.Getenv("DOWNLOAD_INTERVAL_SECONDS"); envInterval != "" {
				if parsed, err := strconv.Atoi(envInterval); err == nil {
					intervalSeconds = parsed
				}
			}

			downloadInterval := time.Duration(intervalSeconds) * time.Second
			log.Printf("Starting continuous SFTP downloads every %v", downloadInterval)

			// Use a dummy hub for the scheduler
			downloader := NewScheduledDownloader(config, downloadInterval, nil)
			downloader.Start()

			// Keep the program running
			select {}

		default:
			fmt.Printf("Usage: %s [download|continuous|server]\n", os.Args[0])
			fmt.Println("  download    - One-time SFTP download")
			fmt.Println("  continuous  - Continuous SFTP downloads (no WebSocket)")
			fmt.Println("  server      - WebSocket server with scheduled downloads (default)")
			os.Exit(1)
		}
	} else {
		// WebSocket server mode with scheduled downloads (default)
		fmt.Println("Starting WebSocket server for Gold Price & Exchange Rate Monitor")
		fmt.Println("This includes automatic scheduled SFTP downloads")
		StartWebSocketServer()
	}
}