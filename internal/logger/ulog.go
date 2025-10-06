package logger

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/fhuszti/medias-ms-go/internal/api_context"
)

var std *slog.Logger

// --- handler that appends uid as an attribute (at the end in TextHandler) ---

type userAttrHandler struct{ h slog.Handler }

func (u userAttrHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return u.h.Enabled(ctx, lvl)
}

func (u userAttrHandler) Handle(ctx context.Context, r slog.Record) error {
	if uid, ok := api_context.AuthUserIDFromContext(ctx); ok {
		r.AddAttrs(slog.String("uid", uid.String()))
	} else {
		r.AddAttrs(slog.String("uid", "system"))
	}
	return u.h.Handle(ctx, r)
}

func (u userAttrHandler) WithAttrs(a []slog.Attr) slog.Handler {
	return userAttrHandler{h: u.h.WithAttrs(a)}
}
func (u userAttrHandler) WithGroup(n string) slog.Handler {
	return userAttrHandler{h: u.h.WithGroup(n)}
}

// --- public API ---

// Init
// ENV:
//
//	LOG_FORMAT    json|text (default: json)
//	LOG_LEVEL     debug|info|warn|error (default: info)
//	LOG_SOURCE    true|false (default: false)
func Init() {
	level := parseLevel(getEnv("LOG_LEVEL", "info"))
	addSource := parseBool(getEnv("LOG_SOURCE", "false"))
	format := strings.ToLower(getEnv("LOG_FORMAT", "json"))

	opts := &slog.HandlerOptions{Level: level, AddSource: addSource}

	var base slog.Handler
	if format == "text" {
		base = slog.NewTextHandler(os.Stdout, opts)
	} else {
		base = slog.NewJSONHandler(os.Stdout, opts)
	}

	// Chain: add svc first (so it prints before uid in TextHandler), then wrap to add uid.
	logger := slog.New(userAttrHandler{h: base}).With("svc", "medias-ms")

	std = logger
	slog.SetDefault(std)

	// Keep legacy log.Printf visible (no ctx → no uid).
	log.SetFlags(0)
	log.SetOutput(slog.NewLogLogger(base, slog.LevelInfo).Writer())
}

// --- small helpers ---

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseLevel(s string) slog.Leveler {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func activeLogger() *slog.Logger {
	if std != nil {
		return std
	}
	return slog.Default()
}

// --- convenience wrappers ---

func Info(ctx context.Context, msg string, attrs ...any) {
	activeLogger().InfoContext(ctx, msg, attrs...)
}
func Warn(ctx context.Context, msg string, attrs ...any) {
	activeLogger().WarnContext(ctx, msg, attrs...)
}
func Error(ctx context.Context, msg string, attrs ...any) {
	activeLogger().ErrorContext(ctx, msg, attrs...)
}
func Debug(ctx context.Context, msg string, attrs ...any) {
	activeLogger().DebugContext(ctx, msg, attrs...)
}

func Infof(ctx context.Context, format string, a ...any) {
	activeLogger().InfoContext(ctx, fmt.Sprintf(format, a...))
}
func Errorf(ctx context.Context, format string, a ...any) {
	activeLogger().ErrorContext(ctx, fmt.Sprintf(format, a...))
}
func Warnf(ctx context.Context, format string, a ...any) {
	activeLogger().WarnContext(ctx, fmt.Sprintf(format, a...))
}
func Debugf(ctx context.Context, format string, a ...any) {
	activeLogger().DebugContext(ctx, fmt.Sprintf(format, a...))
}
