package logger

import (
"encoding/json"
"fmt"
"io"
"os"
"time"
)

// Level represents log severity
type Level string

const (
LevelDebug Level = "DEBUG"
LevelInfo  Level = "INFO"
LevelWarn  Level = "WARN"
LevelError Level = "ERROR"
LevelFatal Level = "FATAL"
)

// Logger provides structured logging
type Logger struct {
writer     io.Writer
level      Level
jsonOutput bool
}

// LogEntry represents a structured log entry
type LogEntry struct {
Timestamp string                 `json:"timestamp"`
Level     string                 `json:"level"`
Message   string                 `json:"message"`
Fields    map[string]interface{} `json:"fields,omitempty"`
Error     string                 `json:"error,omitempty"`
}

var defaultLogger *Logger

// InitLogger initializes the global logger
func InitLogger(w io.Writer, jsonOutput bool, level Level) {
defaultLogger = &Logger{
writer:     w,
level:      level,
jsonOutput: jsonOutput,
}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
if defaultLogger == nil {
defaultLogger = &Logger{
writer:     os.Stderr,
level:      LevelInfo,
jsonOutput: false,
}
}
return defaultLogger
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, fields map[string]interface{}, err error) {
if l.shouldLog(level) {
entry := LogEntry{
Timestamp: time.Now().Format(time.RFC3339),
Level:     string(level),
Message:   msg,
Fields:    fields,
}

if err != nil {
entry.Error = err.Error()
}

if l.jsonOutput {
data, _ := json.Marshal(entry)
fmt.Fprintln(l.writer, string(data))
} else {
l.formatText(entry)
}
}
}

// formatText formats log entry as text
func (l *Logger) formatText(entry LogEntry) {
// Color codes for terminal
levelColors := map[string]string{
"DEBUG": "\033[36m", // Cyan
"INFO":  "\033[32m", // Green
"WARN":  "\033[33m", // Yellow
"ERROR": "\033[31m", // Red
"FATAL": "\033[35m", // Magenta"
}
reset := "\033[0m"

color := levelColors[entry.Level]
fmt.Fprintf(l.writer, "%s[%s]%s %s - %s",
color, entry.Level, reset, entry.Timestamp, entry.Message)

if len(entry.Fields) > 0 {
fmt.Fprintf(l.writer, " %v", entry.Fields)
}

if entry.Error != "" {
fmt.Fprintf(l.writer, " error=%s", entry.Error)
}

fmt.Fprintln(l.writer)
}

// shouldLog determines if message should be logged based on level
func (l *Logger) shouldLog(level Level) bool {
levels := map[Level]int{
LevelDebug: 0,
LevelInfo:  1,
LevelWarn:  2,
LevelError: 3,
LevelFatal: 4,
}
return levels[level] >= levels[l.level]
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
f := mergeFields(fields...)
l.log(LevelDebug, msg, f, nil)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
f := mergeFields(fields...)
l.log(LevelInfo, msg, f, nil)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
f := mergeFields(fields...)
l.log(LevelWarn, msg, f, nil)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, fields ...map[string]interface{}) {
f := mergeFields(fields...)
l.log(LevelError, msg, f, err)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...map[string]interface{}) {
f := mergeFields(fields...)
l.log(LevelFatal, msg, f, err)
os.Exit(1)
}

// mergeFields merges multiple field maps
func mergeFields(fields ...map[string]interface{}) map[string]interface{} {
result := make(map[string]interface{})
for _, f := range fields {
for k, v := range f {
result[k] = v
}
}
return result
}

// Global convenience functions
func Debug(msg string, fields ...map[string]interface{}) {
GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...map[string]interface{}) {
GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...map[string]interface{}) {
GetLogger().Warn(msg, fields...)
}

func Error(msg string, err error, fields ...map[string]interface{}) {
GetLogger().Error(msg, err, fields...)
}

func Fatal(msg string, err error, fields ...map[string]interface{}) {
GetLogger().Fatal(msg, err, fields...)
}
