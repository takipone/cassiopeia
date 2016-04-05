package main

import (
	"github.com/mitchellh/cli"
	"github.com/takipone/cassiopeia/command"
)

func Commands(meta *command.Meta) map[string]cli.CommandFactory {
	return map[string]cli.CommandFactory{
		"setup": func() (cli.Command, error) {
			return &command.SetupCommand{
				Meta: *meta,
			}, nil
		},
		"list": func() (cli.Command, error) {
			return &command.ListCommand{
				Meta: *meta,
			}, nil
		},
		"fetch": func() (cli.Command, error) {
			return &command.FetchCommand{
				Meta: *meta,
			}, nil
		},
		"pull": func() (cli.Command, error) {
			return &command.PullCommand{
				Meta: *meta,
			}, nil
		},
		"open": func() (cli.Command, error) {
			return &command.OpenCommand{
				Meta: *meta,
			}, nil
		},
		"env": func() (cli.Command, error) {
			return &command.EnvCommand{
				Meta: *meta,
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Meta:     *meta,
				Version:  Version,
				Revision: GitCommit,
				Name:     Name,
			}, nil
		},
	}
}
