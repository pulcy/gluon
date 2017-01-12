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

package fleet

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/service/etcd"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confName          = "99-fleet.conf"
	confDir           = "/etc/systemd/system/" + serviceName + ".d/"
	confPath          = confDir + confName
	serviceName       = "fleet.service"
	serviceTemplate   = "templates/fleet/fleet.service.tmpl"
	servicePath       = "/etc/systemd/system/" + serviceName
	socketName        = "fleet.socket"
	socketPath        = "/etc/systemd/system/fleet.socket"
	socketTemplate    = "templates/fleet/fleet.socket.tmpl"
	checkScriptSource = "templates/fleet/fleet-check.sh"
	checkScriptPath   = "/home/core/bin/fleet-check.sh"

	serviceFileMode = os.FileMode(0644)
	configFileMode  = os.FileMode(0644)
	scriptFileMode  = os.FileMode(0755)
)

func NewService() service.Service {
	return &fleetService{}
}

type fleetService struct{}

func (t *fleetService) Name() string {
	return "fleet"
}

func (t *fleetService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	if flags.Fleet.IsEnabled() {
		changedConf, err := createFleetConf(deps, flags)
		if err != nil {
			return maskAny(err)
		}
		changedService, err := createService(deps, flags)
		if err != nil {
			return maskAny(err)
		}
		changedSocket, err := createSocket(deps, flags)
		if err != nil {
			return maskAny(err)
		}

		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}

		isActive, err := deps.Systemd.IsActive(serviceName)
		if err != nil {
			return maskAny(err)
		}

		if !isActive || changedService || changedConf || flags.Force {
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
		if changedSocket || flags.Force {
			if err := deps.Systemd.Restart(socketName); err != nil {
				return maskAny(err)
			}
		}

		if _, err := createFleetCheck(deps, flags); err != nil {
			return maskAny(err)
		}
	} else {
		// Remove fleet
		if err := deps.Systemd.StopAndRemove(socketName, socketPath); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.StopAndRemove(serviceName, servicePath, confPath, confDir, checkScriptPath); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	proxy, err := isEtcdProxy(deps, flags)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		HaveEtcd bool
	}{
		HaveEtcd: !proxy,
	}
	changed, err := templates.Render(deps.Logger, serviceTemplate, servicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func createFleetConf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	proxy, err := isEtcdProxy(deps, flags)
	if err != nil {
		return false, maskAny(err)
	}

	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		deps.Logger.Warning("GetClusterMembers failed: %v", err)
	}
	etcdServers := []string{}
	for _, cm := range members {
		if !cm.EtcdProxy {
			etcdServers = append(etcdServers, flags.Etcd.CreateEndpoint(cm.ClusterIP))
		}
	}

	lines := []string{
		"[Service]",
		fmt.Sprintf("Environment=FLEET_METADATA=%s", flags.Fleet.Metadata),
		fmt.Sprintf("Environment=FLEET_ETCD_SERVERS=%s", strings.Join(etcdServers, ",")),
		fmt.Sprintf("Environment=FLEET_PUBLIC_IP=%s", flags.Network.ClusterIP),
		fmt.Sprintf("Environment=FLEET_AGENT_TTL=%v", flags.Fleet.AgentTTL),
		fmt.Sprintf("Environment=FLEET_DISABLE_ENGINE=%v", proxy || flags.Fleet.DisableEngine),
		fmt.Sprintf("Environment=FLEET_DISABLE_WATCHES=%v", flags.Fleet.DisableWatches),
		fmt.Sprintf("Environment=FLEET_ENGINE_RECONCILE_INTERVAL=%d", flags.Fleet.EngineReconcileInterval),
		fmt.Sprintf("Environment=FLEET_TOKEN_LIMIT=%d", flags.Fleet.TokenLimit),
	}
	if flags.Etcd.SecureClients {
		lines = append(lines,
			fmt.Sprintf("Environment=FLEET_ETCD_CERTFILE=%s", etcd.CertsCertPath),
			fmt.Sprintf("Environment=FLEET_ETCD_KEYFILE=%s", etcd.CertsKeyPath),
			fmt.Sprintf("Environment=FLEET_ETCD_CAFILE=%s", etcd.CertsCAPath),
		)
	}

	changed, err := util.UpdateFile(deps.Logger, confPath, []byte(strings.Join(lines, "\n")), configFileMode)
	return changed, maskAny(err)
}

// create/update /home/core/bin/fleet-check.sh
func createFleetCheck(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", checkScriptPath)
	asset, err := templates.Asset(checkScriptSource)
	if err != nil {
		return false, maskAny(err)
	}

	changed, err := util.UpdateFile(deps.Logger, checkScriptPath, asset, scriptFileMode)
	return changed, maskAny(err)
}

func createSocket(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", socketPath)
	opts := struct {
		ClusterIP string
	}{
		ClusterIP: flags.Network.ClusterIP,
	}
	changed, err := templates.Render(deps.Logger, socketTemplate, socketPath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func isEtcdProxy(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	for _, member := range members {
		if member.ClusterIP == flags.Network.ClusterIP {
			return member.EtcdProxy, nil
		}
	}

	return false, nil
}
