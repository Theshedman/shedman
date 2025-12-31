package backends

import "os/exec"

type PacmanBackend struct {}

func NewPacmanBackend() *PacmanBackend {
	return &PacmanBackend{}
}

func (p *PacmanBackend) Name() string {
	return "pacman"
}

func (p *PacmanBackend) Sync() error {
	cmd := exec.Command("/usr/bin/pacman", "-Sy")
	return cmd.Run()
}
