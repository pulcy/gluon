package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/service/docker"
	"github.com/pulcy/gluon/service/env"
	"github.com/pulcy/gluon/service/etcd2"
	"github.com/pulcy/gluon/service/fleet"
	"github.com/pulcy/gluon/service/gluon"
	"github.com/pulcy/gluon/service/iptables"
	"github.com/pulcy/gluon/service/sshd"
	"github.com/pulcy/gluon/systemd"
)

const (
	defaultDockerSubnet            = "172.17.0.0/16"
	defaultPrivateClusterDevice    = "eth1"
	defaultPrivateRegistryUrl      = ""
	defaultPrivateRegistryUserName = "server"
	defaultPrivateRegistryPassword = ""
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = &service.ServiceFlags{}
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

	deps := service.ServiceDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	setupList := createServices(args)
	for _, t := range setupList {
		fmt.Printf("Setup %s\n", t.Name())
		if err := t.Setup(deps, setupFlags); err != nil {
			Exitf("Setup %s failed: %#v\n", t.Name(), err)
		}
	}
}

// Service creates an ordered list of services to configure
func createServices(args []string) []service.Service {
	allServices := []service.Service{
		env.NewService(),
		iptables.NewService(),
		docker.NewService(),
		etcd2.NewService(),
		fleet.NewService(),
		sshd.NewService(),
		gluon.NewService(),
	}
	list := []service.Service{}
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
	for _, s := range allServices {
		if isSelected(s.Name()) {
			list = append(list, s)
		}
	}
	return list
}

func initSetupFlags(flags *pflag.FlagSet, f *service.ServiceFlags) {
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
