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

package etcd2

import (
	"fmt"
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
	confTemplate = "templates/99-etcd2.conf.tmpl"
	confName     = "99-etcd2.conf"
	confPath     = "/etc/systemd/system/etcd2.service.d/" + confName
	serviceName  = "etcd2.service"

	fileMode = os.FileMode(0755)
)

func NewService() service.Service {
	return &etcd2Service{}
}

type etcd2Service struct{}

func (t *etcd2Service) Name() string {
	return "etcd2"
}

func (t *etcd2Service) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	changedConf, err := createEtcd2Conf(deps, flags)
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

	if !isActive || changedConf || flags.Force {
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createEtcd2Conf(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	if flags.PrivateIP == "" {
		return false, maskAny(fmt.Errorf("PrivateIP empty"))
	}
	deps.Logger.Info("creating %s", confPath)

	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		deps.Logger.Warning("GetClusterMembers failed: %v", err)
	}
	clusterItems := []string{}
	name := ""
	clusterState := flags.EtcdClusterState
	if clusterState == "" {
		clusterState = "new"
	}
	etcdProxy := false
	for _, cm := range members {
		if !cm.EtcdProxy {
			clusterItems = append(clusterItems,
				fmt.Sprintf("%s=http://%s:2380", cm.MachineID, cm.PrivateIP))
		}
		if cm.PrivateIP == flags.PrivateIP {
			name = cm.MachineID
			etcdProxy = cm.EtcdProxy
		}
	}

	lines := []string{
		"[Service]",
		"Environment=ETCD_LISTEN_PEER_URLS=" + fmt.Sprintf("http://%s:2380", flags.PrivateIP),
		"Environment=ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379,http://0.0.0.0:4001",
		"Environment=ETCD_INITIAL_CLUSTER=" + strings.Join(clusterItems, ","),
		"Environment=ETCD_INITIAL_CLUSTER_STATE=" + clusterState,
		"Environment=ETCD_INITIAL_ADVERTISE_PEER_URLS=" + fmt.Sprintf("http://%s:2380", flags.PrivateIP),
		"Environment=ETCD_ADVERTISE_CLIENT_URLS=" + fmt.Sprintf("http://%s:2379,http://%s:4001", flags.PrivateIP, flags.PrivateIP),
	}
	if name != "" {
		lines = append(lines, "Environment=ETCD_NAME="+name)
	}
	if etcdProxy {
		lines = append(lines, "Environment=ETCD_PROXY=on")
	}

	changed, err := util.UpdateFile(confPath, []byte(strings.Join(lines, "\n")), fileMode)
	return changed, maskAny(err)
}
