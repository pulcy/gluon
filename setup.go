package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"arvika.pulcy.com/iggi/yard/systemd"
	"arvika.pulcy.com/iggi/yard/topics"
	"arvika.pulcy.com/iggi/yard/topics/weave"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
)

func init() {
	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	systemdClient, err := systemd.NewSystemdClient()
	if err != nil {
		Exitf("Setup cannot create systemd client: %#v\n", err)
	}
	deps := &topics.TopicDependencies{
		Systemd: systemdClient,
	}

	for _, t := range createTopics() {
		fmt.Printf("Setup %s\n", t.Name())
		if err := t.Setup(deps); err != nil {
			Exitf("Setup %s failed: %#v\n", t.Name(), err)
		}
	}
}

// Topics creates an ordered list of topics o provision
func createTopics() []topics.Topic {
	return []topics.Topic{
		weave.NewTopic(),
	}
}
