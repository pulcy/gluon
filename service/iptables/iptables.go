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

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	rulesTemplate        = "templates/iptables.rules.tmpl"
	rulesPath            = "/home/core/iptables.rules"
	serviceTemplate      = "templates/iptables.service.tmpl"
	serviceName          = "iptables.service"
	servicePath          = "/etc/systemd/system/" + serviceName
	netfilterTemplate    = "templates/netfilter.service.tmpl"
	netfilterServiceName = "netfilter.service"
	netfilterServicePath = "/etc/systemd/system/" + netfilterServiceName

	fileMode = os.FileMode(0755)
)

func NewService() service.Service {
	return &iptablesService{}
}

type iptablesService struct{}

func (t *iptablesService) Name() string {
	return "iptables"
}

func (t *iptablesService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedRules, err := createRules(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedNetfilterService, err := createNetfilterService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedIptableService, err := createIptablesService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedRules || changedNetfilterService || changedIptableService {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(netfilterServiceName); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createRules(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", rulesPath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ClusterMemberIPs     []string
		DockerSubnet         string
		PrivateClusterDevice string
	}{
		ClusterMemberIPs:     []string{},
		DockerSubnet:         flags.DockerSubnet,
		PrivateClusterDevice: flags.PrivateClusterDevice,
	}
	for _, cm := range members {
		opts.ClusterMemberIPs = append(opts.ClusterMemberIPs, cm.PrivateIP)
	}
	changed, err := templates.Render(rulesTemplate, rulesPath, opts, fileMode)
	return changed, maskAny(err)
}

func createNetfilterService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", netfilterServicePath)
	changed, err := templates.Render(netfilterTemplate, netfilterServicePath, nil, fileMode)
	return changed, maskAny(err)
}

func createIptablesService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	changed, err := templates.Render(serviceTemplate, servicePath, nil, fileMode)
	return changed, maskAny(err)
}
