package shedman

import (
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
)

// Client provides a programmatic interface to shedman.
type Client struct {
	engine *core.Engine
}

// InstallOptions controls install behavior.
type InstallOptions struct {
	Confirm bool
}

// Option configures client behavior.
type Option func(*InstallOptions)

// WithConfirm sets whether operations should prompt for confirmation.
func WithConfirm(confirm bool) Option {
	return func(opts *InstallOptions) {
		opts.Confirm = confirm
	}
}

// New creates a client with a default engine.
func New() *Client {
	cfg, err := config.LoadDefault()
	if err != nil {
		cfg = config.Default()
	}

	engine := core.NewEngine()
	engine.SetConfig(cfg)

	if backend, err := core.DetectBackendWithConfig(&cfg.Backend); err == nil {
		engine.SetOfficialBackend(backend)
	}

	return &Client{engine: engine}
}

// NewWithEngine creates a client with a provided engine.
func NewWithEngine(engine *core.Engine) *Client {
	return &Client{engine: engine}
}

// Search searches for packages.
func (c *Client) Search(query string) ([]core.PackageInfo, error) {
	if c == nil || c.engine == nil {
		return nil, core.ErrBackendNotFound
	}
	return c.engine.Search(query)
}

// Install installs a package.
func (c *Client) Install(pkg string, opts ...Option) error {
	if c == nil || c.engine == nil {
		return core.ErrBackendNotFound
	}

	options := InstallOptions{Confirm: true}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return c.engine.Install([]string{pkg}, core.InstallOptions{
		NoConfirm: !options.Confirm,
	})
}
