package command

import (
	"strings"
)

type EnvCommand struct {
	Meta
}

func (c *EnvCommand) Run(args []string) int {
	// Write your code here

	return 0
}

func (c *EnvCommand) Synopsis() string {
	return "Print environment values"
}

func (c *EnvCommand) Help() string {
	helpText := `

`
	return strings.TrimSpace(helpText)
}
