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

package sshd

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
	confTemplate = "templates/sshd/sshd_config.tmpl"
	confName     = "sshd_config"
	confPath     = "/etc/ssh/" + confName
	ServiceName  = "ssh.service"

	fileMode = os.FileMode(0600)
)

func NewService() service.Service {
	return &sshdService{}
}

type sshdService struct{}

func (t *sshdService) Name() string {
	return "sshd"
}

func (t *sshdService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConfig, err := createSshdConfig(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedConfig {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createSshdConfig(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", confPath)
	os.Remove(confPath)
	changed, err := templates.Render(deps.Logger, confTemplate, confPath, nil, fileMode)
	return changed, maskAny(err)
}
