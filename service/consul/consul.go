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

package consul

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

var (
	consulServiceName = "consul.service"
	consulServiceTmpl = "templates/" + consulServiceName + ".tmpl"
	consulServicePath = "/etc/systemd/system/" + consulServiceName

	serviceFileMode = os.FileMode(0644)

	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewService() service.Service {
	return &consulService{}
}

type consulService struct{}

func (t *consulService) Name() string {
	return "consul"
}

func (t *consulService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changed, err := createService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changed {
		if err := deps.Systemd.Enable(consulServiceName); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(consulServiceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", consulServicePath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	var join []string
	bootstrapExpect := 0
	for _, m := range members {
		if !m.EtcdProxy {
			bootstrapExpect++
			if m.ClusterIP != flags.Network.ClusterIP {
				join = append(join, fmt.Sprintf("-retry-join %s", m.ClusterIP))
			}
		}
	}
	srv, err := isServer(members, flags)
	if err != nil {
		return false, maskAny(err)
	}
	args := fmt.Sprintf("-advertise=%s %s -client=0.0.0.0 -data-dir=/opt/consul/data", flags.Network.ClusterIP, strings.Join(join, " "))
	if srv {
		args = fmt.Sprintf("-server -bootstrap-expect=%d %s", bootstrapExpect, args)
	}
	opts := struct {
		Flags string
	}{
		Flags: args,
	}
	changed, err := templates.Render(consulServiceTmpl, consulServicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func isServer(members []service.ClusterMember, flags *service.ServiceFlags) (bool, error) {
	for _, member := range members {
		if member.ClusterIP == flags.Network.ClusterIP {
			return !member.EtcdProxy, nil
		}
	}

	return false, nil
}
