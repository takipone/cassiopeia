package command

import (
	"testing"

	"github.com/mitchellh/cli"
)

func TestSetupCommand_implement(t *testing.T) {
	var _ cli.Command = &SetupCommand{}
}
