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
	"path"
	"strings"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/service/etcd"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	kubeApiServiceTemplate = "templates/kubernetes/kube-apiserver.yaml.tmpl"
)

// createKubeApiServerManifest creates the manifest containing the kubernetes Kube-apiserver pod.
func createKubeApiServerManifest(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	configChanged, err := createKubeConfig(deps, flags, c)
	if err != nil {
		return false, maskAny(err)
	}
	if err := util.EnsureDirectoryOf(c.ManifestPath(), 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", c.ManifestPath())
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	var etcdEndpoints []string
	for _, m := range members {
		if !m.EtcdProxy {
			etcdEndpoints = append(etcdEndpoints, flags.Etcd.CreateEndpoint(m.ClusterIP))
		}
	}
	opts := struct {
		Image                 string
		EtcdEndpoints         string
		EtcdCAPath            string
		EtcdCertPath          string
		EtcdKeyPath           string
		ServiceClusterIPRange string
		SecurePort            int
		AdvertiseAddress      string
		CertPath              string
		KeyPath               string
		CAPath                string
		CertificatesFolder    string
	}{
		Image:                 flags.Kubernetes.APIServerImage,
		EtcdEndpoints:         strings.Join(etcdEndpoints, ","),
		EtcdCAPath:            etcd.CertsCAPath,
		EtcdCertPath:          etcd.CertsCertPath,
		EtcdKeyPath:           etcd.CertsKeyPath,
		ServiceClusterIPRange: flags.Kubernetes.ServiceClusterIPRange,
		SecurePort:            flags.Kubernetes.APIServerPort,
		AdvertiseAddress:      flags.Network.ClusterIP,
		CertPath:              c.CertificatePath(),
		KeyPath:               c.KeyPath(),
		CAPath:                c.CAPath(),
		CertificatesFolder:    path.Dir(c.CertificatePath()),
	}
	changed, err := templates.Render(kubeApiServiceTemplate, c.ManifestPath(), opts, manifestFileMode)
	return changed || configChanged, maskAny(err)
}
