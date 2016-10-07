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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	tmpFilesConfPath   = "/usr/lib/tmpfiles.d/rkt.conf"
	tmpFilesConfSource = "templates/rkt.conf"

	apiServiceName      = "rkt-api.service"
	gcServiceName       = "rkt-gc.service"
	gcTimerName         = "rkt-gc.timer"
	metadataServiceName = "rkt-metadata.service"
	metadataSocketName  = "rkt-metadata.socket"

	privateRegistryAuthConfPath = "/etc/rkt/auth.d/gluon-private-registry.json"

	serviceFileMode = os.FileMode(0644)

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
	if _, err := createPrivateRegistryAuthConf(deps, flags); err != nil {
		return maskAny(err)
	}

	for _, serviceName := range []string{
		apiServiceName,
		gcServiceName,
		gcTimerName,
		metadataSocketName, // Keep this before metadataServiceName
		metadataServiceName,
	} {
		changed, err := createService(serviceName, deps, flags)
		if err != nil {
			return maskAny(err)
		}

		if flags.Force || changed {
			if err := deps.Systemd.Enable(serviceName); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Reload(); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Restart(serviceName); err != nil {
				return maskAny(err)
			}
		}
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

func createService(serviceName string, deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	serviceTemplate := serviceTemplate(serviceName)
	servicePath := servicePath(serviceName)
	deps.Logger.Info("creating %s", servicePath)
	opts := struct{}{}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func serviceTemplate(serviceName string) string {
	return fmt.Sprintf("templates/%s.tmpl", serviceName)
}

func servicePath(serviceName string) string {
	return "/etc/systemd/system/" + serviceName
}

func createPrivateRegistryAuthConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	if flags.Docker.PrivateRegistryPassword != "" && flags.Docker.PrivateRegistryUrl != "" && flags.Docker.PrivateRegistryUserName != "" {
		deps.Logger.Info("creating %s", privateRegistryAuthConfPath)
		// Create config
		/*	{
			"rktKind": "auth",
			"rktVersion": "v1",
			"domains": ["coreos.com", "tectonic.com"],
			"type": "basic",
			"credentials": {
				"user": "foo",
				"password": "bar"
			}
		}*/
		cf := struct {
			Kind        string   `json:"rktKind"`
			Version     string   `json:"rktVersion"`
			Registries  []string `json:"registries"`
			Type        string   `json:"type"`
			Credentials struct {
				User     string `json:"user"`
				Password string `json:"password"`
			} `json:"credentials"`
		}{
			Kind:       "dockerAuth",
			Version:    "v1",
			Registries: []string{flags.Docker.PrivateRegistryUrl},
			Type:       "basic",
		}
		cf.Credentials.User = flags.Docker.PrivateRegistryUserName
		cf.Credentials.Password = flags.Docker.PrivateRegistryPassword

		// Save
		os.MkdirAll(filepath.Dir(privateRegistryAuthConfPath), 0700)
		raw, err := json.MarshalIndent(cf, "", "\t")
		if err != nil {
			return false, maskAny(err)
		}
		changed, err := util.UpdateFile(privateRegistryAuthConfPath, raw, 0600)
		return changed, maskAny(err)
	} else {
		deps.Logger.Warningf("Skip creating %s", privateRegistryAuthConfPath)
	}

	return false, nil
}
