package command

import (
	"testing"

	"github.com/mitchellh/cli"
)

func TestFetchCommand_implement(t *testing.T) {
	var _ cli.Command = &FetchCommand{}
}
