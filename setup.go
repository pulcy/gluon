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
	setupFlags = &topics.TopicFlags{}
)

func init() {
	cmdSetup.Flags().StringVar(&setupFlags.WeavePassword, "weave-password", def("WEAVE_PASSWORD", ""), "Password protecting inter-host weave traffic")
	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	deps := &topics.TopicDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	setupList := createTopics()
	for _, t := range setupList {
		if err := t.Defaults(setupFlags); err != nil {
			Exitf("Defaults %s failed: %#v\n", t.Name(), err)
		}
	}
	for _, t := range setupList {
		fmt.Printf("Setup %s\n", t.Name())
		if err := t.Setup(deps, setupFlags); err != nil {
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
