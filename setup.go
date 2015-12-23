package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"arvika.pulcy.com/pulcy/yard/systemd"
	"arvika.pulcy.com/pulcy/yard/topics"
	"arvika.pulcy.com/pulcy/yard/topics/docker"
	"arvika.pulcy.com/pulcy/yard/topics/env"
	"arvika.pulcy.com/pulcy/yard/topics/iptables"
	"arvika.pulcy.com/pulcy/yard/topics/sshd"
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

	if err := setupFlags.SetupDefaults(projectVersion); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
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
		sshd.NewTopic(),
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
	flags.StringVar(&f.YardPassphrase, "yard-passphrase", "", "Passphrase for yard image")
	flags.StringVar(&f.YardImage, "yard-image", "", "Yard docker image name")
	// ETCD
	flags.StringVar(&f.DiscoveryURL, "discovery-url", "", "ETCD discovery URL")
	// Docker
	flags.StringVar(&f.DockerIP, "docker-ip", "", "IP address docker binds ports to")
	flags.StringVar(&f.DockerSubnet, "docker-subnet", defaultDockerSubnet, "Subnet used by docker")
	flags.StringVar(&f.PrivateRegistryUrl, "private-registry-url", defaultPrivateRegistryUrl, "URL of private docker registry")
	flags.StringVar(&f.PrivateRegistryUserName, "private-registry-username", defaultPrivateRegistryUserName, "Username for private registry")
	flags.StringVar(&f.PrivateRegistryPassword, "private-registry-password", defaultPrivateRegistryPassword, "Password for private registry")
	// IPTables
	flags.StringVar(&f.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the private IP")
}
