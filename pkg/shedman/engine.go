package shedman

type Engine struct {
	backends []PackageBackend
}

func NewEngine() *Engine {
	return &Engine{ backends: make([]PackageBackend, 0) }
}

func (e *Engine) AddBackend(backend PackageBackend) {
	e.backends = append(e.backends, backend)
}

func (e *Engine) Sync() error {
	for _, backend := range e.backends {
		if err := backend.Sync(); err != nil {
			return err
		}
	}

	return nil
}