package custom_log

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ic2hrmk/promtail"
)

type LogClient interface {
	Close()
	Log(level slog.Level, message string, attrs map[string]any)
}

type LokiClient struct {
	indentifier map[string]string
	lokiUrl     string

	client promtail.Client
}

func NewLokiClient(lokiUrl string, indentifier map[string]string) LogClient {
	promtailClient, err := promtail.NewJSONv1Client(lokiUrl, indentifier)
	if err != nil {
		return nil
	}

	if _, err := promtailClient.Ping(); err != nil {
		return nil
	}

	return &LokiClient{
		client: promtailClient,
	}
}

func (c *LokiClient) Log(
	level slog.Level,
	message string,
	attrs map[string]any,
) {
	var pLevel promtail.Level
	switch level {
	case slog.LevelError:
		pLevel = promtail.Error
	// case slog.LevelFatal:
	// 	pLevel = promtail.Fatal
	// case slog.LevelPanic:
	// 	pLevel = promtail.Panic
	case slog.LevelWarn:
		pLevel = promtail.Warn
	case slog.LevelInfo:
		pLevel = promtail.Info
	case slog.LevelDebug:
		pLevel = promtail.Debug
	}

	labels := make(map[string]string)
	additionalData := make(map[string]any)
	for k, v := range attrs {
		if _, ok := labelFields[k]; ok {
			labels[k] = fmt.Sprintf("%v", v)
		} else {
			additionalData[k] = v
		}
	}

	if len(additionalData) > 0 {
		additionalDataBytes, err := json.Marshal(additionalData)
		if err != nil {
			return
		}

		c.client.LogfWithLabels(pLevel, labels, "%s, %v", message, string(additionalDataBytes))
	} else {
		c.client.LogfWithLabels(pLevel, labels, message)
	}
}

func (c *LokiClient) Close() {
	if c.client == nil {
		return
	}

	c.client.Close()
}
