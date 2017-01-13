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

import "fmt"

type Component struct {
	name       string
	masterOnly bool
	isManifest bool
}

// NewServiceComponent creates a new component that runs in a systemd service
func NewServiceComponent(name string, masterOnly bool) Component {
	return Component{name, masterOnly, false}
}

// NewManifestComponent creates a new component that runs in a static pod inside kubelet
func NewManifestComponent(name string, masterOnly bool) Component {
	return Component{name, masterOnly, true}
}

// String returns the name of the component
func (c Component) String() string {
	return c.name
}

// Name returns the name of the component
func (c Component) Name() string {
	return c.name
}

// MasterOnly returns true if this component should only be deployed on master nodes.
func (c Component) MasterOnly() bool {
	return c.masterOnly
}

// IsManifest returns true if this component is deployed as a static pod inside kubelet using a manifest.
func (c Component) IsManifest() bool {
	return c.isManifest
}

// ServiceName returns the name of the systemd service that runs the component.
func (c Component) ServiceName() string {
	if c.isManifest {
		return ""
	}
	return fmt.Sprintf("%s.service", c)
}

// ServicePath returns the full path of the file containing the systemd service that runs the component.
func (c Component) ServicePath() string {
	if c.isManifest {
		return ""
	}
	return servicePath(c.ServiceName())
}

// ManifestName returns the name of the static pod manifest that runs the component.
func (c Component) ManifestName() string {
	if !c.isManifest {
		return ""
	}
	return fmt.Sprintf("%s.yaml", c)
}

// ManifestPath returns the full path of the file containing the static pod manifest that runs the component.
func (c Component) ManifestPath() string {
	if !c.isManifest {
		return ""
	}
	return manifestPath(c.ManifestName())
}

// AddonPath returns the full path of the file containing the addon that runs the component.
func (c Component) AddonPath() string {
	if !c.isManifest {
		return ""
	}
	return addonPath(c.ManifestName())
}

// CertificatesServiceName returns the name of the systemd service that generates the TLS certificates for the component.
func (c Component) CertificatesServiceName() string {
	return fmt.Sprintf("%s-certs.service", c)
}

// CertificatesServicePath returns the full path of the file containing the systemd service that generates the TLS certificates for the component.
func (c Component) CertificatesServicePath() string {
	return servicePath(c.CertificatesServiceName())
}

// CertificatesTemplateName returns the name of the consul-template template that generates the TLS certificates for the component.
func (c Component) CertificatesTemplateName() string {
	return fmt.Sprintf("%s-certs.template", c)
}

// CertificatesTemplatePath returns the full path of the consul-template template that generates the TLS certificates for the component.
func (c Component) CertificatesTemplatePath() string {
	return certificatePath(c.CertificatesTemplateName())
}

// CertificatesTemplateOutputPath returns the full path of the file created by the consul-template template that generates the TLS certificates for the component.
func (c Component) CertificatesTemplateOutputPath() string {
	return certificatePath(fmt.Sprintf("%s.serial", c))
}

// CertificatesConfigName returns the name config file used by consul-template for the component.
func (c Component) CertificatesConfigName() string {
	return fmt.Sprintf("%s-certs-config.json", c)
}

// CertificatePath returns the full path of the public key part of the certificate for this component.
func (c Component) CertificatePath() string {
	return fmt.Sprintf("/opt/certs/%s-cert.pem", c)
}

// KeyPath returns the full path of the private key part of the certificate for this component.
func (c Component) KeyPath() string {
	return fmt.Sprintf("/opt/certs/%s-key.pem", c)
}

// CAPath returns the full path of the CA certificate for this component.
func (c Component) CAPath() string {
	return fmt.Sprintf("/opt/certs/%s-ca.pem", c)
}

// KubeConfigPath returns the full path of the kubeconfig configuration file for this component.
func (c Component) KubeConfigPath() string {
	return fmt.Sprintf("/var/lib/%s/kubeconfig", c)
}

// JobID returns the ID of the vault-monkey job used to access certificates for this component.
func (c Component) JobID(clusterID string) string {
	return jobID(clusterID, c.Name())
}

// RestartCommand returns a full command that restarts the component
func (c Component) RestartCommand() string {
	if c.isManifest {
		return fmt.Sprintf("/usr/bin/touch %s", c.ManifestPath())
	} else {
		return fmt.Sprintf("/bin/systemctl restart %s", c.ServiceName())
	}
}
