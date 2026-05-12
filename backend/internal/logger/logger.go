package logger

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// Level represents log severity.
type Level string

const (
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

var std = log.New(os.Stdout, "", 0)

var (
	sinkMu sync.Mutex
	sink   func(Entry)
)

// RegisterSink sets a callback invoked on every WARN and ERROR log entry.
// Call once at startup. The sink must not call logger functions (re-entrant deadlock).
func RegisterSink(fn func(Entry)) {
	sinkMu.Lock()
	defer sinkMu.Unlock()
	sink = fn
}

// Entry is the structured JSON log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     Level  `json:"level"`
	Message   string `json:"message"`
	// Optional fields
	ErrorCode      string `json:"error_code,omitempty"`
	Endpoint       string `json:"endpoint,omitempty"`
	RawResponse    string `json:"raw_response,omitempty"`
	RequestContext string `json:"request_context,omitempty"`
	Extra          any    `json:"extra,omitempty"`
}

func write(e Entry) {
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	b, _ := json.Marshal(e)
	std.Println(string(b))

	if e.Level == LevelWarn || e.Level == LevelError {
		sinkMu.Lock()
		fn := sink
		sinkMu.Unlock()
		if fn != nil {
			fn(e)
		}
	}
}

// Info logs an informational message.
func Info(msg string, extra ...any) {
	e := Entry{Level: LevelInfo, Message: msg}
	if len(extra) > 0 {
		e.Extra = extra[0]
	}
	write(e)
}

// Warn logs a warning message.
func Warn(msg string, extra ...any) {
	e := Entry{Level: LevelWarn, Message: msg}
	if len(extra) > 0 {
		e.Extra = extra[0]
	}
	write(e)
}

// Error logs an error message.
func Error(msg string, extra ...any) {
	e := Entry{Level: LevelError, Message: msg}
	if len(extra) > 0 {
		e.Extra = extra[0]
	}
	write(e)
}

// AutomationInfo logs an INFO-level message AND persists it to the sink (Firestore).
// Use for important automation lifecycle events (scan start/end, filter summary, orders placed).
// Unlike Info(), this always triggers the sink so it appears in the service log UI.
func AutomationInfo(msg string, extra ...any) {
	e := Entry{Level: LevelInfo, Message: msg}
	if len(extra) > 0 {
		e.Extra = extra[0]
	}
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	b, _ := json.Marshal(e)
	std.Println(string(b))

	sinkMu.Lock()
	fn := sink
	sinkMu.Unlock()
	if fn != nil {
		fn(e)
	}
}

// KISError logs a KIS API error with mandatory fields per CLAUDE.md spec:
// Error Code, Timestamp, and raw KIS API Response Message MUST be included.
// requestContext captures the query parameters or stock code context for diagnostics.
func KISError(endpoint, errorCode, rawResponse, requestContext string) {
	write(Entry{
		Level:          LevelError,
		Message:        "KIS API error",
		Endpoint:       endpoint,
		ErrorCode:      errorCode,
		RawResponse:    rawResponse,
		RequestContext: requestContext,
	})
}
