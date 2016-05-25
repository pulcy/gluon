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

package binaries

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
	mountTemplate = "templates/usr-bin.mount.tmpl"
	mountName     = "usr-bin.mount"
	mountPath     = "/etc/systemd/system/" + mountName
	mountPoint    = "/usr/bin"
	upperDir      = "/home/core/bin/overlay"
	workDir       = "/home/core/bin/overlay-work"

	fileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &binariesService{}
}

type binariesService struct{}

func (t *binariesService) Name() string {
	return "binaries"
}

func (t *binariesService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedMount, err := createBinariesMount(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedMount {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(mountName); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Enable(mountName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createBinariesMount(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", mountPath)
	opts := struct {
		MountPoint string
		LowerDir   string
		UpperDir   string
		WorkDir    string
	}{
		MountPoint: mountPoint,
		LowerDir:   mountPoint,
		UpperDir:   upperDir,
		WorkDir:    workDir,
	}
	changed, err := templates.Render(mountTemplate, mountPath, opts, fileMode)
	return changed, maskAny(err)
}
