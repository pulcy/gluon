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

package weave

import (
	"os"
	"os/exec"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	weaveServiceName       = "weave.service"
	weaveServiceTmpl       = "templates/weave/" + weaveServiceName + ".tmpl"
	weaveServicePath       = "/etc/systemd/system/" + weaveServiceName
	cniConfTmpl            = "templates/weave/cni.conf"
	cniConfPath            = "/etc/cni/net.d/10-weave.conf"
	cniPluginDir           = "/opt/cni/bin"
	rktNetworkConfPath     = "/etc/rkt/net.d/10-default-weave.conf"
	rktNetworkConfTemplate = "templates/weave/rkt-net-weave.conf.tmpl"

	serviceFileMode = os.FileMode(0644)
	configFileMode  = os.FileMode(0644)

	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewService() service.Service {
	return &weaveService{}
}

type weaveService struct{}

func (t *weaveService) Name() string {
	return "weave"
}

func (t *weaveService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	os.MkdirAll(cniPluginDir, 0755)
	changed, err := createService(deps, flags)
	if err != nil {
		return maskAny(err)
	}
	if _, err := createCniConf(deps, flags); err != nil {
		return maskAny(err)
	}
	if err := setupCni(deps, flags); err != nil {
		return maskAny(err)
	}
	if _, err := createRktNetwork(deps, flags); err != nil {
		return maskAny(err)
	}

	if flags.Force || changed {
		if err := deps.Systemd.Enable(weaveServiceName); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(weaveServiceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", weaveServicePath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	var peers []string
	for _, m := range members {
		peers = append(peers, m.PrivateHostIP)
	}
	name, err := getPeerName(deps, flags)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		Seed     string
		Peers    string
		Name     string
		Hostname string
		IPRange  string
		IPInit   string
	}{
		Seed:     flags.Weave.Seed,
		Peers:    strings.Join(peers, " "),
		Name:     name,
		Hostname: flags.Weave.Hostname,
		IPRange:  flags.Weave.IPRange,
		IPInit:   "seed=${SEED}",
	}
	changed, err := templates.Render(weaveServiceTmpl, weaveServicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func createCniConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", cniConfPath)
	changed, err := templates.Render(cniConfTmpl, cniConfPath, nil, configFileMode)
	return changed, maskAny(err)
}

func setupCni(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	deps.Logger.Info("running weave setup-cni")
	cmd := exec.Command("weave", "setup-cni")
	if out, err := cmd.CombinedOutput(); err != nil {
		deps.Logger.Error(string(out))
		return maskAny(err)
	}
	return nil
}

func createRktNetwork(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	if err := util.EnsureDirectoryOf(rktNetworkConfPath, 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", rktNetworkConfPath)
	opts := struct {
		Subnet  string
		Gateway string
	}{
		Subnet:  flags.Weave.RktSubnet,
		Gateway: flags.Weave.RktGateway,
	}
	changed, err := templates.Render(rktNetworkConfTemplate, rktNetworkConfPath, opts, configFileMode)
	return changed, maskAny(err)
}

func getPeerName(deps service.ServiceDependencies, flags *service.ServiceFlags) (string, error) {
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return "", maskAny(err)
	}
	for _, member := range members {
		if member.ClusterIP == flags.Network.ClusterIP {
			name, err := util.WeaveNameFromMachineID(member.MachineID)
			if err != nil {
				return "", maskAny(err)
			}
			return name, nil
		}
	}

	return "", nil
}
