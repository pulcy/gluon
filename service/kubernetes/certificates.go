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
	"net"
	"strings"
	"text/template"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/util"
)

const (
	certsServiceTemplate  = "templates/kubernetes/certs.service.tmpl"
	certsTemplateTemplate = "templates/kubernetes/certs.template.tmpl"
)

// createCertsTemplate creates the consul-template used by the k8s-certs service.
func createCertsTemplate(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component, altNames []string, addInternalApiServerIP bool) (bool, error) {
	certsTemplatesPath := c.CertificatesTemplatePath()
	if err := util.EnsureDirectoryOf(certsTemplatesPath, 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", certsTemplatesPath)
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	privateHostIP, err := flags.PrivateHostIP(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ClusterID  string
		CommonName string
		Component  string
		IPSans     string
		SecretArgs string
		CertPath   string
		KeyPath    string
		CAPath     string
	}{
		ClusterID:  clusterID,
		CommonName: c.Name(),
		Component:  c.Name(),
		IPSans:     strings.Join([]string{flags.Network.ClusterIP, privateHostIP}, ","),
		SecretArgs: "",
		CertPath:   c.CertificatePath(),
		KeyPath:    c.KeyPath(),
		CAPath:     c.CAPath(),
	}
	if len(altNames) > 0 {
		opts.SecretArgs = fmt.Sprintf(`"alt_names=%s" `, strings.Join(altNames, ","))
	}
	if addInternalApiServerIP {
		serviceIP, _, err := net.ParseCIDR(flags.Kubernetes.ServiceClusterIPRange)
		if err != nil {
			return false, maskAny(err)
		}
		internalApiServerIP := serviceIP.To4()
		internalApiServerIP[3] = 1
		opts.IPSans = opts.IPSans + "," + internalApiServerIP.String()
	}
	setDelims := func(t *template.Template) {
		t.Delims("[[", "]]")
	}
	changed, err := templates.Render(deps.Logger, certsTemplateTemplate, certsTemplatesPath, opts, templateFileMode, setDelims)
	return changed, maskAny(err)
}

// createCertsService creates the k8s-certs service.
func createCertsService(deps service.ServiceDependencies, flags *service.ServiceFlags, c Component) (bool, error) {
	deps.Logger.Info("creating %s", c.CertificatesServicePath())
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		VaultMonkeyImage   string
		ConsulAddress      string
		JobID              string
		TemplatePath       string
		TemplateOutputPath string
		ConfigFileName     string
		Component          string
		RestartCommand     string
		TokenTemplate      string
		TokenPolicy        string
		TokenRole          string
	}{
		VaultMonkeyImage:   flags.VaultMonkeyImage,
		ConsulAddress:      flags.Network.ClusterIP + ":8500",
		JobID:              c.JobID(clusterID),
		TemplatePath:       c.CertificatesTemplatePath(),
		TemplateOutputPath: c.CertificatesTemplateOutputPath(),
		ConfigFileName:     c.CertificatesConfigName(),
		Component:          c.Name(),
		RestartCommand:     c.RestartCommand(),
		TokenTemplate:      `{ "vault": { "token": "{{.Token}}" }}`,
		TokenPolicy:        tokenPolicy(clusterID, c.Name()),
		TokenRole:          tokenRole(clusterID, c.Name()),
	}
	changed, err := templates.Render(deps.Logger, certsServiceTemplate, c.CertificatesServicePath(), opts, serviceFileMode)
	return changed, maskAny(err)
}
