package pacman

import (
	"bufio"
	"strings"

	"github.com/theshedman/shedman/internal/util"
)

func parsePacmanSize(output, key string) int64 {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, ":"); idx > 0 {
			k := strings.TrimSpace(line[:idx])
			if k == key {
				val := strings.TrimSpace(line[idx+1:])
				return util.ParseSize(val)
			}
		}
	}
	return 0
}
