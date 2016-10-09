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

package gluon

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
	serviceTemplate = "templates/gluon.service.tmpl"
	serviceName     = "gluon.service"
	servicePath     = "/etc/systemd/system/" + serviceName
	gluonPath       = "/home/core/bin/gluon"

	fileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &gluonService{}
}

type gluonService struct{}

func (t *gluonService) Name() string {
	return "gluon"
}

func (t *gluonService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if err := flags.SetupDefaults(deps.Logger); err != nil {
		return maskAny(err)
	}
	if flags.Docker.DockerSubnet == "" {
		return errgo.New("docker-subnet is missing")
	}

	changedFlags, err := flags.Save()
	if err != nil {
		return maskAny(err)
	}

	changedService, err := createService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedFlags || changedService {
		if err := os.Remove(gluonPath); err != nil {
			if !os.IsNotExist(err) {
				return maskAny(err)
			}
		}

		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Enable(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		GluonImage           string
		PrivateClusterDevice string
		DockerSubnet         string
		WeaveHostname        string
	}{
		GluonImage:           flags.GluonImage,
		PrivateClusterDevice: flags.Network.PrivateClusterDevice,
		DockerSubnet:         flags.Docker.DockerSubnet,
		WeaveHostname:        flags.Weave.Hostname,
	}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, fileMode)
	return changed, maskAny(err)
}
