package custom_log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// LogLevel represents different log levels
type LogLevel int

const (
	DEBUG LogLevel = 7
	INFO  LogLevel = 6
	WARN  LogLevel = 5
	ERROR LogLevel = 4
	FATAL LogLevel = 3
)

// GraylogLogger represents a logger for a specific service
type GraylogLogger struct {
	FacilityName string
	ServiceName  string
	GraylogURL   string
	HTTPClient   *http.Client
}

// LogMessage represents a log message to send to Graylog
type LogMessage struct {
	Version      string         `json:"version"`
	Host         string         `json:"host"`
	ShortMessage string         `json:"short_message"`
	FullMessage  string         `json:"full_message,omitempty"`
	Timestamp    float64        `json:"timestamp"`
	Level        int            `json:"level"`
	Facility     string         `json:"facility"`
	Service      string         `json:"_service"`
	Extra        map[string]any `json:"-"`
}

// MarshalJSON adds extra fields with underscore prefix
func (lm *LogMessage) MarshalJSON() ([]byte, error) {
	type Alias LogMessage
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(lm),
	}

	result := make(map[string]any)
	data, _ := json.Marshal(aux)
	json.Unmarshal(data, &result)

	// Add extra fields with underscore prefix
	for k, v := range lm.Extra {
		result["_"+k] = v
	}

	return json.Marshal(result)
}

// NewGraylogLogger creates a new logger for a service
func NewGraylogLogger(
	facilityName string,
	serviceName, graylogURL string,
) *GraylogLogger {
	return &GraylogLogger{
		FacilityName: facilityName,
		ServiceName:  serviceName,
		GraylogURL:   graylogURL,
		HTTPClient:   &http.Client{},
	}
}

func (gl *GraylogLogger) Log(
	level slog.Level,
	message string,
	attrs map[string]any,
) {
	fullMessage, ok := attrs["full_message"].(string)
	if !ok {
		fullMessage = ""
	}

	group, ok := attrs["group"]
	if ok && group == "api" {
		words := strings.Split(message, " ")
		for i, word := range words {
			words[i] = removeAllANSICodes(word)
			if i == 0 {
				attrs["method"] = words[i]
			} else {
				dt := strings.Split(words[i], ":")
				if len(dt) == 2 {
					attrs[dt[0]] = dt[1]
				}
			}
		}

		message = strings.Join(words, " ")
	}

	switch level {
	case slog.LevelError:
		gl.Error(message, fullMessage, attrs)
	case slog.LevelWarn:
		gl.Warn(message, attrs)
	case slog.LevelInfo:
		gl.Info(message, attrs)
	case slog.LevelDebug:
		gl.Debug(message, attrs)
	}
}

func removeAllANSICodes(str string) string {
	// This regex matches all ANSI escape sequences, not just color codes
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(str, "")
}

func (gl *GraylogLogger) Close() {
	// No resources to close for HTTP client
}

// sendLog sends a log message to Graylog
func (gl *GraylogLogger) sendLog(
	level LogLevel,
	message string,
	fullMessage string,
	extra map[string]any,
) error {
	loggedAt, ok := extra["logged_at"].(time.Time)
	if !ok {
		loggedAt = time.Now()
	} else {
		delete(extra, "logged_at")
	}

	logMsg := &LogMessage{
		Version:      "1.1",
		Host:         gl.ServiceName,
		ShortMessage: message,
		FullMessage:  fullMessage,
		Timestamp:    float64(loggedAt.Unix()),
		Level:        int(level),
		Facility:     gl.FacilityName,
		Service:      gl.ServiceName,
		Extra:        extra,
	}

	jsonData, err := json.Marshal(logMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal log message: %v", err)
	}

	resp, err := gl.HTTPClient.Post(
		gl.GraylogURL,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send log to Graylog: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Graylog returned status: %d", resp.StatusCode)
	}

	return nil
}

// Debug logs a debug message
func (gl *GraylogLogger) Debug(message string, extra ...map[string]any) {
	extraFields := make(map[string]any)
	if len(extra) > 0 {
		extraFields = extra[0]
	}

	gl.sendLog(DEBUG, message, "", extraFields)
}

// Info logs an info message
func (gl *GraylogLogger) Info(message string, extra ...map[string]any) {
	extraFields := make(map[string]any)
	if len(extra) > 0 {
		extraFields = extra[0]
	}

	gl.sendLog(INFO, message, "", extraFields)
}

// Warn logs a warning message
func (gl *GraylogLogger) Warn(message string, extra ...map[string]any) {
	extraFields := make(map[string]any)
	if len(extra) > 0 {
		extraFields = extra[0]
	}

	gl.sendLog(WARN, message, "", extraFields)
}

// Error logs an error message
func (gl *GraylogLogger) Error(
	message string,
	fullMessage string,
	extra ...map[string]any,
) {
	extraFields := make(map[string]any)
	if len(extra) > 0 {
		extraFields = extra[0]
	}

	gl.sendLog(ERROR, message, fullMessage, extraFields)
}

// Fatal logs a fatal message
func (gl *GraylogLogger) Fatal(
	message string,
	fullMessage string,
	extra ...map[string]any,
) {
	extraFields := make(map[string]any)
	if len(extra) > 0 {
		extraFields = extra[0]
	}

	gl.sendLog(FATAL, message, fullMessage, extraFields)
}
