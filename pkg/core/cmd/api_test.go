package cmd

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

type mockAPIServer struct {
	addr string
}

func (m *mockAPIServer) Serve(addr string) error {
	m.addr = addr
	return nil
}

func TestRunAPI(t *testing.T) {
	eng := core.NewEngine()

	mockServer := &mockAPIServer{}
	origFactory := apiServerFactory
	apiServerFactory = func(_ *core.Engine) apiServer {
		return mockServer
	}
	t.Cleanup(func() {
		apiServerFactory = origFactory
	})

	addr := "127.0.0.1:9999"
	if err := RunAPI(eng, addr); err != nil {
		t.Fatalf("RunAPI failed: %v", err)
	}

	if mockServer.addr != addr {
		t.Errorf("expected server to use addr %s, got %s", addr, mockServer.addr)
	}
}
