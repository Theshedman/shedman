package svc

// ServiceBackend defines the interface for service management (e.g. systemd)
type ServiceBackend interface {
	Name() string
	List() ([]Service, error)
	Enable(name string) error
	Disable(name string) error
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	IsActive(name string) (bool, error)
	IsEnabled(name string) (bool, error)
}
