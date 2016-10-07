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

package fleet

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate      = "templates/99-fleet.conf.tmpl"
	confName          = "99-fleet.conf"
	confPath          = "/etc/systemd/system/fleet.service.d/" + confName
	serviceName       = "fleet.service"
	checkScriptSource = "templates/fleet-check.sh"
	checkScriptPath   = "/home/core/bin/fleet-check.sh"

	fileMode = os.FileMode(0755)
)

func NewService() service.Service {
	return &fleetService{}
}

type fleetService struct{}

func (t *fleetService) Name() string {
	return "fleet"
}

func (t *fleetService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConf, err := createFleetConf(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	isActive, err := deps.Systemd.IsActive(serviceName)
	if err != nil {
		return maskAny(err)
	}

	if !isActive || changedConf || flags.Force {
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	if _, err := createFleetCheck(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createFleetConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	proxy, err := isEtcdProxy(deps, flags)
	if err != nil {
		return false, maskAny(err)
	}
	lines := []string{
		"[Service]",
		fmt.Sprintf("Environment=FLEET_METADATA=%s", flags.Fleet.Metadata),
		fmt.Sprintf("Environment=FLEET_PUBLIC_IP=%s", flags.Network.ClusterIP),
		fmt.Sprintf("Environment=FLEET_AGENT_TTL=%v", flags.Fleet.AgentTTL),
		fmt.Sprintf("Environment=FLEET_DISABLE_ENGINE=%v", proxy || flags.Fleet.DisableEngine),
		fmt.Sprintf("Environment=FLEET_DISABLE_WATCHES=%v", flags.Fleet.DisableWatches),
		fmt.Sprintf("Environment=FLEET_ENGINE_RECONCILE_INTERVAL=%d", flags.Fleet.EngineReconcileInterval),
		fmt.Sprintf("Environment=FLEET_TOKEN_LIMIT=%d", flags.Fleet.TokenLimit),
	}

	changed, err := util.UpdateFile(confPath, []byte(strings.Join(lines, "\n")), fileMode)
	return changed, maskAny(err)
}

// create/update /home/core/bin/fleet-check.sh
func createFleetCheck(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", checkScriptPath)
	asset, err := templates.Asset(checkScriptSource)
	if err != nil {
		return false, maskAny(err)
	}

	changed, err := util.UpdateFile(checkScriptPath, asset, 0755)
	return changed, maskAny(err)
}

func isEtcdProxy(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	for _, member := range members {
		if member.ClusterIP == flags.Network.ClusterIP {
			return member.EtcdProxy, nil
		}
	}

	return false, nil
}
