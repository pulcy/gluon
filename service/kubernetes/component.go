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

type Component string

// ServiceName returns the name of the systemd service that runs the component.
func (c Component) ServiceName() string {
	return fmt.Sprintf("k8s-%s.service", c)
}

// ServicePath returns the full path of the file containing the systemd service that runs the component.
func (c Component) ServicePath() string {
	return servicePath(c.ServiceName())
}

// CertificatesServiceName returns the name of the systemd service that generates the TLS certificates for the component.
func (c Component) CertificatesServiceName() string {
	return fmt.Sprintf("k8s-%s-certs.service", c)
}

// CertificatesServicePath returns the full path of the file containing the systemd service that generates the TLS certificates for the component.
func (c Component) CertificatesServicePath() string {
	return servicePath(c.CertificatesServiceName())
}

// CertificatesTemplateName returns the name of the consul-template template that generates the TLS certificates for the component.
func (c Component) CertificatesTemplateName() string {
	return fmt.Sprintf("k8s-%s-certs.template", c)
}

// CertificatesTemplatePath returns the full path of the consul-template template that generates the TLS certificates for the component.
func (c Component) CertificatesTemplatePath() string {
	return certificatePath(c.CertificatesTemplateName())
}

// CertificatesTemplateOutputPath returns the full path of the file created by the consul-template template that generates the TLS certificates for the component.
func (c Component) CertificatesTemplateOutputPath() string {
	return certificatePath(fmt.Sprintf("/opt/certs/k8s-%s.serial", c))
}

// CertificatesConfigName returns the name config file used by consul-template for the component.
func (c Component) CertificatesConfigName() string {
	return fmt.Sprintf("k8s-%s-certs-config.json", c)
}

// CertificatePath returns the full path of the public key part of the certificate for this component.
func (c Component) CertificatePath() string {
	return fmt.Sprintf("/opt/certs/k8s-%s-cert.pem", c)
}

// KeyPath returns the full path of the private key part of the certificate for this component.
func (c Component) KeyPath() string {
	return fmt.Sprintf("/opt/certs/k8s-%s-key.pem", c)
}

// CAPath returns the full path of the CA certificate for this component.
func (c Component) CAPath() string {
	return fmt.Sprintf("/opt/certs/k8s-%s-ca.pem", c)
}

// KubeConfigPath returns the full path of the kubeconfig configuration file for this component.
func (c Component) KubeConfigPath() string {
	return fmt.Sprintf("/var/lib/%s/kubeconfig", c)
}

// JobID returns the ID of the vault-monkey job used to access certificates for this component.
func (c Component) JobID(clusterID string) string {
	return fmt.Sprintf("ca-%s-pki-k8s-%s", clusterID, c)
}
