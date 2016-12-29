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

package etcd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/juju/errgo"

	"strconv"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate    = "templates/99-etcd2.conf.tmpl"
	confName        = "99-etcd2.conf"
	confDir         = "/etc/systemd/system/" + serviceName + ".d/"
	confPath        = confDir + confName
	serviceName     = "etcd2.service"
	serviceTemplate = "templates/etcd.service.tmpl"
	servicePath     = "/etc/systemd/system/" + serviceName
	environmentPath = "/etc/environment"
	etcdUser        = "etcd"
	dataPath        = "/var/lib/etcd2"
	initTemplate    = "templates/etcd-init.sh.tmpl"
	initPath        = "/root/etcd-init.sh"

	configFileMode  = os.FileMode(0644)
	serviceFileMode = os.FileMode(0644)
	dataPathMode    = os.FileMode(0755)
	initFileMode    = os.FileMode(0755)
)

func NewService() service.Service {
	return &etcdService{}
}

type etcdService struct{}

func (t *etcdService) Name() string {
	return "etcd"
}

func (t *etcdService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	cfg, err := createEtcdConfig(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	_, err = createEtcdEnvironment(deps, cfg)
	if err != nil {
		return maskAny(err)
	}

	if cfg.IsProxy {
		// We do not want an etcd service, remove it
		if exists, err := deps.Systemd.Exists(serviceName); err != nil {
			return maskAny(err)
		} else if exists {
			if err := deps.Systemd.Disable(serviceName); err != nil {
				deps.Logger.Errorf("Disabling %s failed: %#v", serviceName, err)
			} else {
				os.Remove(servicePath)
				os.RemoveAll(confDir)
			}
		}
	} else {

		if err := createEtcdUserAndPath(deps); err != nil {
			return maskAny(err)
		}

		changedService, err := createService(deps, flags)
		if err != nil {
			return maskAny(err)
		}
		changedConf, err := createEtcd2Conf(deps, cfg)
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
	}

	return nil
}

// createEtcdUserAndPath ensures the ETCD user exists.
func createEtcdUserAndPath(deps service.ServiceDependencies) error {
	deps.Logger.Info("creating %s", initPath)
	if _, err := templates.Render(initTemplate, initPath, nil, initFileMode); err != nil {
		return maskAny(err)
	}
	// Call init script
	deps.Logger.Info("running %s", initPath)
	cmd := exec.Command(initPath)
	if err := cmd.Run(); err != nil {
		return maskAny(err)
	}
	return nil
}

type etcdConfig struct {
	ClusterState        string
	ClusterIP           string
	IsProxy             bool
	Name                string
	PrivateHostIP       string
	ListenPeerURLs      string // URLs for ETCD-ETCD peer communication
	AdvertisePeerURLs   string // URLs for ETCD-ETCD peer communication
	ListenClientURLs    string // Listen URLs for client-ETCD communication
	AdvertiseClientURLs string // Advertised URLs for client-ETCD communication
	Endpoints           string // URLs for client-ETCD communication
	InitialCluster      string
	Host                string // IP of 1 ETCD host
	Port                string // Port of 1 ETCD host
}

func createEtcdConfig(deps service.ServiceDependencies, flags *service.ServiceFlags) (etcdConfig, error) {
	if flags.Network.ClusterIP == "" {
		return etcdConfig{}, maskAny(fmt.Errorf("ClusterIP empty"))
	}

	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		deps.Logger.Warning("GetClusterMembers failed: %v", err)
	}

	result := etcdConfig{
		ClusterIP: flags.Network.ClusterIP,
	}
	initialCluster := []string{}
	endpoints := []string{}
	hosts := []string{}
	clientPort := 2379
	result.ClusterState = flags.Etcd.ClusterState
	if result.ClusterState == "" {
		result.ClusterState = "new"
	}
	memberIndex := 0
	for index, cm := range members {
		if !cm.EtcdProxy {
			initialCluster = append(initialCluster,
				fmt.Sprintf("%s=https://%s:2380", cm.MachineID, cm.ClusterIP),
				fmt.Sprintf("%s=https://%s:2381", cm.MachineID, cm.PrivateHostIP),
			)
			endpoints = append(endpoints, fmt.Sprintf("http://%s:%d", cm.ClusterIP, clientPort))
			hosts = append(hosts, cm.ClusterIP)
		}
		if cm.ClusterIP == flags.Network.ClusterIP {
			result.Name = cm.MachineID
			result.IsProxy = cm.EtcdProxy
			result.PrivateHostIP = cm.PrivateHostIP
			if cm.EtcdProxy {
				memberIndex = index
			} else {
				memberIndex = len(hosts) - 1
			}
		}
	}

	result.ListenPeerURLs = fmt.Sprintf("https://%s:2381,https://%s:2380", result.PrivateHostIP, flags.Network.ClusterIP)
	result.AdvertisePeerURLs = fmt.Sprintf("https://%s:2381,https://%s:2380", result.PrivateHostIP, flags.Network.ClusterIP)
	result.ListenClientURLs = fmt.Sprintf("http://%s:%d,http://%s:4001,http://127.0.0.1:%d,http://127.0.0.1:4001", flags.Network.ClusterIP, clientPort, flags.Network.ClusterIP, clientPort)
	result.AdvertiseClientURLs = fmt.Sprintf("http://%s:%d,http://%s:4001", flags.Network.ClusterIP, clientPort, flags.Network.ClusterIP)

	result.InitialCluster = strings.Join(initialCluster, ",")
	result.Endpoints = strings.Join(endpoints, ",")
	result.Host = hosts[memberIndex%len(hosts)]
	result.Port = strconv.Itoa(clientPort)

	return result, nil

}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	changed, err := templates.Render(serviceTemplate, servicePath, nil, serviceFileMode)
	return changed, maskAny(err)
}

func createEtcd2Conf(deps service.ServiceDependencies, cfg etcdConfig) (bool, error) {
	if cfg.ClusterIP == "" {
		return false, maskAny(fmt.Errorf("ClusterIP empty"))
	}
	deps.Logger.Info("creating %s", confPath)

	lines := []string{
		"[Service]",
		"Environment=ETCD_PEER_AUTO_TLS=true",
		"Environment=ETCD_LISTEN_PEER_URLS=" + cfg.ListenPeerURLs,
		"Environment=ETCD_LISTEN_CLIENT_URLS=" + cfg.ListenClientURLs,
		"Environment=ETCD_INITIAL_CLUSTER=" + cfg.InitialCluster,
		"Environment=ETCD_INITIAL_CLUSTER_STATE=" + cfg.ClusterState,
		"Environment=ETCD_INITIAL_ADVERTISE_PEER_URLS=" + cfg.AdvertisePeerURLs,
		"Environment=ETCD_ADVERTISE_CLIENT_URLS=" + cfg.AdvertiseClientURLs,
	}
	if cfg.Name != "" {
		lines = append(lines, "Environment=ETCD_NAME="+cfg.Name)
	}
	if cfg.IsProxy {
		lines = append(lines, "Environment=ETCD_PROXY=on")
	}

	changed, err := util.UpdateFile(confPath, []byte(strings.Join(lines, "\n")), configFileMode)
	return changed, maskAny(err)
}

func createEtcdEnvironment(deps service.ServiceDependencies, cfg etcdConfig) (bool, error) {
	if cfg.ClusterIP == "" {
		return false, maskAny(fmt.Errorf("ClusterIP empty"))
	}
	deps.Logger.Info("creating %s", environmentPath)

	kv := []util.KeyValuePair{
		util.KeyValuePair{Key: "ETCD_ENDPOINTS", Value: cfg.Endpoints},
		util.KeyValuePair{Key: "ETCDCTL_ENDPOINTS", Value: cfg.Endpoints},
		util.KeyValuePair{Key: "ETCD_HOST", Value: cfg.Host},
		util.KeyValuePair{Key: "ETCD_PORT", Value: cfg.Port},
	}

	changed, err := util.AppendEnvironmentFile(environmentPath, kv, configFileMode)
	return changed, maskAny(err)
}
