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

package nomad

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

var (
	nomadServiceName = "nomad.service"
	nomadServiceTmpl = "templates/" + nomadServiceName + ".tmpl"
	nomadServicePath = "/etc/systemd/system/" + nomadServiceName
	nomadConfigTmpl  = "templates/nomad.conf.tmpl"
	nomadConfigPath  = "/etc/nomad/agent.conf"
	vaultEnvPath     = "/etc/pulcy/vault.env"

	configFileMode  = os.FileMode(0644)
	serviceFileMode = os.FileMode(0644)

	maskAny = errgo.MaskFunc(errgo.Any)
)

func NewService() service.Service {
	return &nomadService{}
}

type nomadService struct{}

func (t *nomadService) Name() string {
	return "nomad"
}

func (t *nomadService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConfig, err := createConfig(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedService, err := createService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedConfig || changedService {
		if err := deps.Systemd.Enable(nomadServiceName); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(nomadServiceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", nomadServicePath)
	args := fmt.Sprintf("-bind=0.0.0.0 -data-dir=/opt/nomad/data -config=%s", nomadConfigPath)
	opts := struct {
		Flags string
	}{
		Flags: args,
	}
	changed, err := templates.Render(nomadServiceTmpl, nomadServicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func createConfig(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", nomadConfigPath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	srv, err := isServer(members, flags)
	if err != nil {
		return false, maskAny(err)
	}
	vaultEnv, err := readVaultEnv()
	if err != nil {
		return false, maskAny(err)
	}
	vaultAddress := vaultEnv["VAULT_ADDR"]
	vaultCAFile := vaultEnv["VAULT_CACERT"]
	bootstrapExpect := 0
	for _, m := range members {
		if !m.EtcdProxy {
			bootstrapExpect++
		}
	}
	opts := struct {
		AdvertiseIP     string
		ConsulAddress   string
		MetaLines       []string
		IsServer        bool
		BootstrapExpect int
		HasVault        bool
		VaultAddress    string
		VaultCAFile     string
	}{
		AdvertiseIP:     flags.Network.ClusterIP,
		ConsulAddress:   fmt.Sprintf("%s:8500", flags.Network.ClusterIP),
		IsServer:        srv,
		BootstrapExpect: bootstrapExpect,
		HasVault:        vaultAddress != "" && vaultCAFile != "",
		VaultAddress:    vaultAddress,
		VaultCAFile:     vaultCAFile,
	}
	metaParts := strings.Split(flags.Fleet.Metadata, ",")
	for _, meta := range metaParts {
		kv := strings.SplitN(meta, "=", 2)
		if len(kv) == 2 {
			opts.MetaLines = append(opts.MetaLines, fmt.Sprintf("\"%s\" = \"%s\"", kv[0], kv[1]))
		}
	}

	changed, err := templates.Render(nomadConfigTmpl, nomadConfigPath, opts, configFileMode)
	return changed, maskAny(err)
}

func isServer(members []service.ClusterMember, flags *service.ServiceFlags) (bool, error) {
	for _, member := range members {
		if member.ClusterIP == flags.Network.ClusterIP {
			return !member.EtcdProxy, nil
		}
	}

	return false, nil
}

func readVaultEnv() (map[string]string, error) {
	result := make(map[string]string)
	raw, err := ioutil.ReadFile(vaultEnvPath)
	if os.IsNotExist(err) {
		return result, nil
	} else if err != nil {
		return nil, maskAny(err)
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result, nil
}
