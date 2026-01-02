package output

import (
	"strings"

	"github.com/spf13/cobra"
)

// SetupColoredHelp configures Cobra with a colored help template
func SetupColoredHelp(cmd *cobra.Command) {
	cobra.AddTemplateFunc("cyan", func(s string) string {
		return Colorize(Cyan, s)
	})
	cobra.AddTemplateFunc("green", func(s string) string {
		return Colorize(Green, s)
	})
	cobra.AddTemplateFunc("yellow", func(s string) string {
		return Colorize(Yellow, s)
	})
	cobra.AddTemplateFunc("bold", func(s string) string {
		return Colorize(Bold, s)
	})
	cobra.AddTemplateFunc("magenta", func(s string) string {
		return Colorize(Magenta, s)
	})

	cmd.SetUsageTemplate(coloredUsageTemplate)
	cmd.SetHelpTemplate(coloredHelpTemplate)
}

var coloredUsageTemplate = `{{bold "Usage:"}}{{if .Runnable}}
  {{cyan .UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{cyan .CommandPath}} {{cyan "[command]"}}{{end}}{{if gt (len .Aliases) 0}}

{{bold "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{bold "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{bold "Available Commands:"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{green (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{bold .Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{green (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{bold "Additional Commands:"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{green (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{bold "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces | colorizeFlags}}{{end}}{{if .HasAvailableInheritedFlags}}

{{bold "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces | colorizeFlags}}{{end}}{{if .HasHelpSubCommands}}

{{bold "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{green (rpad .CommandPath .CommandPathPadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{cyan .CommandPath}} {{cyan "[command] --help"}}" for more information about a command.{{end}}
`

var coloredHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

func init() {
	cobra.AddTemplateFunc("colorizeFlags", colorizeFlags)
}

// colorizeFlags adds color to flag names in the flag usage string
func colorizeFlags(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Find the flags part (before the description)
		// Format is typically: "  -s, --long-flag type   description"
		trimmed := strings.TrimLeft(line, " ")
		leadingSpaces := line[:len(line)-len(trimmed)]

		// Find where the description starts (after multiple spaces)
		descIdx := strings.Index(trimmed, "   ")
		if descIdx == -1 {
			descIdx = len(trimmed)
		}

		flagPart := trimmed[:descIdx]
		descPart := ""
		if descIdx < len(trimmed) {
			descPart = trimmed[descIdx:]
		}

		// Color the flag part
		// Handle -s, --long patterns
		coloredFlag := ""
		for j := 0; j < len(flagPart); j++ {
			ch := flagPart[j]
			if ch == '-' {
				// Find the end of this flag
				end := j + 1
				for end < len(flagPart) && flagPart[end] != ' ' && flagPart[end] != ',' {
					end++
				}
				coloredFlag += Colorize(Yellow, flagPart[j:end])
				j = end - 1
			} else {
				coloredFlag += string(ch)
			}
		}

		lines[i] = leadingSpaces + coloredFlag + descPart
	}
	return strings.Join(lines, "\n")
}
