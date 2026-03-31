package custom_log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	timeFormat = "[2006-01-02 15:04:05.000]"
	reset      = "\033[0m"

	SlogFieldsKey = "slog_fields"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

var (
	PostColor string = fmt.Sprintf(
		"\033[%sm%s%s",
		strconv.Itoa(lightGreen),
		"PST",
		"\033[0m",
	)
	GetColor string = fmt.Sprintf(
		"\033[%sm%s%s",
		strconv.Itoa(lightBlue),
		"GET",
		"\033[0m",
	)
	PutColor string = fmt.Sprintf(
		"\033[%sm%s%s",
		strconv.Itoa(lightYellow),
		"PUT",
		"\033[0m",
	)
	DeleteColor string = fmt.Sprintf(
		"\033[%sm%s%s",
		strconv.Itoa(lightRed),
		"DEL",
		"\033[0m",
	)
	PatchColor string = fmt.Sprintf(
		"\033[%sm%s%s",
		strconv.Itoa(lightMagenta),
		"PTC",
		"\033[0m",
	)

	SuccessColor string = fmt.Sprintf(
		"\033[%sm%%v%s",
		strconv.Itoa(green),
		"\033[0m",
	)

	FailColor string = fmt.Sprintf(
		"\033[%sm%%v%s",
		strconv.Itoa(yellow),
		"\033[0m",
	)

	ErrorColor string = fmt.Sprintf(
		"\033[%sm%%v%s",
		strconv.Itoa(red),
		"\033[0m",
	)

	labelFields = map[string]bool{
		"trace_id": true,
		"user_id":  true,
		"api_key":  true,
		"ip":       true,
		"group":    true,
	}
)

type Handler struct {
	h slog.Handler
	b *bytes.Buffer
	m *sync.Mutex

	client LogClient
}

func NewHandler(
	opts *slog.HandlerOptions,
	customOpt ...HandlerOptions,
) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	b := &bytes.Buffer{}
	handler := &Handler{
		b: b,
		h: slog.NewTextHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		m: &sync.Mutex{},
	}

	for _, opt := range customOpt {
		opt(handler)
	}

	return handler
}

type HandlerOptions func(*Handler)

func WithTextHandlerOptions(opts *slog.HandlerOptions) func(*Handler) {
	return func(h *Handler) {
		h.h = slog.NewTextHandler(h.b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		})
	}
}

func WithLogClient(client LogClient) func(*Handler) {
	return func(h *Handler) {
		h.client = client
	}
}

func WithJSONHandlerOptions(opts *slog.HandlerOptions) func(*Handler) {
	return func(h *Handler) {
		h.h = slog.NewJSONHandler(h.b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		})
	}
}

func NewLogHandler(h slog.Handler) *Handler {
	return &Handler{
		h: h,
		b: &bytes.Buffer{},
		m: &sync.Mutex{},
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{h: h.h.WithAttrs(attrs), b: h.b, m: h.m, client: h.client}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{h: h.h.WithGroup(name), b: h.b, m: h.m, client: h.client}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var level string
	level = r.Level.String() + ":"
	switch r.Level {
	case slog.LevelDebug:
		level = "DEB"
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = "INF"
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = "WRN"
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = "ERR"
		level = colorize(lightRed, level)
	}

	if attrs, ok := ctx.Value(SlogFieldsKey).([]slog.Attr); ok {
		r.AddAttrs(attrs...)
	}

	data, err := h.computeAttrs(ctx, r)
	if err != nil {
		fmt.Println("err", err.Error())
		return err
	}

	attrs := make(map[string]any)
	var attrsString string
	switch h.h.(type) {
	case *slog.TextHandler:
		items := parseLogString(string(data))
		for k, v := range items {
			attrs[k] = strings.TrimSuffix(v, "\n")
		}

		attrsString = string(data)
	case *slog.JSONHandler:
		if err := json.Unmarshal(data, &attrs); err != nil {
			return fmt.Errorf("error when unmarshaling data: %w", err)
		}

		var attrBytes []byte
		attrBytes, err = json.MarshalIndent(attrs, "", "  ")
		if err != nil {
			return fmt.Errorf("error when marshaling attrs: %w", err)
		}

		attrsString = string(attrBytes)
	}

	fmt.Println(
		colorize(lightGray, r.Time.Format(timeFormat)),
		level,
		colorize(white, r.Message),
		colorize(darkGray, strings.TrimSuffix(attrsString, "\n")),
	)

	attrs["logged_at"] = r.Time
	if h.client != nil {
		go func() {
			h.client.Log(r.Level, r.Message, attrs)
		}()
	}

	return nil
}

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

func (h *Handler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) ([]byte, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()
	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf(
			"error when calling inner handler's Handle: %w",
			err,
		)
	}

	return h.b.Bytes(), nil
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

func parseLogString(logStr string) map[string]string {
	re := regexp.MustCompile(`(\w+)=(".*?"|\S+)`)
	parsedData := make(map[string]string)
	matches := re.FindAllStringSubmatch(logStr, -1)
	for _, match := range matches {
		key := match[1]
		value := match[2]
		parsedData[key] = strings.Trim(value, "\"")
	}

	return parsedData
}
