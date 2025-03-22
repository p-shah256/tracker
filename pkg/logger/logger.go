package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BoldRed     = "\033[1;31m"
	BoldGreen   = "\033[1;32m"
	BoldYellow  = "\033[1;33m"
	BoldBlue    = "\033[1;34m"
	BoldMagenta = "\033[1;35m"
	BoldCyan    = "\033[1;36m"
	BoldWhite   = "\033[1;37m"
)

var levelColors = map[slog.Level]string{
	slog.LevelDebug: Cyan,
	slog.LevelInfo:  Green,
	slog.LevelWarn:  Yellow,
	slog.LevelError: Red,
}

type RequestKey string

const (
	RequestIDKey RequestKey = "requestID"
)

type ColoredHandler struct {
	h   slog.Handler
	out io.Writer
}

func NewColoredHandler(w io.Writer, opts *slog.HandlerOptions) *ColoredHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	origHandler := slog.NewTextHandler(w, opts)

	return &ColoredHandler{
		h:   origHandler,
		out: w,
	}
}

func (h *ColoredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *ColoredHandler) Handle(ctx context.Context, r slog.Record) error {
	timeStr := r.Time.Format("15:04:05.000")

	levelColor, ok := levelColors[r.Level]
	if !ok {
		levelColor = White
	}
	levelStr := fmt.Sprintf("%-6s", strings.ToUpper(r.Level.String()))

	var logLine strings.Builder
	logLine.WriteString(fmt.Sprintf("%s%s%s ", Magenta, timeStr, Reset))
	logLine.WriteString(fmt.Sprintf("%s%s%s ", levelColor, levelStr, Reset))

	var hasRequestID bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "request_id" && a.Value.Kind() == slog.KindString {
			// Pretty request ID
			logLine.WriteString(fmt.Sprintf("%s[%s]%s ", BoldBlue, a.Value.String(), Reset))
			hasRequestID = true
		}
		return true
	})

	logLine.WriteString(fmt.Sprintf("%s%s%s ", BoldWhite, r.Message, Reset))

	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "request_id" || !hasRequestID {
			val := a.Value.String()
			if a.Value.Kind() == slog.KindString {
				val = fmt.Sprintf("\"%s\"", val)
			}
			logLine.WriteString(fmt.Sprintf("%s%s%s=%s ", Yellow, a.Key, Reset, val))
		}
		return true
	})

	fmt.Fprintln(h.out, logLine.String())

	return nil
}

func (h *ColoredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColoredHandler{
		h:   h.h.WithAttrs(attrs),
		out: h.out,
	}
}

func (h *ColoredHandler) WithGroup(name string) slog.Handler {
	return &ColoredHandler{
		h:   h.h.WithGroup(name),
		out: h.out,
	}
}

func Setup() *ColoredHandler {
	handler := NewColoredHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))

	return handler
}

func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
