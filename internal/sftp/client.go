package sftp

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gold-socket/internal/config"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client wraps SFTP client with configuration
type Client struct {
	config *config.SFTPConfig
	client *sftp.Client
	conn   *ssh.Client
}

// NewClient creates a new SFTP client
func NewClient(cfg *config.SFTPConfig) *Client {
	return &Client{
		config: cfg,
	}
}

// Connect establishes SFTP connection
func (c *Client) Connect() error {
	sshConfig := &ssh.ClientConfig{
		User: c.config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := net.JoinHostPort(c.config.Host, strconv.Itoa(c.config.Port))
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %v", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}

	c.conn = conn
	c.client = client
	return nil
}

// Close closes SFTP and SSH connections
func (c *Client) Close() error {
	if c.client != nil {
		c.client.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// DownloadFile downloads a single file from remote to local
func (c *Client) DownloadFile(remotePath, localPath string) error {
	remoteFile, err := c.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %v", remotePath, err)
	}
	defer remoteFile.Close()

	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %v", err)
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	return nil
}

// DownloadWithValidation downloads file and validates before replacing local
// Returns (wasUpdated, error) - wasUpdated is true if file was actually updated
func (c *Client) DownloadWithValidation(remotePath, localPath, filename string) (bool, error) {
	tempPath := localPath + ".tmp"

	log.Printf("Downloading: %s -> %s", remotePath, tempPath)

	err := c.DownloadFile(remotePath, tempPath)
	if err != nil {
		return false, fmt.Errorf("error downloading %s: %v", remotePath, err)
	}

	shouldUpdate, reason := ValidateServerData(tempPath, localPath)

	if shouldUpdate {
		err = os.Rename(tempPath, localPath)
		if err != nil {
			os.Remove(tempPath)
			return false, fmt.Errorf("error moving temp file: %v", err)
		}
		log.Printf("Updated: %s (%s)", filename, reason)
		return true, nil
	}

	os.Remove(tempPath)
	log.Printf("Kept existing: %s (%s)", filename, reason)
	return false, nil
}

func (c *Client) shouldDownload(remotePath, localPath string) (bool, string) {
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return true, "Local file does not exist"
	}

	hasValid, err := ValidateFileData(localPath)
	if err != nil {
		return true, fmt.Sprintf("Error validating local file: %v", err)
	}
	if !hasValid {
		return true, "Local file contains invalid or zero data"
	}

	remoteInfo, err := c.client.Stat(remotePath)
	if err != nil {
		return true, fmt.Sprintf("Could not stat remote file: %v", err)
	}

	localInfo, err := os.Stat(localPath)
	if err != nil {
		return true, fmt.Sprintf("Could not stat local file: %v", err)
	}

	if remoteInfo.Size() == localInfo.Size() {
		remoteMod := remoteInfo.ModTime()
		if remoteMod.IsZero() {
			return true, "Remote modification time unavailable"
		}
		if !remoteMod.After(localInfo.ModTime()) {
			return false, "Remote file unchanged"
		}
	}

	return true, "Remote file updated"
}

// DownloadMultipleFiles downloads configured remote files
func (c *Client) DownloadMultipleFiles() error {
	return c.DownloadFilesSelective(true, true)
}

// DownloadFilesSelective downloads files based on selection flags
func (c *Client) DownloadFilesSelective(downloadFile1, downloadFile2 bool) error {
	if err := c.Connect(); err != nil {
		return err
	}
	defer c.Close()

	type fileToDownload struct {
		remotePath string
		filename   string
	}

	var files []fileToDownload

	if downloadFile1 && c.config.RemotePath != "" {
		files = append(files, fileToDownload{
			remotePath: c.config.RemotePath,
			filename:   filepath.Base(c.config.RemotePath),
		})
	}

	if downloadFile2 && c.config.RemotePath2 != "" {
		files = append(files, fileToDownload{
			remotePath: c.config.RemotePath2,
			filename:   filepath.Base(c.config.RemotePath2),
		})
	}

	if len(files) == 0 {
		log.Printf("No files need to be downloaded")
		return nil
	}

	downloadCount := 0
	skippedCount := 0
	hadError := false
	for _, file := range files {
		localPath := filepath.Join(c.config.LocalPath, file.filename)

		shouldDownload, reason := c.shouldDownload(file.remotePath, localPath)
		if !shouldDownload {
			log.Printf("Skipping download: %s (%s)", file.filename, reason)
			skippedCount++
			continue
		}

		wasUpdated, err := c.DownloadWithValidation(file.remotePath, localPath, file.filename)
		if err != nil {
			log.Printf("Error downloading %s: %v", file.remotePath, err)
			hadError = true
			continue
		}

		if wasUpdated {
			downloadCount++
		}
	}

	if downloadCount == 0 && hadError {
		return fmt.Errorf("failed to download any files")
	}

	if downloadCount == 0 {
		log.Printf("No files updated (checked %d, skipped %d)", len(files)-skippedCount, skippedCount)
		return nil
	}

	log.Printf("Updated %d files in %s", downloadCount, c.config.LocalPath)
	return nil
}

// DownloadFilesWithConfig creates client and downloads files
func DownloadFilesWithConfig(cfg *config.SFTPConfig) error {
	client := NewClient(cfg)
	return client.DownloadMultipleFiles()
}
