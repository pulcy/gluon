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
	"github.com/pulcy/gluon/service/gluon"
	"github.com/pulcy/gluon/service/iptables"
	"github.com/pulcy/gluon/service/journal"
	"github.com/pulcy/gluon/service/kubernetes"
	"github.com/pulcy/gluon/service/rkt"
	"github.com/pulcy/gluon/service/sshd"
	"github.com/pulcy/gluon/service/vault"
	"github.com/pulcy/gluon/service/weave"
	"github.com/pulcy/gluon/systemd"
)

const (
	defaultDockerSubnet         = "172.17.0.0/16"
	defaultRktSubnet            = "172.22.0.0/16"
	defaultPrivateClusterDevice = "eth1"

	defaultWeaveHostname = "hosts.weave.local"
)

var (
	cmdSetup = &cobra.Command{
		Use: "setup",
		Run: runSetup,
	}
	setupFlags = &service.ServiceFlags{}
)

func init() {
	LoadEnv()
	cmdSetup.Flags().BoolVar(&setupFlags.Force, "force", false, "Restart services, even if nothing has changed")
	// Gluon
	cmdSetup.Flags().StringVar(&setupFlags.GluonImage, "gluon-image", "", "Gluon docker image name")
	cmdSetup.Flags().StringVar(&setupFlags.VaultMonkeyImage, "vault-monkey-image", "", "VaultMonkey docker image name")
	// Docker
	cmdSetup.Flags().StringVar(&setupFlags.Docker.DockerIP, "docker-ip", "", "IP address docker binds ports to")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.DockerSubnet, "docker-subnet", defaultDockerSubnet, "Subnet used by docker")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.PrivateRegistryUrl, "private-registry-url", "", "URL of private docker registry")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.PrivateRegistryUserName, "private-registry-username", "", "Username for private registry")
	cmdSetup.Flags().StringVar(&setupFlags.Docker.PrivateRegistryPassword, "private-registry-password", "", "Password for private registry")
	// Rkt
	cmdSetup.Flags().StringVar(&setupFlags.Rkt.RktSubnet, "rkt-subnet", defaultRktSubnet, "Subnet used by rkt")
	// Network
	cmdSetup.Flags().StringVar(&setupFlags.Network.ClusterIP, "private-ip", "", "IP address of this host in the cluster network")
	cmdSetup.Flags().StringVar(&setupFlags.Network.PrivateClusterDevice, "private-cluster-device", defaultPrivateClusterDevice, "Network device connected to the cluster IP")
	// ETCD
	cmdSetup.Flags().StringVar(&setupFlags.Etcd.ClusterState, "etcd-cluster-state", "", "State of the ETCD cluster new|existing")
	cmdSetup.Flags().BoolVar(&setupFlags.Etcd.UseVaultCA, "etcd-use-vault-ca", defaultEtcdUseVaultCA(), "If set, use vault to create peer (and optional client) TLS certificates")
	cmdSetup.Flags().BoolVar(&setupFlags.Etcd.SecureClients, "etcd-secure-clients", defaultEtcdSecureClients(), "If set, force clients to connect over TLS")
	// Kubernetes
	cmdSetup.Flags().BoolVar(&setupFlags.Kubernetes.Enabled, "k8s-enabled", defaultKubernetesEnabled(), "If set, kubernetes will be installed")
	cmdSetup.Flags().StringVar(&setupFlags.Kubernetes.APIDNSName, "k8s-api-dns-name", defaultKubernetesAPIDNSName(), "Alternate name of the Kubernetes API server")
	cmdSetup.Flags().StringVar(&setupFlags.Kubernetes.Metadata, "k8s-metadata", "", "Metadata list for kubelet")
	// Vault
	cmdSetup.Flags().StringVar(&setupFlags.Vault.VaultImage, "vault-image", "", "Pulcy Vault docker image name")
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
		// The order of entries is relevant!
		binaries.NewService(),
		env.NewService(),
		iptables.NewService(),
		journal.NewService(),
		docker.NewService(),
		rkt.NewService(),
		weave.NewService(),
		consul.NewService(),
		vault.NewService(),
		etcd.NewService(),
		kubernetes.NewService(),
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
