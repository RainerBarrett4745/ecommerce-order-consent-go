package consent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGrantRequestAndRetry(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/consent/grant/user-42" {
			t.Fatalf("unexpected boundary: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer credential")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["category"] != "marketing" || body["idempotency_key"] != "evt-9001" {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"data":{"category":"marketing"},"error":null,"metadata":{}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", server.Client())
	client.sleep = func(context.Context, time.Duration) error { return nil }
	if err := client.Grant(context.Background(), "user-42", "marketing", "checkout", "evt-9001"); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests=%d, want 2", requests)
	}
}

func TestEnvelopeErrorKeepsClientStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"BAD_CATEGORY","message":"category rejected"},"metadata":{}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", server.Client())
	err := client.Revoke(context.Background(), "user-42", "marketing", "evt-9002")
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.StatusCode != http.StatusBadRequest || apiErr.Code != "BAD_CATEGORY" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
