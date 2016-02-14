package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pulcy/gluon/systemd"
	"github.com/pulcy/gluon/topics"
	"github.com/pulcy/gluon/topics/docker"
	"github.com/pulcy/gluon/topics/env"
	"github.com/pulcy/gluon/topics/etcd2"
	"github.com/pulcy/gluon/topics/fleet"
	"github.com/pulcy/gluon/topics/gluon"
	"github.com/pulcy/gluon/topics/iptables"
	"github.com/pulcy/gluon/topics/sshd"
)

const (
	defaultDockerSubnet            = "172.17.0.0/16"
	defaultPrivateClusterDevice    = "eth1"
	defaultPrivateRegistryUrl      = "https://registry.pulcy.com"
	defaultPrivateRegistryUserName = "server"
	defaultPrivateRegistryPassword = ""
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = &topics.TopicFlags{}
)

func init() {
	initSetupFlags(cmdSetup.Flags(), setupFlags)
	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := setupFlags.SetupDefaults(); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	if setupFlags.PrivateClusterDevice == "" {
		Exitf("private-cluster-device missing\n")
	}

	deps := &topics.TopicDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	setupList := createTopics(args)
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
func createTopics(args []string) []topics.Topic {
	allTopics := []topics.Topic{
		env.NewTopic(),
		iptables.NewTopic(),
		docker.NewTopic(),
		etcd2.NewTopic(),
		fleet.NewTopic(),
		sshd.NewTopic(),
		gluon.NewTopic(),
	}
	list := []topics.Topic{}
	isSelected := func(name string) bool {
		if len(args) == 0 {
			return true
		}
		for _, a := range args {
			if name == a {
				return true
			}
		}
		return false
	}
	for _, t := range allTopics {
		if isSelected(t.Name()) {
			list = append(list, t)
		}
	}
	return list
}

func initSetupFlags(flags *pflag.FlagSet, f *topics.TopicFlags) {
	flags.BoolVar(&f.Force, "force", false, "Restart services, even if nothing has changed")
	// Gluon
	flags.StringVar(&f.GluonImage, "gluon-image", "", "Gluon docker image name")
	// Docker
	flags.StringVar(&f.DockerIP, "docker-ip", "", "IP address docker binds ports to")
	flags.StringVar(&f.DockerSubnet, "docker-subnet", defaultDockerSubnet, "Subnet used by docker")
	flags.StringVar(&f.PrivateRegistryUrl, "private-registry-url", defaultPrivateRegistryUrl, "URL of private docker registry")
	flags.StringVar(&f.PrivateRegistryUserName, "private-registry-username", defaultPrivateRegistryUserName, "Username for private registry")
	flags.StringVar(&f.PrivateRegistryPassword, "private-registry-password", defaultPrivateRegistryPassword, "Password for private registry")
	// Network
	flags.StringVar(&f.PrivateIP, "private-ip", "", "IP address of private network")
	flags.StringVar(&f.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the private IP")
	// Fleet
	flags.StringVar(&f.FleetMetadata, "fleet-metadata", "", "Metadata list for fleet")
}
