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
	deps := &topics.TopicDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
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
