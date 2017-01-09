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
	"path"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	kubeControllerManagerTemplate = "templates/kubernetes/kube-controller-manager.yaml.tmpl"
)

// createKubeControllerManagerManifest creates the manifest containing the kubernetes Kube-controller-manager pod.
func createKubeControllerManagerManifest(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	if err := util.EnsureDirectoryOf(c.ManifestPath(), 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", c.ManifestPath())
	var apiServers []string
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	for _, m := range members {
		if !m.EtcdProxy {
			apiServers = append(apiServers, fmt.Sprintf("https://%s:%d", m.ClusterIP, flags.Kubernetes.APIServerPort))
		}
	}
	opts := struct {
		Image                 string
		Master                string
		ServiceClusterIPRange string
		KeyPath               string
		CAPath                string
		CertificatesFolder    string
	}{
		Image:                 flags.Kubernetes.APIServerImage,
		Master:                apiServers[0],
		ServiceClusterIPRange: flags.Kubernetes.ServiceClusterIPRange,
		KeyPath:               c.KeyPath(),
		CAPath:                c.CAPath(),
		CertificatesFolder:    path.Dir(c.CertificatePath()),
	}
	changed, err := templates.Render(kubeControllerManagerTemplate, c.ManifestPath(), opts, manifestFileMode)
	return changed, maskAny(err)
}
