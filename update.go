package main

import (
	"github.com/spf13/cobra"

	"arvika.pulcy.com/pulcy/yard/systemd"
	"arvika.pulcy.com/pulcy/yard/topics"
	"arvika.pulcy.com/pulcy/yard/topics/yard"
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
