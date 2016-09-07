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

package rkt

import (
	"fmt"
	"os/exec"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	tmpFilesConfPath   = "/usr/lib/tmpfiles.d/rkt.conf"
	tmpFilesConfSource = "templates/rkt.conf"

	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewService() service.Service {
	return &rktService{}
}

type rktService struct{}

func (t *rktService) Name() string {
	return "rkt"
}

func (t *rktService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if _, err := createTmpFilesConf(deps, flags); err != nil {
		return maskAny(err)
	}
	if err := setupDataDir(deps, flags); err != nil {
		return maskAny(err)
	}
	if err := addCoreToRktGroup(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createTmpFilesConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", tmpFilesConfPath)
	asset, err := templates.Asset(tmpFilesConfSource)
	if err != nil {
		return false, maskAny(err)
	}

	changed, err := util.UpdateFile(tmpFilesConfPath, asset, 0644)
	return changed, maskAny(err)

}

func setupDataDir(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	cmd := exec.Command("/usr/bin/rkt-scripts/setup-data-dir.sh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return maskAny(err)
	}
	return nil
}

func addCoreToRktGroup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	cmd := exec.Command("gpasswd", "-a", "core", "rkt")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return maskAny(err)
	}
	return nil
}
