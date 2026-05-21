package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type AlertHook func(level, message, detail string)

var alertHook AlertHook

func RegisterAlertHook(h AlertHook) {
	alertHook = h
}

// Setup configures the process-wide logger for GCP Cloud Logging compatible JSON.
func Setup() {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Key = "severity"
				if l, ok := a.Value.Any().(slog.Level); ok {
					a.Value = slog.StringValue(toGCPSeverity(l))
				}
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	})
	slog.SetDefault(slog.New(hookHandler{Handler: h}))
}

type hookHandler struct {
	slog.Handler
}

func (h hookHandler) Handle(ctx context.Context, r slog.Record) error {
	if alertHook != nil && r.Level >= slog.LevelWarn && !strings.HasPrefix(r.Message, "discord webhook") {
		detail := ""
		r.Attrs(func(a slog.Attr) bool {
			if detail != "" {
				detail += " "
			}
			detail += a.Key + "=" + a.Value.String()
			return true
		})
		go alertHook(r.Level.String(), r.Message, detail)
	}
	return h.Handler.Handle(ctx, r)
}

func toGCPSeverity(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// Info logs through slog. Kept for package-level callsite compatibility.
func Info(msg string, extra ...any) {
	slog.Info(msg, attrs(extra...)...)
}

// Warn logs through slog. Kept for package-level callsite compatibility.
func Warn(msg string, extra ...any) {
	slog.Warn(msg, attrs(extra...)...)
}

// Error logs through slog. Kept for package-level callsite compatibility.
func Error(msg string, extra ...any) {
	slog.Error(msg, attrs(extra...)...)
}

// AutomationInfo now writes only to stdout via slog; Firestore log persistence was removed.
func AutomationInfo(msg string, extra ...any) {
	slog.Info(msg, attrs(extra...)...)
}

func attrs(extra ...any) []any {
	if len(extra) == 0 || extra[0] == nil {
		return nil
	}
	if m, ok := extra[0].(map[string]any); ok {
		out := make([]any, 0, len(m)*2)
		for k, v := range m {
			out = append(out, k, v)
		}
		return out
	}
	return []any{"extra", extra[0]}
}

// KISError logs KIS API errors with stable fields for Cloud Logging filters.
func KISError(endpoint, errorCode, rawResponse, requestContext string) {
	slog.Error("KIS API error",
		"endpoint", endpoint,
		"error_code", errorCode,
		"raw_response", rawResponse,
		"request_context", requestContext,
	)
}
