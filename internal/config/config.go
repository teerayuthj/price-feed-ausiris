package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	SFTP     SFTPConfig
	Server   ServerConfig
	Redis    RedisConfig
	Schedule ScheduleConfig
}

// SFTPConfig holds SFTP connection settings
type SFTPConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	RemotePath  string
	RemotePath2 string
	LocalPath   string
}

// ServerConfig holds HTTP/WebSocket server settings
type ServerConfig struct {
	Port           string
	StaticDir      string
	DataDir        string
	AppEnv         string
	ServiceName    string
	PublicBaseURL  string
	AllowedOrigins []string
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Enabled  bool
}

// ScheduleConfig holds scheduler settings
type ScheduleConfig struct {
	DownloadInterval time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	sftpPort := getEnvInt("SFTP_PORT", 22)

	cfg := &Config{
		SFTP: SFTPConfig{
			Host:        getEnv("SFTP_HOST", ""),
			Port:        sftpPort,
			User:        getEnv("SFTP_USER", ""),
			Password:    getEnv("SFTP_PASSWORD", ""),
			RemotePath:  getEnv("SFTP_REMOTE_PATH", ""),
			RemotePath2: getEnv("SFTP_REMOTE_PATH2", ""),
			LocalPath:   getEnv("SFTP_LOCAL_PATH", "./raw-data"),
		},
		Server: ServerConfig{
			Port:           getEnv("WEBSOCKET_PORT", "8080"),
			StaticDir:      getEnv("STATIC_DIR", "./web/static"),
			DataDir:        getEnv("DATA_DIR", "./raw-data"),
			AppEnv:         getEnv("APP_ENV", "development"),
			ServiceName:    getEnv("SERVICE_NAME", "gold-socket"),
			PublicBaseURL:  getEnv("PUBLIC_BASE_URL", ""),
			AllowedOrigins: getEnvList("ALLOWED_ORIGINS"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			Enabled:  getEnv("REDIS_ENABLED", "false") == "true",
		},
		Schedule: ScheduleConfig{
			DownloadInterval: time.Duration(getEnvInt("DOWNLOAD_INTERVAL_SECONDS", 1)) * time.Second,
		},
	}

	// Validate required SFTP config
	if cfg.SFTP.Host == "" || cfg.SFTP.User == "" || cfg.SFTP.Password == "" {
		return nil, fmt.Errorf("missing required SFTP configuration (SFTP_HOST, SFTP_USER, SFTP_PASSWORD)")
	}

	return cfg, nil
}

// LoadWithoutValidation loads config without requiring SFTP credentials
func LoadWithoutValidation() *Config {
	_ = godotenv.Load()

	return &Config{
		SFTP: SFTPConfig{
			Host:        getEnv("SFTP_HOST", ""),
			Port:        getEnvInt("SFTP_PORT", 22),
			User:        getEnv("SFTP_USER", ""),
			Password:    getEnv("SFTP_PASSWORD", ""),
			RemotePath:  getEnv("SFTP_REMOTE_PATH", ""),
			RemotePath2: getEnv("SFTP_REMOTE_PATH2", ""),
			LocalPath:   getEnv("SFTP_LOCAL_PATH", "./raw-data"),
		},
		Server: ServerConfig{
			Port:           getEnv("WEBSOCKET_PORT", "8080"),
			StaticDir:      getEnv("STATIC_DIR", "./web/static"),
			DataDir:        getEnv("DATA_DIR", "./raw-data"),
			AppEnv:         getEnv("APP_ENV", "development"),
			ServiceName:    getEnv("SERVICE_NAME", "gold-socket"),
			PublicBaseURL:  getEnv("PUBLIC_BASE_URL", ""),
			AllowedOrigins: getEnvList("ALLOWED_ORIGINS"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			Enabled:  getEnv("REDIS_ENABLED", "false") == "true",
		},
		Schedule: ScheduleConfig{
			DownloadInterval: time.Duration(getEnvInt("DOWNLOAD_INTERVAL_SECONDS", 1)) * time.Second,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}
