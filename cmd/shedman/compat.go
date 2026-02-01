package main

import "strings"

func rewritePacmanArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	flags, consumed := collectShortFlags(args)
	if consumed == 0 {
		return args
	}
	rest := args[consumed:]

	if strings.Contains(flags, "S") {
		if strings.Contains(flags, "g") {
			if len(rest) == 0 {
				return []string{"group", "list"}
			}
			return append([]string{"group", "info"}, rest...)
		}

		refresh := countFlag(flags, 'y') >= 2
		switch {
		case strings.Contains(flags, "y") && strings.Contains(flags, "u"):
			cmd := []string{"update"}
			if refresh {
				cmd = append(cmd, "--refresh")
			}
			return append(cmd, rest...)
		case strings.Contains(flags, "s"):
			return append([]string{"search"}, rest...)
		case strings.Contains(flags, "i"):
			return append([]string{"info"}, rest...)
		case strings.Contains(flags, "y"):
			cmd := []string{"sync"}
			if refresh {
				cmd = append(cmd, "--refresh")
			}
			return append(cmd, rest...)
		default:
			return append([]string{"install"}, rest...)
		}
	}

	if strings.Contains(flags, "R") {
		newArgs := []string{"remove"}
		if strings.Contains(flags, "c") {
			newArgs = append(newArgs, "--cascade")
		}
		if strings.Contains(flags, "n") {
			newArgs = append(newArgs, "--nosave")
		}
		if strings.Contains(flags, "s") {
			newArgs = append(newArgs, "--recursive")
		}
		return append(newArgs, rest...)
	}

	if strings.Contains(flags, "Q") {
		if strings.Contains(flags, "s") {
			return append([]string{"search", "--installed"}, rest...)
		}
		if strings.Contains(flags, "i") {
			return append([]string{"info"}, rest...)
		}
		if strings.Contains(flags, "l") {
			return append([]string{"files"}, rest...)
		}
	}

	return args
}

func collectShortFlags(args []string) (string, int) {
	var flags strings.Builder
	consumed := 0
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			flags.WriteString(strings.TrimPrefix(arg, "-"))
			consumed++
			continue
		}
		break
	}
	return flags.String(), consumed
}

func countFlag(flags string, target rune) int {
	count := 0
	for _, flag := range flags {
		if flag == target {
			count++
		}
	}
	return count
}
