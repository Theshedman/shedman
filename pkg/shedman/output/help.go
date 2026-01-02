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
		// Color the flag part (starts with -)
		if strings.Contains(line, "--") {
			// Find and color --flagname
			parts := strings.SplitN(line, " ", 2)
			if len(parts) >= 1 {
				flagPart := parts[0]
				// Color short and long flags
				flagPart = strings.Replace(flagPart, "-", Colorize(Yellow, "-"), 2)
				if len(parts) == 2 {
					lines[i] = flagPart + " " + parts[1]
				} else {
					lines[i] = flagPart
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}
