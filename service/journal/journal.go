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

package journal

import (
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
	socketPath = "/lib/systemd/system/" + socketName
	socketName = "systemd-journal-gatewayd.socket"

	configFileMode = os.FileMode(0644)
)

func NewService() service.Service {
	return &journalService{}
}

type journalService struct{}

func (t *journalService) Name() string {
	return "journal"
}

func (t *journalService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConf, err := createJournalConf(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	isActive, err := deps.Systemd.IsActive(socketName)
	if err != nil {
		return maskAny(err)
	}

	if !isActive || changedConf || flags.Force {
		if err := deps.Systemd.Restart(socketName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createJournalConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	lines := []string{
		"[Unit]",
		"Description=Journal Gateway Service Socket",
		"Documentation=man:systemd-journal-gatewayd(8)",
		"",
		"[Socket]",
		"ListenStream=[::1]:19531",
		"",
		"[Install]",
		"WantedBy=sockets.target",
	}

	changed, err := util.UpdateFile(socketPath, []byte(strings.Join(lines, "\n")), configFileMode)
	return changed, maskAny(err)
}
