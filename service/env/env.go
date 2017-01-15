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
	"strings"

	"github.com/juju/errgo"

	"io/ioutil"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	bashrcTemplate = "templates/env/bashrc.tmpl"
	bashrcPath     = "/home/core/.bashrc"
	resolvConf     = "/etc/resolv.conf"

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
	if err := addGoogleDNS(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createBashrc(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	deps.Logger.Info("creating %s", bashrcPath)
	os.Remove(bashrcPath)
	opts := struct {
		FleetEnabled      bool
		KubernetesEnabled bool
	}{
		FleetEnabled:      flags.Fleet.IsEnabled(),
		KubernetesEnabled: flags.Kubernetes.IsEnabled(),
	}
	if _, err := templates.Render(deps.Logger, bashrcTemplate, bashrcPath, opts, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func addGoogleDNS(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	content, err := ioutil.ReadFile(resolvConf)
	if os.IsNotExist(err) {
		content = []byte("")
	} else if err != nil {
		return maskAny(err)
	}
	dnsLine := "nameserver 8.8.8.8"
	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == dnsLine {
			// Google DNS already found
			return nil
		}
	}
	// Append google dns
	lines = append(lines, dnsLine)
	content = []byte(strings.Join(lines, "\n") + "\n")
	if _, err := util.UpdateFile(deps.Logger, resolvConf, content, 0755); err != nil {
		return maskAny(err)
	}
	return nil
}
