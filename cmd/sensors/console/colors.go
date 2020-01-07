package console

import "github.com/fatih/color"

// Available ANSI colors
var (
	Yellow = color.New(color.FgYellow).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Green  = color.New(color.FgGreen).SprintFunc()
	White  = color.New(color.FgHiWhite).SprintFunc()
	Bold   = color.New(color.Bold).SprintFunc()
)
