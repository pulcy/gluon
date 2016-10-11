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

package proxy

import (
	"io/ioutil"
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
	environment = "/etc/environment"
	dockerEnv   = "/etc/pulcy/docker.env"

	fileMode      = os.FileMode(0644)
	noProxyPrefix = "NO_PROXY="
)

func NewService() service.Service {
	return &proxyService{}
}

type proxyService struct{}

func (t *proxyService) Name() string {
	return "proxy"
}

func (t *proxyService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	noProxy, err := createNoProxy(deps, flags)
	if err != nil {
		return maskAny(err)
	}
	if err := updateNoProxy(environment, noProxy, deps, flags); err != nil {
		return maskAny(err)
	}
	if err := updateNoProxy(dockerEnv, noProxy, deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

// createNoProxy creates the value for the no_proxy environment variable.
func createNoProxy(deps service.ServiceDependencies, flags *service.ServiceFlags) (string, error) {
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return "", maskAny(err)
	}
	list := []string{".local", ".private"}
	for _, cm := range members {
		list = append(list, cm.ClusterIP)
		if cm.ClusterIP != cm.PrivateHostIP {
			list = append(list, cm.PrivateHostIP)
		}
	}
	return strings.Join(list, ","), nil
}

func updateNoProxy(path, noProxy string, deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	deps.Logger.Info("updating %s", path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	origContent, err := ioutil.ReadFile(path)
	if err != nil {
		return maskAny(err)
	}
	lines := strings.Split(string(origContent), "\n")
	updatedLines := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), noProxyPrefix) && line != "" {
			updatedLines = append(updatedLines, line)
		}
	}
	updatedLines = append(updatedLines, noProxyPrefix+noProxy)

	newContent := strings.Join(updatedLines, "\n")
	if _, err := util.UpdateFile(path, []byte(newContent), fileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
