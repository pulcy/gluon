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

package iptables

import (
	"os"
	"os/exec"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/service/docker"
	"github.com/pulcy/gluon/service/sshd"
	"github.com/pulcy/gluon/templates"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	v4membersTemplate = "templates/iptables/ip4tables.members.sh.tmpl"
	v4membersPath     = "/home/core/ip4tables.members.sh"
	v4rulesTemplate   = "templates/iptables/ip4tables.rules.tmpl"
	v4rulesPath       = "/home/core/ip4tables.rules"
	v4serviceTemplate = "templates/iptables/ip4tables.service.tmpl"
	v4serviceName     = "ip4tables.service"
	v4servicePath     = "/etc/systemd/system/" + v4serviceName

	v6rulesTemplate   = "templates/iptables/ip6tables.rules.tmpl"
	v6rulesPath       = "/home/core/ip6tables.rules"
	v6serviceTemplate = "templates/iptables/ip6tables.service.tmpl"
	v6serviceName     = "ip6tables.service"
	v6servicePath     = "/etc/systemd/system/" + v6serviceName

	netfilterTemplate    = "templates/iptables/netfilter.service.tmpl"
	netfilterServiceName = "netfilter.service"
	netfilterServicePath = "/etc/systemd/system/" + netfilterServiceName

	rulesFileMode   = os.FileMode(0600)
	scriptFileMode  = os.FileMode(0700)
	serviceFileMode = os.FileMode(0644)
)

var (
	restartServiceNames = []string{
		netfilterServiceName,
		v4serviceName,
		v6serviceName,
		sshd.ServiceName,
		docker.ServiceName,
	}
)

func NewService() service.Service {
	return &iptablesService{}
}

type iptablesService struct{}

func (t *iptablesService) Name() string {
	return "iptables"
}

func (t *iptablesService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedV4Members, err := createV4Members(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedV4Rules, err := createV4Rules(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedV6Rules, err := createV6Rules(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedNetfilterService, err := createNetfilterService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedIp4tableService, err := createIp4tablesService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedIp6tableService, err := createIp6tablesService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedV4Rules || changedV6Rules || changedNetfilterService || changedIp4tableService || changedIp6tableService {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		for _, name := range restartServiceNames {
			if err := deps.Systemd.Restart(name); err != nil {
				return maskAny(err)
			}
		}
	}
	if flags.Force || changedV4Members {
		deps.Logger.Debugf("executing %s", v4membersPath)
		cmd := exec.Command(v4membersPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			deps.Logger.Errorf("%s failed:\n%s\n%#v\n", v4membersPath, string(output), err)
			return maskAny(err)
		}
	}

	return nil
}

func createV4Members(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", v4membersPath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ClusterMemberIPs     []string // Cluster specific IP addresses
		PrivateMemberIPs     []string // Private (host) specific IP addresses
		DockerSubnet         string
		PrivateClusterDevice string
	}{
		ClusterMemberIPs:     []string{},
		PrivateMemberIPs:     []string{},
		DockerSubnet:         flags.Docker.DockerSubnet,
		PrivateClusterDevice: flags.Network.PrivateClusterDevice,
	}
	for _, cm := range members {
		opts.ClusterMemberIPs = append(opts.ClusterMemberIPs, cm.ClusterIP)
		opts.PrivateMemberIPs = append(opts.PrivateMemberIPs, cm.PrivateHostIP)
	}
	changed, err := templates.Render(deps.Logger, v4membersTemplate, v4membersPath, opts, scriptFileMode)
	return changed, maskAny(err)
}

func createV4Rules(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", v4rulesPath)
	opts := struct {
		DockerSubnet            string
		RktSubnet               string
		WeaveSubnet             string
		PrivateClusterDevice    string
		ClusterSubnet           string
		KubernetesAPIServer     bool
		KubernetesAPIServerPort int
	}{
		DockerSubnet:            flags.Docker.DockerSubnet,
		RktSubnet:               flags.Rkt.RktSubnet,
		WeaveSubnet:             flags.Weave.IPRange,
		PrivateClusterDevice:    flags.Network.PrivateClusterDevice,
		ClusterSubnet:           flags.Network.ClusterSubnet,
		KubernetesAPIServer:     flags.Kubernetes.IsEnabled(),
		KubernetesAPIServerPort: flags.Kubernetes.APIServerPort,
	}
	changed, err := templates.Render(deps.Logger, v4rulesTemplate, v4rulesPath, opts, rulesFileMode)
	return changed, maskAny(err)
}

func createV6Rules(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", v6rulesPath)
	opts := struct {
		KubernetesAPIServer     bool
		KubernetesAPIServerPort int
	}{
		KubernetesAPIServer:     flags.Kubernetes.IsEnabled(),
		KubernetesAPIServerPort: flags.Kubernetes.APIServerPort,
	}
	changed, err := templates.Render(deps.Logger, v6rulesTemplate, v6rulesPath, opts, rulesFileMode)
	return changed, maskAny(err)
}

func createNetfilterService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", netfilterServicePath)
	changed, err := templates.Render(deps.Logger, netfilterTemplate, netfilterServicePath, nil, serviceFileMode)
	return changed, maskAny(err)
}

func createIp4tablesService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", v4servicePath)
	changed, err := templates.Render(deps.Logger, v4serviceTemplate, v4servicePath, nil, serviceFileMode)
	return changed, maskAny(err)
}

func createIp6tablesService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", v6servicePath)
	changed, err := templates.Render(deps.Logger, v6serviceTemplate, v6servicePath, nil, serviceFileMode)
	return changed, maskAny(err)
}
