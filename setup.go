// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/spf13/cobra"

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
	defaultDockerSubnet         = "172.17.0.0/16"
	defaultPrivateClusterDevice = "eth1"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = &service.ServiceFlags{}
)

func init() {
	cmdSetup.Flags().BoolVar(&setupFlags.Force, "force", false, "Restart services, even if nothing has changed")
	// Gluon
	cmdSetup.Flags().StringVar(&setupFlags.GluonImage, "gluon-image", "", "Gluon docker image name")
	// Docker
	cmdSetup.Flags().StringVar(&setupFlags.DockerIP, "docker-ip", "", "IP address docker binds ports to")
	cmdSetup.Flags().StringVar(&setupFlags.DockerSubnet, "docker-subnet", defaultDockerSubnet, "Subnet used by docker")
	cmdSetup.Flags().StringVar(&setupFlags.PrivateRegistryUrl, "private-registry-url", "", "URL of private docker registry")
	cmdSetup.Flags().StringVar(&setupFlags.PrivateRegistryUserName, "private-registry-username", "", "Username for private registry")
	cmdSetup.Flags().StringVar(&setupFlags.PrivateRegistryPassword, "private-registry-password", "", "Password for private registry")
	// Network
	cmdSetup.Flags().StringVar(&setupFlags.ClusterIP, "private-ip", "", "IP address of this host in the cluster network")
	cmdSetup.Flags().StringVar(&setupFlags.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the cluster IP")
	// Fleet
	cmdSetup.Flags().StringVar(&setupFlags.FleetMetadata, "fleet-metadata", "", "Metadata list for fleet")
	// ETCD
	cmdSetup.Flags().StringVar(&setupFlags.EtcdClusterState, "etcd-cluster-state", "", "State of the ETCD cluster new|existing")
	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := setupFlags.SetupDefaults(); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	assertArgIsSet(setupFlags.GluonImage, "--gluon-image")
	assertArgIsSet(setupFlags.DockerIP, "--docker-ip")
	assertArgIsSet(setupFlags.DockerSubnet, "--docker-subnet")
	assertArgIsSet(setupFlags.ClusterIP, "--private-ip")
	assertArgIsSet(setupFlags.PrivateClusterDevice, "--private-cluster-device")

	deps := service.ServiceDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	services := []service.Service{
		env.NewService(),
		iptables.NewService(),
		docker.NewService(),
		etcd2.NewService(),
		fleet.NewService(),
		sshd.NewService(),
		gluon.NewService(),
	}
	for i, t := range services {
		log.Info("%d/%d Setup %s", i+1, len(services), t.Name())
		if err := t.Setup(deps, setupFlags); err != nil {
			Exitf("Setup %s failed: %#v\n", t.Name(), err)
		}
	}
	log.Info("Done")
}
