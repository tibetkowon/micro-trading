package slack

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientRoutesMessagesToSpecificWebhooks(t *testing.T) {
	var fallbackHits int32
	var tradeHits int32

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fallbackHits, 1)
	}))
	defer fallback.Close()

	trade := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tradeHits, 1)
	}))
	defer trade.Close()

	client := NewWithWebhooks(Webhooks{
		Default: fallback.URL,
		Trade:   trade.URL,
	})

	client.TradeBuy("005930", "삼성전자", 70000, 1, "test")
	client.AlertWarn("warn", "detail")

	if got := atomic.LoadInt32(&tradeHits); got != 1 {
		t.Fatalf("trade webhook hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&fallbackHits); got != 1 {
		t.Fatalf("fallback webhook hits = %d, want 1", got)
	}
}

func TestClientDoesNotSendWithoutRoute(t *testing.T) {
	client := NewWithWebhooks(Webhooks{})

	client.AlertWarn("warn", "detail")
	client.TradeBuy("005930", "삼성전자", 70000, 1, "test")
}
