package http_test

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	shedhttp "github.com/theshedman/shedman/internal/http"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRetryClient_UsesFirstMirror_WhenSucceeds(t *testing.T) {
	var callCount int32

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&callCount, 1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("success")),
			Header:     make(http.Header),
		}, nil
	})}

	rc := shedhttp.NewRetryClientWithClient([]string{"http://mirror1"}, client)

	resp, err := rc.Get("/test")
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

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "mirror1":
			atomic.AddInt32(&firstCalls, 1)
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("fail")),
				Header:     make(http.Header),
			}, nil
		case "mirror2":
			atomic.AddInt32(&secondCalls, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("success from second")),
				Header:     make(http.Header),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("unknown")),
				Header:     make(http.Header),
			}, nil
		}
	})}

	rc := shedhttp.NewRetryClientWithClient([]string{"http://mirror1", "http://mirror2"}, client)

	resp, err := rc.Get("/test")
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
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("fail")),
			Header:     make(http.Header),
		}, nil
	})}

	rc := shedhttp.NewRetryClientWithClient([]string{"http://mirror1", "http://mirror2"}, client)

	_, err := rc.Get("/test")
	if err == nil {
		t.Error("Expected error when all mirrors fail")
	}
}

func TestRetryClient_ReturnsError_WhenNoMirrors(t *testing.T) {
	rc := shedhttp.NewRetryClientWithClient(nil, &http.Client{})

	_, err := rc.Get("/test")
	if err == nil {
		t.Error("Expected error when no mirrors are configured")
	}
}
