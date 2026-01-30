package mirror

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReflectorBackend_List(t *testing.T) {
	tmp := t.TempDir() + "/mirrorlist"
	content := "## Testland\n" +
		"Server = https://mirror.example/$repo/os/$arch\n" +
		"#Server = https://disabled.example/$repo/os/$arch\n"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write mirrorlist: %v", err)
	}

	backend := NewReflectorBackendWithExecutor(nil, tmp, time.Second)
	mirrors, err := backend.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(mirrors) != 1 {
		t.Fatalf("Expected 1 mirror, got %d", len(mirrors))
	}
	if mirrors[0].Country != "Testland" {
		t.Errorf("Unexpected country: %s", mirrors[0].Country)
	}
	if mirrors[0].URL != "https://mirror.example/$repo/os/$arch" {
		t.Errorf("Unexpected URL: %s", mirrors[0].URL)
	}
}

func TestReflectorBackend_Test(t *testing.T) {
	tmp := t.TempDir() + "/mirrorlist"
	content := "Server = http://good.test/$repo/os/$arch\n" +
		"Server = http://bad.test/$repo/os/$arch\n"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write mirrorlist: %v", err)
	}

	backend := NewReflectorBackendWithExecutor(nil, tmp, 50*time.Millisecond)
	backend.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Host, "good.test") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			}
			return nil, errors.New("dial error")
		}),
	}
	mirrors, err := backend.Test()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	if len(mirrors) != 1 {
		t.Fatalf("Expected 1 reachable mirror, got %d", len(mirrors))
	}
	if mirrors[0].Speed <= 0 {
		t.Errorf("Expected measured speed, got %v", mirrors[0].Speed)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
