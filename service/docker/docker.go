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

package docker

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	serviceTemplate = "templates/docker.service.tmpl"
	ServiceName     = "docker.service"
	servicePath     = "/etc/systemd/system/" + ServiceName
	rootConfigPath1 = "/root/.docker/config"
	rootConfigPath2 = "/root/.docker/config.json"
	cleanupSource   = "templates/docker-cleanup.sh"
	cleanupPath     = "/home/core/bin/docker-cleanup.sh"

	fileMode = os.FileMode(0755)
)

// ConfigFile ~/.docker/config.json file info
// Taken from https://github.com/docker/docker/blob/master/cliconfig/config.go
type ConfigFile struct {
	AuthConfigs map[string]AuthConfig `json:"auths"`
}

// AuthConfig contains authorization information for connecting to a Registry
// Taken from https://github.com/docker/engine-api/blob/master/types/auth.go
type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth"`
	Email         string `json:"email"`
	ServerAddress string `json:"serveraddress,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

func NewService() service.Service {
	return &dockerService{}
}

type dockerService struct{}

func (t *dockerService) Name() string {
	return "docker"
}

func (t *dockerService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConfig1, err := createDockerConfig(rootConfigPath1, deps, flags)
	if err != nil {
		return maskAny(err)
	}
	changedConfig2, err := createDockerConfig(rootConfigPath2, deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedService, err := createDockerService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedConfig1 || changedConfig2 || changedService {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(ServiceName); err != nil {
			return maskAny(err)
		}
	}

	if _, err := createDockerCleanup(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createDockerService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		DockerIP string
	}{
		DockerIP: flags.Docker.DockerIP,
	}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, fileMode)
	return changed, maskAny(err)
}

func createDockerConfig(rootConfigPath string, deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	if flags.Docker.PrivateRegistryPassword != "" && flags.Docker.PrivateRegistryUrl != "" && flags.Docker.PrivateRegistryUserName != "" {
		deps.Logger.Info("creating %s", rootConfigPath)
		// Load config file
		cf := ConfigFile{
			AuthConfigs: make(map[string]AuthConfig),
		}

		// Set authentication entries
		cf.AuthConfigs[flags.Docker.PrivateRegistryUrl] = AuthConfig{
			Auth: encodeAuth(AuthConfig{
				Username: flags.Docker.PrivateRegistryUserName,
				Password: flags.Docker.PrivateRegistryPassword,
				Email:    "",
			}),
		}

		// Save
		os.MkdirAll(filepath.Dir(rootConfigPath), 0700)
		raw, err := json.MarshalIndent(cf, "", "\t")
		if err != nil {
			return false, maskAny(err)
		}
		changed, err := util.UpdateFile(rootConfigPath, raw, 0600)
		return changed, maskAny(err)
	} else {
		deps.Logger.Warning("Skip creating .docker config")
	}

	return false, nil
}

// encodeAuth creates a base64 encoded string to containing authorization information
// Taken from https://github.com/docker/docker/blob/master/cliconfig/config.go
func encodeAuth(authConfig AuthConfig) string {
	authStr := authConfig.Username + ":" + authConfig.Password
	msg := []byte(authStr)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(msg)))
	base64.StdEncoding.Encode(encoded, msg)
	return string(encoded)
}

// create/update /home/core/bin/docker-cleanup.sh
func createDockerCleanup(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", cleanupPath)
	asset, err := templates.Asset(cleanupSource)
	if err != nil {
		return false, maskAny(err)
	}

	changed, err := util.UpdateFile(cleanupPath, asset, 0755)
	return changed, maskAny(err)
}
