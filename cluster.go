package main

import (
	"github.com/spf13/cobra"

	"arvika.pulcy.com/pulcy/yard/systemd"
	"arvika.pulcy.com/pulcy/yard/topics"
	"arvika.pulcy.com/pulcy/yard/topics/iptables"
)

var (
	cmdCluster = &cobra.Command{
		Use: "cluster",
		Run: showUsage,
	}
	cmdClusterUpdate = &cobra.Command{
		Use: "update",
		Run: runClusterUpdate,
	}
	clusterUpdateFlags = &topics.TopicFlags{}
)

func init() {
	// Etcd
	cmdClusterUpdate.Flags().StringVar(&clusterUpdateFlags.DiscoveryUrl, "discovery-url", "", "Full URL for setting up etcd member lists")
	cmdClusterUpdate.Flags().StringVar(&setupFlags.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the private IP")

	cmdMain.AddCommand(cmdCluster)
	cmdCluster.AddCommand(cmdClusterUpdate)
}

func runClusterUpdate(cmd *cobra.Command, args []string) {
	if clusterUpdateFlags.DiscoveryUrl == "" {
		Exitf("discovery-url missing\n")
	}
	if setupFlags.PrivateClusterDevice == "" {
		Exitf("private-cluster-device missing\n")
	}

	deps := &topics.TopicDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	if err := iptables.UpdatePrivateCluster(deps, clusterUpdateFlags); err != nil {
		Exitf("Update private cluster failed: %#v\n", err)
	}
}
