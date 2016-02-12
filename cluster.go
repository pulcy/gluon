package main

import (
	"github.com/spf13/cobra"

	"github.com/pulcy/yard/systemd"
	"github.com/pulcy/yard/topics"
	"github.com/pulcy/yard/topics/iptables"
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
	cmdClusterUpdate.Flags().StringVar(&clusterUpdateFlags.DiscoveryURL, "discovery-url", "", "Full URL for setting up etcd member lists")
	cmdClusterUpdate.Flags().StringVar(&clusterUpdateFlags.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the private IP")

	cmdMain.AddCommand(cmdCluster)
	cmdCluster.AddCommand(cmdClusterUpdate)
}

func runClusterUpdate(cmd *cobra.Command, args []string) {
	if clusterUpdateFlags.PrivateClusterDevice == "" {
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
