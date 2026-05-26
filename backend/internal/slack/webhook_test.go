package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	slackapi "github.com/slack-go/slack"
)

func TestClientRoutesMessagesToSpecificChannel(t *testing.T) {
	recorder := newSlackAPIRecorder(t)
	defer recorder.close()

	client := newWithChannels("xoxb-test-token", Channels{
		Default: "CDEFAULT",
		Trade:   "CTRADE",
	}, slackapi.OptionAPIURL(recorder.url()))

	client.TradeBuy("005930", "삼성전자", 70000, 1, "test")
	client.AlertWarn("warn", "detail")

	recorder.assertChannels("CTRADE", "CDEFAULT")
}

func TestClientDoesNotSendWithoutRoute(t *testing.T) {
	recorder := newSlackAPIRecorder(t)
	defer recorder.close()

	client := newWithChannels("xoxb-test-token", Channels{}, slackapi.OptionAPIURL(recorder.url()))

	client.AlertWarn("warn", "detail")
	client.TradeBuy("005930", "삼성전자", 70000, 1, "test")

	recorder.assertChannels()
}

func TestClientFallsBackToDefault(t *testing.T) {
	recorder := newSlackAPIRecorder(t)
	defer recorder.close()

	client := newWithChannels("xoxb-test-token", Channels{
		Default: "CDEFAULT",
	}, slackapi.OptionAPIURL(recorder.url()))

	client.TradeBuy("005930", "삼성전자", 70000, 1, "test")

	recorder.assertChannels("CDEFAULT")
}

type slackAPIRecorder struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	channels []string
}

func newSlackAPIRecorder(t *testing.T) *slackAPIRecorder {
	t.Helper()
	recorder := &slackAPIRecorder{t: t}
	recorder.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("path = %s, want /chat.postMessage", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		recorder.mu.Lock()
		recorder.channels = append(recorder.channels, r.Form.Get("channel"))
		recorder.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"channel": r.Form.Get("channel"),
			"ts":      "1710000000.000000",
		})
	}))
	return recorder
}

func (r *slackAPIRecorder) url() string {
	return r.server.URL + "/"
}

func (r *slackAPIRecorder) close() {
	r.server.Close()
}

func (r *slackAPIRecorder) assertChannels(want ...string) {
	r.t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.channels) != len(want) {
		r.t.Fatalf("channels = %v, want %v", r.channels, want)
	}
	for i := range want {
		if r.channels[i] != want[i] {
			r.t.Fatalf("channels = %v, want %v", r.channels, want)
		}
	}
}
