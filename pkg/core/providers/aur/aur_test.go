package aur

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/theshedman/shedman/pkg/core"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAURBackend_Name(t *testing.T) {
	c := core.NewPackageFileCache(24 * time.Hour)
	b := New(c)
	if b.Name() != "aur" {
		t.Errorf("Expected name 'aur', got '%s'", b.Name())
	}
}

func TestAURBackend_Sync_Success(t *testing.T) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/nonexistent")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"version":5,"type":"multiinfo","resultcount":1,"results":[]}`)),
			Header:     make(http.Header),
		}, nil
	})}

	c := core.NewPackageFileCache(24 * time.Hour)
	b := NewWithURL("http://aur.test/rpc/", c)
	b.client = client
	err := b.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestAURBackend_Sync_Failure(t *testing.T) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/nonexistent")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("fail")),
			Header:     make(http.Header),
		}, nil
	})}

	c := core.NewPackageFileCache(24 * time.Hour)
	b := NewWithURL("http://aur.test/rpc/", c)
	b.client = client
	err := b.Sync()
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}
