package main

import (
	"github.com/spf13/cobra"

	"github.com/pulcy/yard/systemd"
	"github.com/pulcy/yard/topics"
	"github.com/pulcy/yard/topics/yard"
)

var (
	cmdUpdate = &cobra.Command{
		Use:   "update",
		Short: "Update the yard service",
		Run:   runUpdate,
	}
	updateFlags = &topics.TopicFlags{}
)

func init() {
	initSetupFlags(cmdUpdate.Flags(), updateFlags)
	cmdMain.AddCommand(cmdUpdate)
}

func runUpdate(cmd *cobra.Command, args []string) {
	yardVersion := projectVersion
	switch len(args) {
	case 0:
		// Ok
	case 1:
		yardVersion = args[0]
	default:
		Exitf("Invalid arguments\n")
	}
	deps := &topics.TopicDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}
	if err := yard.Setup(deps, updateFlags, yardVersion); err != nil {
		Exitf("Update failed: %#v\n", err)
	}
}
