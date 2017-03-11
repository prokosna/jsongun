package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
	"github.com/prokosna/jsongun/internal/command"
)

func main() {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}
	c := cli.NewCLI("jsongun", "0.0.1")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"fire": func() (cli.Command, error) {
			return &command.FireCommand{
				Ui: &cli.ColoredUi{
					Ui:          ui,
					OutputColor: cli.UiColorCyan,
				},
			}, nil
		},
	}
	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	os.Exit(exitStatus)
}
