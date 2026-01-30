package mirror

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/alpm"
	"github.com/theshedman/shedman/pkg/executor"
)

// ReflectorBackend implements MirrorBackend using reflector
type ReflectorBackend struct {
	exec           executor.Executor
	mirrorlistPath string
	testTimeout    time.Duration
	httpClient     *http.Client
}

func NewReflectorBackend() *ReflectorBackend {
	return &ReflectorBackend{
		exec:           &executor.RealExecutor{},
		mirrorlistPath: "/etc/pacman.d/mirrorlist",
		testTimeout:    3 * time.Second,
	}
}

func NewReflectorBackendWithExecutor(exec executor.Executor, mirrorlistPath string, testTimeout time.Duration) *ReflectorBackend {
	if exec == nil {
		exec = &executor.RealExecutor{}
	}
	if mirrorlistPath == "" {
		mirrorlistPath = "/etc/pacman.d/mirrorlist"
	}
	if testTimeout <= 0 {
		testTimeout = 3 * time.Second
	}
	return &ReflectorBackend{
		exec:           exec,
		mirrorlistPath: mirrorlistPath,
		testTimeout:    testTimeout,
	}
}

func (r *ReflectorBackend) Name() string { return "reflector" }

func (r *ReflectorBackend) List() ([]Mirror, error) {
	file, err := os.Open(r.mirrorlistPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var mirrors []Mirror
	var currentCountry string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "##") {
			currentCountry = strings.TrimSpace(strings.TrimPrefix(line, "##"))
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "Server") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			url := strings.TrimSpace(parts[1])
			if url == "" {
				continue
			}
			mirrors = append(mirrors, Mirror{
				URL:     url,
				Country: currentCountry,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return mirrors, nil
}

func (r *ReflectorBackend) Test() ([]Mirror, error) {
	mirrors, err := r.List()
	if err != nil {
		return nil, err
	}
	if len(mirrors) == 0 {
		return nil, fmt.Errorf("no mirrors found in %s", r.mirrorlistPath)
	}

	conf := alpm.DefaultPacmanConf()
	arch := conf.GetArchitecture()
	client := r.httpClient
	if client == nil {
		timeout := r.testTimeout
		if timeout <= 0 {
			timeout = 3 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	var tested []Mirror
	for _, mirror := range mirrors {
		testURL := buildTestURL(mirror.URL, arch)
		start := time.Now()
		if err := probeMirror(client, testURL); err != nil {
			continue
		}
		mirror.Speed = time.Since(start)
		tested = append(tested, mirror)
	}

	if len(tested) == 0 {
		return nil, fmt.Errorf("failed to reach any mirrors")
	}

	sort.Slice(tested, func(i, j int) bool {
		return tested[i].Speed < tested[j].Speed
	})

	return tested, nil
}

func (r *ReflectorBackend) Select(topN int, countries []string, sort string) error {
	if r.exec == nil {
		r.exec = &executor.RealExecutor{}
	}

	args := []string{"--save", r.mirrorlistPath, "--latest", fmt.Sprintf("%d", topN), "--protocol", "https", "--sort", sort}

	if len(countries) > 0 {
		args = append(args, "--country", strings.Join(countries, ","))
	}

	cmd := r.exec.Command("reflector", args...)
	return cmd.Run()
}

func (r *ReflectorBackend) IsAvailable() bool {
	_, err := exec.LookPath("reflector")
	return err == nil
}

func buildTestURL(baseURL, arch string) string {
	url := strings.ReplaceAll(baseURL, "$repo", "core")
	url = strings.ReplaceAll(url, "$arch", arch)
	url = strings.TrimRight(url, "/")
	return url + "/core.db"
}

func probeMirror(client *http.Client, url string) error {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return nil
		}
	}

	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("unexpected status %d", resp.StatusCode)
}
