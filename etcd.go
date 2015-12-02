package main

import (
	"github.com/spf13/cobra"

	"arvika.pulcy.com/pulcy/yard/systemd"
	"arvika.pulcy.com/pulcy/yard/topics"
	"arvika.pulcy.com/pulcy/yard/topics/etcd2"
)

var (
	cmdEtcd = &cobra.Command{
		Use: "etcd",
		Run: showUsage,
	}
	cmdEtcdUpdate = &cobra.Command{
		Use: "update",
		Run: runEtcdUpdate,
	}
	etcdUpdateFlags = &topics.TopicFlags{}
)

func init() {
	// Etcd
	cmdEtcdUpdate.Flags().StringVar(&etcdUpdateFlags.DiscoveryURL, "discovery-url", "", "Full URL for setting up etcd member lists")
	cmdEtcdUpdate.Flags().StringVar(&etcdUpdateFlags.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the private IP")

	cmdMain.AddCommand(cmdEtcd)
	cmdEtcd.AddCommand(cmdEtcdUpdate)
}

func runEtcdUpdate(cmd *cobra.Command, args []string) {
	if etcdUpdateFlags.PrivateClusterDevice == "" {
		Exitf("private-cluster-device missing\n")
	}

	deps := &topics.TopicDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	if err := etcd2.ConfigureEtcd2(deps, etcdUpdateFlags); err != nil {
		Exitf("Reconfigure etcd2 failed: %#v\n", err)
	}
}
