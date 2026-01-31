package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity level of log messages
type LogLevel int

const (
	// ErrorLevel - only errors
	ErrorLevel LogLevel = iota
	// WarnLevel - warnings and errors
	WarnLevel
	// InfoLevel - info, warnings, and errors
	InfoLevel
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m" // ERROR - Red
	colorYellow = "\033[33m" // WARN - Yellow
	colorCyan   = "\033[36m" // INFO - Cyan
	colorGray   = "\033[90m" // Dimmed - Gray
)

var (
	currentLevel = InfoLevel
	logger       *log.Logger
	mu           sync.Mutex
	noColor      = false
)

func init() {
	noColor = !isTerminal(os.Stdout)
	logger = log.New(os.Stdout, "", 0) // We'll handle formatting ourselves
}

// isTerminal checks if the writer is a terminal (works on Unix/Linux/macOS)
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		stat, _ := f.Stat()
		return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	}
	return false
}

// SetNoColor explicitly disables colored output
func SetNoColor(disable bool) {
	noColor = disable
}

// SetLevel sets the global log level
func SetLevel(level LogLevel) {
	currentLevel = level
}

// SetLevelFromString sets the log level from string (error, warn, info)
func SetLevelFromString(levelStr string) error {
	switch levelStr {
	case "error":
		SetLevel(ErrorLevel)
	case "warn":
		SetLevel(WarnLevel)
	case "info":
		SetLevel(InfoLevel)
	default:
		return fmt.Errorf("invalid log level: %s (use: error, warn, info)", levelStr)
	}
	return nil
}

// GetLevel returns the current log level
func GetLevel() LogLevel {
	return currentLevel
}

// getTimestamp returns the current timestamp in HH:MM:SS format
func getTimestamp() string {
	return time.Now().Format("15:04:05")
}

// formatMsg formats a log message with timestamp and colored level tag
func formatMsg(levelTag, color, format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)

	if noColor {
		return fmt.Sprintf("%s [%s] %s", getTimestamp(), levelTag, msg)
	}

	// Colored output: dimmed timestamp + colored level tag + message
	return fmt.Sprintf("%s%s%s %s%s%s %s",
		colorGray, getTimestamp(), colorReset,
		color, levelTag, colorReset,
		msg,
	)
}

// Info logs info messages in cyan (only when level >= info)
func Info(format string, args ...interface{}) {
	if currentLevel >= InfoLevel {
		mu.Lock()
		logger.Println(formatMsg("INFO", colorCyan, format, args...))
		mu.Unlock()
	}
}

// Warn logs warning messages in yellow (only when level >= warn)
func Warn(format string, args ...interface{}) {
	if currentLevel >= WarnLevel {
		mu.Lock()
		logger.Println(formatMsg("WARN", colorYellow, format, args...))
		mu.Unlock()
	}
}

// Error logs error messages in red (always shown)
func Error(format string, args ...interface{}) {
	mu.Lock()
	logger.Println(formatMsg("ERROR", colorRed, format, args...))
	mu.Unlock()
}

// Println logs a message with newline (respecting current level as Info, plain text)
func Println(args ...interface{}) {
	if currentLevel >= InfoLevel {
		mu.Lock()
		logger.Println(args...)
		mu.Unlock()
	}
}

// Printf logs a formatted message (respecting current level as Info, plain text)
func Printf(format string, args ...interface{}) {
	if currentLevel >= InfoLevel {
		mu.Lock()
		logger.Printf(format, args...)
		mu.Unlock()
	}
}

// Fatal logs an error message in red and exits
func Fatal(format string, args ...interface{}) {
	mu.Lock()
	logger.Fatalf(formatMsg("FATAL", colorRed, format, args...))
	mu.Unlock()
}
