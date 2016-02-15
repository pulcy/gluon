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
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate = "templates/99-fleet.conf.tmpl"
	confName     = "99-fleet.conf"
	confPath     = "/etc/systemd/system/fleet.service.d/" + confName
	serviceName  = "fleet.service"

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

	return nil
}

func createFleetConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	lines := []string{
		"[Service]",
		fmt.Sprintf("Environment=FLEET_METADATA=%s", flags.FleetMetadata),
		fmt.Sprintf("Environment=FLEET_PUBLIC_IP=%s", flags.PrivateIP),
	}

	changed, err := util.UpdateFile(confPath, []byte(strings.Join(lines, "\n")), fileMode)
	return changed, maskAny(err)
}
