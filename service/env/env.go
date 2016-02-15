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

package env

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
	bashrcTemplate = "templates/bashrc.tmpl"
	bashrcPath     = "/home/core/.bashrc"

	fileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &envService{}
}

type envService struct{}

func (t *envService) Name() string {
	return "env"
}

func (t *envService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if err := createBashrc(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createBashrc(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	deps.Logger.Info("creating %s", bashrcPath)
	os.Remove(bashrcPath)
	if _, err := templates.Render(bashrcTemplate, bashrcPath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
