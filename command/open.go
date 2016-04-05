package command

import (
	"os"
	"strings"

	"github.com/skratchdot/open-golang/open"
)

type OpenCommand struct {
	Meta
}

func (c *OpenCommand) Run(args []string) int {
	open.Run(os.Getenv("CASSIOPEIA_ANALYZER_URL"))

	return 0
}

func (c *OpenCommand) Synopsis() string {
	return "Open local analyzer in your browser"
}

func (c *OpenCommand) Help() string {
	helpText := `

`
	return strings.TrimSpace(helpText)
}
