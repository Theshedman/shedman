package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultTimeout = 30 * time.Second
)

// RetryClient is an HTTP client that tries multiple mirrors in order
type RetryClient struct {
	mirrors []string
	client  *http.Client
}

// NewRetryClient creates a new RetryClient with the given mirror URLs and timeout
func NewRetryClient(mirrors []string, timeout time.Duration) *RetryClient {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return NewRetryClientWithClient(mirrors, &http.Client{Timeout: timeout})
}

// NewRetryClientWithClient creates a RetryClient using a custom http.Client.
func NewRetryClientWithClient(mirrors []string, client *http.Client) *RetryClient {
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	return &RetryClient{
		mirrors: mirrors,
		client:  client,
	}
}

// Get performs a GET request, trying each mirror in order until one succeeds
func (c *RetryClient) Get(path string) (*http.Response, error) {
	if len(c.mirrors) == 0 {
		return nil, fmt.Errorf("no mirrors configured")
	}

	var lastErr error
	var errors []string

	for _, mirror := range c.mirrors {
		url := mirror + path
		resp, err := c.client.Get(url)
		if err != nil {
			lastErr = err
			errors = append(errors, fmt.Sprintf("%s: %v", mirror, err))
			continue
		}

		// Check for successful status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		// Non-success status, close body and try next
		_ = resp.Body.Close()

		errors = append(errors, fmt.Sprintf("%s: status %d", mirror, resp.StatusCode))
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("all mirrors failed: %s", strings.Join(errors, "; "))
	}

	return nil, lastErr
}
