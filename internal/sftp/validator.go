package sftp

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// FileInfo stores file metadata for comparison
type FileInfo struct {
	Path         string
	Hash         string
	Size         int64
	HasValidData bool
}

// GetFileHash calculates MD5 hash of a file
func GetFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetFileInfo returns file information including validation
func GetFileInfo(filePath string) (*FileInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	hash, err := GetFileHash(filePath)
	if err != nil {
		return nil, err
	}

	hasValidData, err := ValidateFileData(filePath)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:         filePath,
		Hash:         hash,
		Size:         stat.Size(),
		HasValidData: hasValidData,
	}, nil
}

// ValidateFileData checks if file contains valid non-zero data
func ValidateFileData(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}

	contentStr := string(content)
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")

	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return false, nil
	}

	if strings.Contains(filePath, "exrate.txt") {
		return validateExrateData(lines), nil
	} else if strings.Contains(filePath, "market_retail.txt") {
		return validateMarketRetailData(lines), nil
	}

	return true, nil
}

// validateExrateData validates exchange rate data
func validateExrateData(lines []string) bool {
	validLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		buyRate, err1 := strconv.ParseFloat(parts[1], 64)
		sellRate, err2 := strconv.ParseFloat(parts[2], 64)

		if err1 == nil && err2 == nil && buyRate > 0 && sellRate > 0 {
			validLines++
		}
	}

	return validLines > 0
}

// validateMarketRetailData validates gold price data
func validateMarketRetailData(lines []string) bool {
	validLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 15 {
			continue
		}

		usdBuy, err1 := strconv.ParseFloat(parts[0], 64)
		usdSell, err2 := strconv.ParseFloat(parts[1], 64)
		buyBar, err3 := strconv.ParseFloat(parts[3], 64)
		sellBar, err4 := strconv.ParseFloat(parts[4], 64)

		if err1 == nil && err2 == nil && err3 == nil && err4 == nil &&
			usdBuy > 0 && usdSell > 0 && buyBar > 0 && sellBar > 0 {
			validLines++
		}
	}

	return validLines > 0
}

// CompareFiles checks if two files are different
func CompareFiles(file1, file2 string) (bool, error) {
	info1, err := GetFileInfo(file1)
	if err != nil {
		return false, err
	}

	info2, err := GetFileInfo(file2)
	if err != nil {
		return false, err
	}

	return info1.Hash != info2.Hash || info1.Size != info2.Size, nil
}

// ShouldDownload determines if we should download based on file validation
func ShouldDownload(localFile string) (bool, string) {
	if _, err := os.Stat(localFile); os.IsNotExist(err) {
		return true, "Local file does not exist"
	}

	hasValid, err := ValidateFileData(localFile)
	if err != nil {
		return true, fmt.Sprintf("Error validating local file: %v", err)
	}

	if !hasValid {
		return true, "Local file contains invalid or zero data"
	}

	return true, "Checking server for updates"
}

// ValidateServerData validates downloaded data before deciding to keep it
func ValidateServerData(tempFile, localFile string) (bool, string) {
	hasValidServer, err := ValidateFileData(tempFile)
	if err != nil {
		return false, fmt.Sprintf("Error validating server data: %v", err)
	}

	if !hasValidServer {
		if _, err := os.Stat(localFile); os.IsNotExist(err) {
			return false, "Server data is invalid and no local file exists"
		}

		hasValidLocal, err := ValidateFileData(localFile)
		if err != nil || !hasValidLocal {
			return false, "Server data is invalid and local file is also invalid"
		}

		return false, "Server data is invalid, keeping existing valid local file"
	}

	return true, "Server data is valid, updating local file"
}
