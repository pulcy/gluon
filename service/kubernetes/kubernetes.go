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
	"os"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/service"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)

	components = map[Component]componentSetup{
		// Components that should be installed on all nodes
		NewServiceComponent("kubelet", false):    createKubeletService,
		NewServiceComponent("kube-proxy", false): createKubeProxyService,
		// Components that should be installed on master nodes only
		NewManifestComponent("kube-apiserver", true):          createKubeApiServerManifest,
		NewManifestComponent("kube-controller-manager", true): createKubeControllerManagerManifest,
	}
)

const (
	configFileMode   = os.FileMode(0644)
	manifestFileMode = os.FileMode(0644)
	serviceFileMode  = os.FileMode(0644)
	templateFileMode = os.FileMode(0400)
)

type componentSetup func(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error)

func NewService() service.Service {
	return &k8sService{}
}

type k8sService struct{}

func (t *k8sService) Name() string {
	return "kubernetes"
}

func (t *k8sService) Setup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	runKubernetes := true
	for c, setup := range components {
		installComponent := runKubernetes
		if c.MasterOnly() && !flags.HasRole("core") {
			installComponent = false
		}
		var certsTemplateChanged, certsServiceChanged bool
		if installComponent {
			// Create k8s-*-certs.service and template file
			var err error
			if certsTemplateChanged, err = createCertsTemplate(deps, flags, c); err != nil {
				return maskAny(err)
			}
			if certsServiceChanged, err = createCertsService(deps, flags, c); err != nil {
				return maskAny(err)
			}
			isActive, err := deps.Systemd.IsActive(c.CertificatesServiceName())
			if err != nil {
				return maskAny(err)
			}

			if !isActive || certsTemplateChanged || certsServiceChanged || flags.Force {
				if err := deps.Systemd.Enable(c.CertificatesServiceName()); err != nil {
					return maskAny(err)
				}
				if err := deps.Systemd.Reload(); err != nil {
					return maskAny(err)
				}
				if err := deps.Systemd.Restart(c.CertificatesServiceName()); err != nil {
					return maskAny(err)
				}
			}

			// Create component service / manifest
			serviceChanged, err := setup(deps, flags, c)
			if err != nil {
				return maskAny(err)
			}

			if !c.IsManifest() {
				isActive, err = deps.Systemd.IsActive(c.ServiceName())
				if err != nil {
					return maskAny(err)
				}

				if !isActive || serviceChanged || certsTemplateChanged || certsServiceChanged || flags.Force {
					if err := deps.Systemd.Enable(c.ServiceName()); err != nil {
						return maskAny(err)
					}
					if err := deps.Systemd.Reload(); err != nil {
						return maskAny(err)
					}
					if err := deps.Systemd.Restart(c.ServiceName()); err != nil {
						return maskAny(err)
					}
				}
			}
		} else {
			// Component service no longer needed, remove it
			if c.IsManifest() {
				os.Remove(c.ManifestPath())
			} else {
				if exists, err := deps.Systemd.Exists(c.ServiceName()); err != nil {
					return maskAny(err)
				} else if exists {
					if err := deps.Systemd.Disable(c.ServiceName()); err != nil {
						deps.Logger.Errorf("Disabling %s failed: %#v", c.ServiceName(), err)
					} else {
						os.Remove(c.ServicePath())
					}
				}
			}

			// k8s-*-certs no longer needed, remove it
			if exists, err := deps.Systemd.Exists(c.CertificatesServiceName()); err != nil {
				return maskAny(err)
			} else if exists {
				if err := deps.Systemd.Disable(c.CertificatesServiceName()); err != nil {
					deps.Logger.Errorf("Disabling %s failed: %#v", c.CertificatesServiceName(), err)
				} else {
					os.Remove(c.CertificatesServicePath())
					os.Remove(c.CertificatesTemplatePath())
				}
			}
		}
	}

	return nil
}

// getAPIServers creates a list of URL to the API servers of the cluster.
func getAPIServers(deps service.ServiceDependencies, flags *service.ServiceFlags) ([]string, error) {
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return nil, maskAny(err)
	}
	var apiServers []string
	for _, m := range members {
		if !m.EtcdProxy {
			apiServers = append(apiServers, fmt.Sprintf("https://%s:%d", m.ClusterIP, flags.Kubernetes.APIServerPort))
		}
	}
	return apiServers, nil
}

// servicePath returns the full path of the file containing the service with given name.
func servicePath(serviceName string) string {
	return "/etc/systemd/system/" + serviceName
}

// manifestPath returns the full path of the file containing the manifest with given name.
func manifestPath(manifestName string) string {
	return "/etc/kubernetes/manifests/" + manifestName
}

// certificatePath returns the full path of the file with given name.
func certificatePath(fileName string) string {
	return "/opt/certs/" + fileName
}
