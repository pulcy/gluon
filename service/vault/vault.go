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

package vault

import (
	"os"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

var (
	vaultServiceName = "consul.service"
	vaultServiceTmpl = "templates/" + vaultServiceName + ".tmpl"
	vaultServicePath = "/etc/systemd/system/" + vaultServiceName

	serviceFileMode = os.FileMode(0644)

	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewService() service.Service {
	return &vaultService{}
}

type vaultService struct{}

func (t *vaultService) Name() string {
	return "vault"
}

func (t *vaultService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if flags.HasRole("vault") {
		// Setup a vault server
		changed, err := createService(deps, flags)
		if err != nil {
			return maskAny(err)
		}

		if flags.Force || changed {
			if err := deps.Systemd.Enable(vaultServiceName); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Reload(); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Restart(vaultServiceName); err != nil {
				return maskAny(err)
			}
		}
	} else {
		// Do not setup a vault server, remove if found
		if exists, err := deps.Systemd.Exists(vaultServiceName); err != nil {
			return maskAny(err)
		} else if exists {
			if err := deps.Systemd.Disable(vaultServiceName); err != nil {
				deps.Logger.Errorf("Disabling %s failed: %#v", vaultServiceName, err)
			} else {
				os.Remove(vaultServicePath)
			}
		}
	}

	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", vaultServicePath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	privateIP := ""
	for _, m := range members {
		if m.ClusterIP == flags.Network.ClusterIP {
			privateIP = m.PrivateHostIP
		}
	}
	opts := struct {
		PublicIP  string
		ClusterIP string
		PrivateIP string
	}{
		PublicIP:  "${COREOS_PUBLIC_IPV4}",
		ClusterIP: flags.Network.ClusterIP,
		PrivateIP: privateIP,
	}
	changed, err := templates.Render(vaultServiceTmpl, vaultServicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}
