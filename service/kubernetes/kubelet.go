// Copyright (c) 2017 Pulcy.
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

package kubernetes

import (
	"fmt"
	"strings"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
)

const (
	kubeletServiceTemplate = "templates/kubernetes/kubelet.service.tmpl"
)

// createKubeletService creates the file containing the kubernetes Kubelet service.
func createKubeletService(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	configChanged, err := createKubeConfig(deps, flags, c)
	if err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", c.ServicePath())
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	var apiServers []string
	for _, m := range members {
		if !m.EtcdProxy {
			apiServers = append(apiServers, fmt.Sprintf("https://%s:%d", m.ClusterIP, flags.Kubernetes.APIServerPort))
		}
	}
	opts := struct {
		APIServers          string
		ClusterDNS          string
		KubeConfigPath      string
		RegisterSchedulable bool
		NodeIP              string
		CertPath            string
		KeyPath             string
	}{
		APIServers:          strings.Join(apiServers, ","),
		ClusterDNS:          flags.Kubernetes.ClusterDNS,
		KubeConfigPath:      c.KubeConfigPath(),
		RegisterSchedulable: !flags.HasRole("core"),
		NodeIP:              flags.Network.ClusterIP,
		CertPath:            c.CertificatePath(),
		KeyPath:             c.KeyPath(),
	}
	changed, err := templates.Render(kubeletServiceTemplate, c.ServicePath(), opts, serviceFileMode)
	return changed || configChanged, maskAny(err)
}
