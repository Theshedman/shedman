package http_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	shedhttp "github.com/theshedman/shedman/internal/http"
)

func TestRetryClient_UsesFirstMirror_WhenSucceeds(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))

	}))
	defer server.Close()

	client := shedhttp.NewRetryClient([]string{server.URL}, 0)

	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryClient_FailsOver_WhenFirstFails(t *testing.T) {
	var firstCalls, secondCalls int32

	// First server always fails
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&firstCalls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer first.Close()

	// Second server succeeds
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&secondCalls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success from second"))

	}))
	defer second.Close()

	client := shedhttp.NewRetryClient([]string{first.URL, second.URL}, 0)

	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("Expected success from failover, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if atomic.LoadInt32(&firstCalls) != 1 {
		t.Error("Expected first mirror to be tried")
	}
	if atomic.LoadInt32(&secondCalls) != 1 {
		t.Error("Expected second mirror to be tried after first failed")
	}
}

func TestRetryClient_ReturnsError_WhenAllFail(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer first.Close()

	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer second.Close()

	client := shedhttp.NewRetryClient([]string{first.URL, second.URL}, 0)

	_, err := client.Get("/test")
	if err == nil {
		t.Error("Expected error when all mirrors fail")
	}
}

func TestRetryClient_ReturnsError_WhenNoMirrors(t *testing.T) {
	client := shedhttp.NewRetryClient(nil, 0)

	_, err := client.Get("/test")
	if err == nil {
		t.Error("Expected error when no mirrors are configured")
	}
}
