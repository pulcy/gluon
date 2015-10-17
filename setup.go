package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"arvika.pulcy.com/pulcy/yard/systemd"
	"arvika.pulcy.com/pulcy/yard/topics"
	"arvika.pulcy.com/pulcy/yard/topics/docker"
	"arvika.pulcy.com/pulcy/yard/topics/iptables"
)

const (
	defaultDockerSubnet         = "172.17.0.0/16"
	defaultPrivateClusterDevice = "eth1"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = &topics.TopicFlags{}
)

func init() {
	// Etcd
	cmdSetup.Flags().StringVar(&setupFlags.DiscoveryUrl, "discovery-url", "", "Full URL for setting up etcd member lists")
	// Docker
	cmdSetup.Flags().StringVar(&setupFlags.DockerIP, "docker-ip", "", "IP address docker binds ports to")
	cmdSetup.Flags().StringVar(&setupFlags.DockerSubnet, "docker-subnet", defaultDockerSubnet, "Subnet used by docker")
	// IPTables
	cmdSetup.Flags().StringVar(&setupFlags.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the private IP")

	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	if setupFlags.DiscoveryUrl == "" {
		Exitf("discovery-url missing\n")
	}
	if setupFlags.DockerIP == "" {
		Exitf("docker-ip missing\n")
	}
	if setupFlags.DockerSubnet == "" {
		Exitf("docker-subnet missing\n")
	}
	if setupFlags.PrivateClusterDevice == "" {
		Exitf("private-cluster-device missing\n")
	}

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
		iptables.NewTopic(),
		docker.NewTopic(),
	}
}
