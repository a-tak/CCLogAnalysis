package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Level represents log level
type Level int

const (
	// DEBUG level for detailed debugging information
	DEBUG Level = iota
	// INFO level for general information
	INFO
	// WARN level for warnings
	WARN
	// ERROR level for errors
	ERROR
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging
type Logger struct {
	level Level
	out   io.Writer
}

// New creates a new Logger instance
// Log level is controlled by LOG_LEVEL environment variable (default: INFO)
func New() *Logger {
	level := INFO
	levelStr := os.Getenv("LOG_LEVEL")

	if levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			level = DEBUG
		case "INFO":
			level = INFO
		case "WARN":
			level = WARN
		case "ERROR":
			level = ERROR
		default:
			level = INFO
		}
	}

	return &Logger{
		level: level,
		out:   os.Stdout,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(DEBUG, message, nil)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(INFO, message, nil)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(WARN, message, nil)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(ERROR, message, nil)
}

// DebugWithContext logs a debug message with context
func (l *Logger) DebugWithContext(message string, context map[string]interface{}) {
	l.log(DEBUG, message, context)
}

// InfoWithContext logs an info message with context
func (l *Logger) InfoWithContext(message string, context map[string]interface{}) {
	l.log(INFO, message, context)
}

// WarnWithContext logs a warning message with context
func (l *Logger) WarnWithContext(message string, context map[string]interface{}) {
	l.log(WARN, message, context)
}

// ErrorWithContext logs an error message with context
func (l *Logger) ErrorWithContext(message string, context map[string]interface{}) {
	l.log(ERROR, message, context)
}

// SetOutput sets the output writer for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
}

// SetLevel sets the log level for the logger
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// log is an internal method that handles the actual logging
func (l *Logger) log(level Level, message string, context map[string]interface{}) {
	// ログレベルのフィルタリング
	if level < l.level {
		return
	}

	// タイムスタンプを取得
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// ログメッセージを構築
	logMsg := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), message)

	// コンテキスト情報を追加
	if context != nil && len(context) > 0 {
		contextStr := ""
		for key, value := range context {
			contextStr += fmt.Sprintf(" %s=%v", key, value)
		}
		logMsg += contextStr
	}

	// 出力
	fmt.Fprintln(l.out, logMsg)
}
