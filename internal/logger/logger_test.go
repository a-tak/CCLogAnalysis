package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		want     Level
	}{
		{
			name:     "デフォルトはINFO",
			logLevel: "",
			want:     INFO,
		},
		{
			name:     "DEBUG レベル",
			logLevel: "DEBUG",
			want:     DEBUG,
		},
		{
			name:     "INFO レベル",
			logLevel: "INFO",
			want:     INFO,
		},
		{
			name:     "WARN レベル",
			logLevel: "WARN",
			want:     WARN,
		},
		{
			name:     "ERROR レベル",
			logLevel: "ERROR",
			want:     ERROR,
		},
		{
			name:     "不正な値はINFO",
			logLevel: "INVALID",
			want:     INFO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を設定
			if tt.logLevel != "" {
				os.Setenv("LOG_LEVEL", tt.logLevel)
				defer os.Unsetenv("LOG_LEVEL")
			} else {
				os.Unsetenv("LOG_LEVEL")
			}

			logger := New()
			if logger.level != tt.want {
				t.Errorf("New() level = %v, want %v", logger.level, tt.want)
			}
		})
	}
}

func TestLogger_LogLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		logLevel      Level
		logFunc       func(*Logger)
		shouldContain string
		shouldLog     bool
	}{
		{
			name:     "INFO レベルで DEBUG は出力されない",
			logLevel: INFO,
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			shouldContain: "debug message",
			shouldLog:     false,
		},
		{
			name:     "INFO レベルで INFO は出力される",
			logLevel: INFO,
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			shouldContain: "info message",
			shouldLog:     true,
		},
		{
			name:     "ERROR レベルで WARN は出力されない",
			logLevel: ERROR,
			logFunc: func(l *Logger) {
				l.Warn("warning message")
			},
			shouldContain: "warning message",
			shouldLog:     false,
		},
		{
			name:     "ERROR レベルで ERROR は出力される",
			logLevel: ERROR,
			logFunc: func(l *Logger) {
				l.Error("error message")
			},
			shouldContain: "error message",
			shouldLog:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// バッファを作成してロガーをセットアップ
			buf := &bytes.Buffer{}
			logger := &Logger{
				level: tt.logLevel,
				out:   buf,
			}

			// ログ関数を実行
			tt.logFunc(logger)

			// 出力を確認
			output := buf.String()
			containsMessage := strings.Contains(output, tt.shouldContain)

			if tt.shouldLog && !containsMessage {
				t.Errorf("Expected log to contain %q, but got: %s", tt.shouldContain, output)
			}
			if !tt.shouldLog && containsMessage {
				t.Errorf("Expected no log output, but got: %s", output)
			}
		})
	}
}

func TestLogger_StructuredLogging(t *testing.T) {
	tests := []struct {
		name          string
		logFunc       func(*Logger)
		wantContains  []string
		wantLevel     string
	}{
		{
			name: "Info with context",
			logFunc: func(l *Logger) {
				l.InfoWithContext("test message", map[string]interface{}{
					"project": "test-project",
					"count":   42,
				})
			},
			wantContains: []string{"INFO", "test message", "project", "test-project", "count", "42"},
			wantLevel:    "INFO",
		},
		{
			name: "Error with context",
			logFunc: func(l *Logger) {
				l.ErrorWithContext("error occurred", map[string]interface{}{
					"session_id": "abc123",
					"error":      "database error",
				})
			},
			wantContains: []string{"ERROR", "error occurred", "session_id", "abc123", "error", "database error"},
			wantLevel:    "ERROR",
		},
		{
			name: "Debug with context",
			logFunc: func(l *Logger) {
				l.DebugWithContext("debug info", map[string]interface{}{
					"step": "processing",
				})
			},
			wantContains: []string{"DEBUG", "debug info", "step", "processing"},
			wantLevel:    "DEBUG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := &Logger{
				level: DEBUG, // すべてのログを出力
				out:   buf,
			}

			tt.logFunc(logger)

			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but got: %s", want, output)
				}
			}
		})
	}
}

func TestLogger_SimpleLogging(t *testing.T) {
	tests := []struct {
		name         string
		logFunc      func(*Logger)
		wantContains []string
	}{
		{
			name: "Debug log",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			wantContains: []string{"DEBUG", "debug message"},
		},
		{
			name: "Info log",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			wantContains: []string{"INFO", "info message"},
		},
		{
			name: "Warn log",
			logFunc: func(l *Logger) {
				l.Warn("warning message")
			},
			wantContains: []string{"WARN", "warning message"},
		},
		{
			name: "Error log",
			logFunc: func(l *Logger) {
				l.Error("error message")
			},
			wantContains: []string{"ERROR", "error message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := &Logger{
				level: DEBUG, // すべてのログを出力
				out:   buf,
			}

			tt.logFunc(logger)

			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but got: %s", want, output)
				}
			}
		})
	}
}
