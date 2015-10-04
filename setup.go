package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"arvika.pulcy.com/pulcy/yard/systemd"
	"arvika.pulcy.com/pulcy/yard/topics"
	"arvika.pulcy.com/pulcy/yard/topics/hosts"
	"arvika.pulcy.com/pulcy/yard/topics/stunnel"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = &topics.TopicFlags{}
)

func init() {
	// Stunnel
	cmdSetup.Flags().StringVar(&setupFlags.StunnelPemPassphrase, "stunnel-pem-passphrase", def("STUNNEL_PEM_PASSPHRASE", ""), "Passphrase used to open stunnel.pem.gpg")
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
		hosts.NewTopic(),
		stunnel.NewTopic(),
	}
}
