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
	"path"
	"strings"
	"text/template"

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
	confName                 = "99-etcd2.conf"
	confDir                  = "/etc/systemd/system/" + serviceName + ".d/"
	confPath                 = confDir + confName
	serviceName              = "etcd2.service"
	serviceTemplate          = "templates/etcd/etcd.service.tmpl"
	servicePath              = "/etc/systemd/system/" + serviceName
	environmentPath          = "/etc/environment"
	etcdUser                 = "etcd"
	dataPath                 = "/var/lib/etcd2"
	initTemplate             = "templates/etcd/etcd-init.sh.tmpl"
	initPath                 = "/root/etcd-init.sh"
	certsServiceName         = "etcd-certs.service"
	certsServiceTemplate     = "templates/etcd/" + certsServiceName + ".tmpl"
	certsServicePath         = "/etc/systemd/system/" + certsServiceName
	certsTemplateName        = "etcd-certs.template"
	certsTemplateTemplate    = "templates/etcd/" + certsTemplateName + ".tmpl"
	certsTemplatesPath       = "/opt/certs/" + certsTemplateName
	certsTemplatesOutputPath = "/opt/certs/etcd.serial"
	CertsCertPath            = "/opt/certs/etcd-cert.pem"
	CertsKeyPath             = "/opt/certs/etcd-key.pem"
	CertsCAPath              = "/opt/certs/etcd-ca.pem"

	configFileMode   = os.FileMode(0644)
	serviceFileMode  = os.FileMode(0644)
	templateFileMode = os.FileMode(0400)
	dataPathMode     = os.FileMode(0755)
	initFileMode     = os.FileMode(0755)
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

	if err := createEtcdUserAndPath(deps); err != nil {
		return maskAny(err)
	}
	if err := addCoreToEtcdGroup(deps, flags); err != nil {
		return maskAny(err)
	}

	var certsTemplateChanged, certsServiceChanged bool
	if flags.Etcd.UseVaultCA {
		// Create etcd-certs.service and template file
		if certsTemplateChanged, err = createCertsTemplate(deps, flags, cfg); err != nil {
			return maskAny(err)
		}
		if certsServiceChanged, err = createCertsService(deps, flags); err != nil {
			return maskAny(err)
		}
		isActive, err := deps.Systemd.IsActive(certsServiceName)
		if err != nil {
			return maskAny(err)
		}

		if !isActive || certsTemplateChanged || certsServiceChanged || flags.Force {
			if err := deps.Systemd.Enable(certsServiceName); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Reload(); err != nil {
				return maskAny(err)
			}
			if err := deps.Systemd.Restart(certsServiceName); err != nil {
				return maskAny(err)
			}
		}
	} else {
		// etcd-certs no longer needed, remove it
		if exists, err := deps.Systemd.Exists(certsServiceName); err != nil {
			return maskAny(err)
		} else if exists {
			if err := deps.Systemd.Disable(certsServiceName); err != nil {
				deps.Logger.Errorf("Disabling %s failed: %#v", certsServiceName, err)
			} else {
				os.Remove(certsServicePath)
				os.Remove(certsTemplatesPath)
			}
		}
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

		if !isActive || certsTemplateChanged || certsServiceChanged || changedService || changedConf || flags.Force {
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

func addCoreToEtcdGroup(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	for _, g := range []string{"etcd"} {
		cmd := exec.Command("gpasswd", "-a", "core", g)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return maskAny(err)
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
	Scheme              string // URL scheme of 1 ETCD host
	UseVaultCA          bool
	SecureClients       bool
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
		ClusterIP:     flags.Network.ClusterIP,
		UseVaultCA:    flags.Etcd.UseVaultCA,
		SecureClients: flags.Etcd.SecureClients,
	}
	initialCluster := []string{}
	endpoints := []string{}
	hosts := []string{}
	clientPort := flags.Etcd.ClientPort
	result.ClusterState = flags.Etcd.ClusterState
	if result.ClusterState == "" {
		result.ClusterState = "new"
	}
	memberIndex := 0
	clientScheme := "http"
	if flags.Etcd.SecureClients {
		clientScheme = "https"
	}
	for index, cm := range members {
		if !cm.EtcdProxy {
			initialCluster = append(initialCluster,
				fmt.Sprintf("%s=https://%s:2380", cm.MachineID, cm.ClusterIP),
				fmt.Sprintf("%s=https://%s:2381", cm.MachineID, cm.PrivateHostIP),
			)
			endpoints = append(endpoints, fmt.Sprintf("%s://%s:%d", clientScheme, cm.ClusterIP, clientPort))
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
	result.ListenClientURLs = strings.Join([]string{
		fmt.Sprintf("%s://%s:%d", clientScheme, flags.Network.ClusterIP, clientPort),
		fmt.Sprintf("%s://%s:4001", clientScheme, flags.Network.ClusterIP),
		fmt.Sprintf("%s://127.0.0.1:%d", clientScheme, clientPort),
		fmt.Sprintf("%s://127.0.0.1:4001", clientScheme),
	}, ",")
	result.AdvertiseClientURLs = strings.Join([]string{
		fmt.Sprintf("%s://%s:%d", clientScheme, flags.Network.ClusterIP, clientPort),
		fmt.Sprintf("%s://%s:4001", clientScheme, flags.Network.ClusterIP),
	}, ",")

	result.InitialCluster = strings.Join(initialCluster, ",")
	result.Endpoints = strings.Join(endpoints, ",")
	result.Host = hosts[memberIndex%len(hosts)]
	result.Port = strconv.Itoa(clientPort)
	result.Scheme = clientScheme

	return result, nil

}

// createCertsTemplate creates the consul-template used by the etcd-certs service.
func createCertsTemplate(deps service.ServiceDependencies, flags *service.ServiceFlags, config etcdConfig) (bool, error) {
	if err := util.EnsureDirectoryOf(certsTemplatesPath, 0755); err != nil {
		return false, maskAny(err)
	}
	deps.Logger.Info("creating %s", certsTemplatesPath)
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ClusterID  string
		CommonName string
		IPSans     string
		CertPath   string
		KeyPath    string
		CAPath     string
	}{
		ClusterID:  clusterID,
		CommonName: hostname,
		IPSans:     strings.Join([]string{config.ClusterIP, config.PrivateHostIP}, ","),
		CertPath:   CertsCertPath,
		KeyPath:    CertsKeyPath,
		CAPath:     CertsCAPath,
	}
	setDelims := func(t *template.Template) {
		t.Delims("[[", "]]")
	}
	changed, err := templates.Render(certsTemplateTemplate, certsTemplatesPath, opts, templateFileMode, setDelims)
	return changed, maskAny(err)
}

// createCertsService creates the etcd-certs service.
func createCertsService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", certsServicePath)
	clusterID, err := flags.ReadClusterID()
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ConsulAddress      string
		JobID              string
		TemplatePath       string
		TemplateOutputPath string
		ServiceName        string
		TokenTemplate      string
		TokenPolicy        string
		TokenRole          string
	}{
		ConsulAddress:      flags.Network.ClusterIP + ":8500",
		JobID:              fmt.Sprintf("ca-%s-pki-etcd", clusterID),
		TemplatePath:       certsTemplatesPath,
		TemplateOutputPath: certsTemplatesOutputPath,
		ServiceName:        serviceName, // Note this is the name of the ETCD service itself.
		TokenTemplate:      `{ "vault": { "token": "{{.Token}}" }}`,
		TokenPolicy:        path.Join("ca", clusterID, "pki/etcd/member"),
		TokenRole:          fmt.Sprintf("etcd-%s", clusterID),
	}
	changed, err := templates.Render(certsServiceTemplate, certsServicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func createService(deps service.ServiceDependencies, flags *service.ServiceFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		Requires []string
		After    []string
	}{
		Requires: []string{},
		After:    []string{},
	}
	if flags.Etcd.UseVaultCA {
		opts.After = append(opts.After, certsServiceName)
	}
	changed, err := templates.Render(serviceTemplate, servicePath, opts, serviceFileMode)
	return changed, maskAny(err)
}

func createEtcd2Conf(deps service.ServiceDependencies, cfg etcdConfig) (bool, error) {
	if cfg.ClusterIP == "" {
		return false, maskAny(fmt.Errorf("ClusterIP empty"))
	}
	deps.Logger.Info("creating %s", confPath)

	lines := []string{
		"[Service]",
		"Environment=ETCD_LISTEN_PEER_URLS=" + cfg.ListenPeerURLs,
		"Environment=ETCD_LISTEN_CLIENT_URLS=" + cfg.ListenClientURLs,
		"Environment=ETCD_INITIAL_CLUSTER=" + cfg.InitialCluster,
		"Environment=ETCD_INITIAL_CLUSTER_STATE=" + cfg.ClusterState,
		"Environment=ETCD_INITIAL_ADVERTISE_PEER_URLS=" + cfg.AdvertisePeerURLs,
		"Environment=ETCD_ADVERTISE_CLIENT_URLS=" + cfg.AdvertiseClientURLs,
	}
	if cfg.UseVaultCA {
		lines = append(lines,
			"Environment=ETCD_PEER_CERT_FILE="+CertsCertPath,
			"Environment=ETCD_PEER_KEY_FILE="+CertsKeyPath,
			"Environment=ETCD_PEER_TRUSTED_CA_FILE="+CertsCAPath,
			"Environment=ETCD_PEER_CLIENT_CERT_AUTH=true",
		)
		if cfg.SecureClients {
			lines = append(lines,
				"Environment=ETCD_CERT_FILE="+CertsCertPath,
				"Environment=ETCD_KEY_FILE="+CertsKeyPath,
				"Environment=ETCD_TRUSTED_CA_FILE="+CertsCAPath,
				"Environment=ETCD_CLIENT_CERT_AUTH=true",
			)
		}
	} else {
		lines = append(lines, "Environment=ETCD_PEER_AUTO_TLS=true")
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
		util.KeyValuePair{Key: "ETCDCTL_ENDPOINT", Value: cfg.Endpoints},
		util.KeyValuePair{Key: "ETCDCTL_ENDPOINTS", Value: cfg.Endpoints},
		util.KeyValuePair{Key: "ETCD_HOST", Value: cfg.Host},
		util.KeyValuePair{Key: "ETCD_PORT", Value: cfg.Port},
		util.KeyValuePair{Key: "ETCD_SCHEME", Value: cfg.Scheme},
	}

	if cfg.SecureClients {
		kv = append(kv,
			util.KeyValuePair{Key: "ETCD_CERT_FILE", Value: CertsCertPath},
			util.KeyValuePair{Key: "ETCD_KEY_FILE", Value: CertsKeyPath},
			util.KeyValuePair{Key: "ETCD_TRUSTED_CA_FILE", Value: CertsCAPath},
			// etcdctl 2
			util.KeyValuePair{Key: "ETCDCTL_CERT_FILE", Value: CertsCertPath},
			util.KeyValuePair{Key: "ETCDCTL_KEY_FILE", Value: CertsKeyPath},
			util.KeyValuePair{Key: "ETCDCTL_CA_FILE", Value: CertsCAPath},
			// etcdctl 3
			util.KeyValuePair{Key: "ETCDCTL_CERT", Value: CertsCertPath},
			util.KeyValuePair{Key: "ETCDCTL_KEY", Value: CertsKeyPath},
			util.KeyValuePair{Key: "ETCDCTL_CACERT", Value: CertsCAPath},
		)
	}

	changed, err := util.AppendEnvironmentFile(environmentPath, kv, configFileMode)
	return changed, maskAny(err)
}
