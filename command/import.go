package command

import (
	"strings"
)

type ImportCommand struct {
	Meta
}

func (c *ImportCommand) Run(args []string) int {
	// Write your code here

	return 0
}

func (c *ImportCommand) Synopsis() string {
	return "Import records from cloud transit to local analyzer(execute processor)"
}

func (c *ImportCommand) Help() string {
	helpText := `

`
	return strings.TrimSpace(helpText)
}
