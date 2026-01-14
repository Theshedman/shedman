package mirror

import (
	"fmt"
	"os/exec"
	"strings"
)

// ReflectorBackend implements MirrorBackend using reflector
type ReflectorBackend struct{}

func NewReflectorBackend() *ReflectorBackend {
	return &ReflectorBackend{}
}

func (r *ReflectorBackend) Name() string { return "reflector" }

func (r *ReflectorBackend) List() ([]Mirror, error) {
	// Parsing /etc/pacman.d/mirrorlist
	return []Mirror{}, nil
}

func (r *ReflectorBackend) Test() ([]Mirror, error) {
	return []Mirror{}, nil
}

func (r *ReflectorBackend) Select(topN int, countries []string, sort string) error {
	args := []string{"--save", "/etc/pacman.d/mirrorlist", "--latest", fmt.Sprintf("%d", topN), "--protocol", "https", "--sort", sort}

	if len(countries) > 0 {
		args = append(args, "--country", strings.Join(countries, ","))
	}

	cmd := exec.Command("reflector", args...)
	return cmd.Run()
}

func (r *ReflectorBackend) IsAvailable() bool {
	_, err := exec.LookPath("reflector")
	return err == nil
}
