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
	"github.com/pulcy/gluon/service/binaries"
	"github.com/pulcy/gluon/service/consul"
	"github.com/pulcy/gluon/service/docker"
	"github.com/pulcy/gluon/service/env"
	"github.com/pulcy/gluon/service/etcd"
	"github.com/pulcy/gluon/service/fleet"
	"github.com/pulcy/gluon/service/gluon"
	"github.com/pulcy/gluon/service/iptables"
	"github.com/pulcy/gluon/service/journal"
	"github.com/pulcy/gluon/service/rkt"
	"github.com/pulcy/gluon/service/sshd"
	"github.com/pulcy/gluon/service/vault"
	"github.com/pulcy/gluon/service/weave"
	"github.com/pulcy/gluon/systemd"
)

const (
	defaultDockerSubnet         = "172.17.0.0/16"
	defaultPrivateClusterDevice = "eth1"

	defaultFleetAgentTTL                = "30s"
	defaultFleetDisableEngine           = false
	defaultFleetDisableWatches          = true
	defaultFleetEngineReconcileInterval = 10
	defaultFleetTokenLimit              = 50
	defaultWeaveHostname                = "hosts.weave.local"
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
	cmdSetup.Flags().StringVar(&setupFlags.Docker.DockerIP, "docker-ip", "", "IP address docker binds ports to")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.DockerSubnet, "docker-subnet", defaultDockerSubnet, "Subnet used by docker")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.PrivateRegistryUrl, "private-registry-url", "", "URL of private docker registry")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.PrivateRegistryUserName, "private-registry-username", "", "Username for private registry")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.PrivateRegistryPassword, "private-registry-password", "", "Password for private registry")
	// Network
	cmdSetup.Flags().StringVar(&setupFlags.Network.ClusterIP, "private-ip", "", "IP address of this host in the cluster network")
	cmdSetup.Flags().StringVar(&setupFlags.Network.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the cluster IP")
	// Fleet
	cmdSetup.Flags().StringVar(&setupFlags.Fleet.Metadata, "fleet-metadata", "", "Metadata list for fleet")
	cmdSetup.Flags().StringVar(&setupFlags.Fleet.AgentTTL, "fleet-agent-ttl", defaultFleetAgentTTL, "agent_ttl option for fleet")
	cmdSetup.Flags().BoolVar(&setupFlags.Fleet.DisableEngine, "fleet-disable-engine", defaultFleetDisableEngine, "disable_engine option for fleet")
	cmdSetup.Flags().BoolVar(&setupFlags.Fleet.DisableWatches, "fleet-disable-watches", defaultFleetDisableWatches, "disable_watches option for fleet")
	cmdSetup.Flags().IntVar(&setupFlags.Fleet.EngineReconcileInterval, "fleet-engine-reconcile-interval", defaultFleetEngineReconcileInterval, "engine_reconcile_interval option for fleet")
	cmdSetup.Flags().IntVar(&setupFlags.Fleet.TokenLimit, "fleet-token-limit", defaultFleetTokenLimit, "token_limit option for fleet")
	// ETCD
	cmdSetup.Flags().StringVar(&setupFlags.Etcd.ClusterState, "etcd-cluster-state", "", "State of the ETCD cluster new|existing")
	// Weave
	cmdSetup.Flags().StringVar(&setupFlags.Weave.Seed, "weave-seed", "", "SEED of the weave network")
	cmdSetup.Flags().StringVar(&setupFlags.Weave.Hostname, "weave-hostname", defaultWeaveHostname, "DNS name for exposed host")

	cmdMain.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	showVersion(cmd, args)

	if err := setupFlags.SetupDefaults(log); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	assertArgIsSet(setupFlags.GluonImage, "--gluon-image")
	assertArgIsSet(setupFlags.Docker.DockerIP, "--docker-ip")
	assertArgIsSet(setupFlags.Docker.DockerSubnet, "--docker-subnet")
	assertArgIsSet(setupFlags.Network.ClusterIP, "--private-ip")
	assertArgIsSet(setupFlags.Network.PrivateClusterDevice, "--private-cluster-device")

	deps := service.ServiceDependencies{
		Systemd: systemd.NewSystemdClient(log),
		Logger:  log,
	}

	services := []service.Service{
		binaries.NewService(),
		env.NewService(),
		iptables.NewService(),
		journal.NewService(),
		docker.NewService(),
		rkt.NewService(),
		vault.NewService(),
		weave.NewService(),
		consul.NewService(),
		etcd.NewService(),
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
