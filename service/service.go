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

package service

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/op/go-logging"

	"github.com/pulcy/gluon/systemd"
	"github.com/pulcy/gluon/util"
)

const (
	defaultVaultMonkeyImage = "pulcy/vault-monkey:0.7.0"
	clusterMembersPath      = "/etc/pulcy/cluster-members"
	privateRegistryUrlPath  = "/etc/pulcy/private-registry-url"
	etcdClusterStatePath    = "/etc/pulcy/etcd-cluster-state"
	gluonImagePath          = "/etc/pulcy/gluon-image"
	privateHostIPPrefix     = "private-host-ip="
	rolesPath               = "/etc/pulcy/roles"
	clusterIDPath           = "/etc/pulcy/cluster-id"
)

type Service interface {
	Name() string
	Setup(deps ServiceDependencies, flags *ServiceFlags) error
}

type ServiceDependencies struct {
	Systemd *systemd.SystemdClient
	Logger  *logging.Logger
}

type ServiceFlags struct {
	Force bool // Start/reload even if nothing has changed

	// gluon
	GluonImage       string
	VaultMonkeyImage string
	Roles            []string

	// Docker
	Docker struct {
		DockerIP                string
		DockerSubnet            string
		PrivateRegistryUrl      string
		PrivateRegistryUserName string
		PrivateRegistryPassword string
	}

	// Rkt
	Rkt struct {
		RktSubnet string
	}

	// Network
	Network struct {
		PrivateClusterDevice string
		ClusterSubnet        string // 'a.b.c.d/x'
		ClusterIP            string // IP address of member used for internal cluster traffic (e.g. etcd)
	}

	// ETCD
	Etcd Etcd

	// Kubernetes config
	Kubernetes Kubernetes

	// Vault config
	Vault Vault

	// Weave
	Weave Weave

	// private cache
	clusterMembers []ClusterMember
}

type discoveryResponse struct {
	Node   discoveryNode `json:"node"`
	Action string        `json:"action"`
}

type discoveryNode struct {
	Key   string          `json:"key,omitempty"`
	Value string          `json:"value,omitempty"`
	Nodes []discoveryNode `json:"nodes,omitempty"`
}

type ClusterMember struct {
	MachineID     string
	ClusterIP     string // IP address of member used for internal cluster traffic (e.g. etcd)
	PrivateHostIP string // IP address of member host (can be same as ClusterIP)
	EtcdProxy     bool
}

// SetupDefaults fills given flags with default value
func (flags *ServiceFlags) SetupDefaults(log *logging.Logger) error {
	if flags.VaultMonkeyImage == "" {
		flags.VaultMonkeyImage = defaultVaultMonkeyImage
	}
	if flags.Docker.PrivateRegistryUrl == "" {
		url, err := ioutil.ReadFile(privateRegistryUrlPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.Docker.PrivateRegistryUrl = string(url)
		}
	}
	if err := flags.Etcd.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Kubernetes.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if err := flags.Vault.setupDefaults(log); err != nil {
		return maskAny(err)
	}
	if flags.Network.PrivateClusterDevice == "" {
		flags.Network.PrivateClusterDevice = "eth1"
	}
	if flags.Network.ClusterSubnet == "" {
		ip := net.ParseIP(flags.Network.ClusterIP)
		mask := ip.DefaultMask()
		network := net.IPNet{IP: ip, Mask: mask}
		flags.Network.ClusterSubnet = network.String()
	}
	if flags.GluonImage == "" {
		content, err := ioutil.ReadFile(gluonImagePath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.GluonImage = strings.TrimSpace(string(content))
		}
	}
	if err := flags.Weave.setupDefaults(log, flags); err != nil {
		return maskAny(err)
	}

	// Setup roles last, since it depends on other flags being initialized
	if len(flags.Roles) == 0 {
		content, err := ioutil.ReadFile(rolesPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			lines := trimLines(strings.Split(string(content), "\n"))
			roles := strings.Join(lines, ",")
			flags.Roles = strings.Split(roles, ",")
		}
	}
	return nil
}

// Save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *ServiceFlags) Save(log *logging.Logger) (bool, error) {
	changes := 0
	if flags.Docker.PrivateRegistryUrl != "" {
		if changed, err := updateContent(log, privateRegistryUrlPath, flags.Docker.PrivateRegistryUrl, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if changed, err := flags.Etcd.save(log); err != nil {
		return false, maskAny(err)
	} else if changed {
		changes++
	}
	if changed, err := flags.Kubernetes.save(log); err != nil {
		return false, maskAny(err)
	} else if changed {
		changes++
	}
	if changed, err := flags.Vault.save(log); err != nil {
		return false, maskAny(err)
	} else if changed {
		changes++
	}
	if flags.GluonImage != "" {
		if changed, err := updateContent(log, gluonImagePath, flags.GluonImage, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if changed, err := flags.Weave.save(log); err != nil {
		return false, maskAny(err)
	} else if changed {
		changes++
	}
	if len(flags.Roles) > 0 {
		content := strings.Join(flags.Roles, "\n")
		if changed, err := updateContent(log, rolesPath, content, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	return (changes > 0), nil
}

// HasRole returns true if the given role is found in flags.Roles.
func (flags *ServiceFlags) HasRole(role string) bool {
	for _, x := range flags.Roles {
		if x == role {
			return true
		}
	}
	return false
}

// GetClusterMembers returns a list of the private IP
// addresses of all the cluster members
func (flags *ServiceFlags) GetClusterMembers(log *logging.Logger) ([]ClusterMember, error) {
	if flags.clusterMembers != nil {
		return flags.clusterMembers, nil
	}

	members, err := flags.getClusterMembersFromFS(log)
	if err != nil {
		return nil, maskAny(err)
	}

	flags.clusterMembers = members
	return members, nil
}

// ReadClusterID reads the cluster ID from /etc/pulcy/cluster-id
func (flags *ServiceFlags) ReadClusterID() (string, error) {
	content, err := ioutil.ReadFile(clusterIDPath)
	if err != nil {
		return "", maskAny(err)
	}
	return strings.TrimSpace(string(content)), nil
}

// PrivateHostIP returns the private IPv4 address of the host.
func (flags *ServiceFlags) PrivateHostIP(log *logging.Logger) (string, error) {
	members, err := flags.GetClusterMembers(log)
	if err != nil {
		return "", maskAny(err)
	}
	for _, m := range members {
		if m.ClusterIP == flags.Network.ClusterIP {
			return m.PrivateHostIP, nil
		}
	}
	return "", maskAny(fmt.Errorf("No cluster member found for %s", flags.Network.ClusterIP))
}

// getClusterMembersFromFS returns a list of the private IP
// addresses from a local configuration file
func (flags *ServiceFlags) getClusterMembersFromFS(log *logging.Logger) ([]ClusterMember, error) {
	content, err := ioutil.ReadFile(clusterMembersPath)
	if err != nil {
		return nil, maskAny(err)
	}
	lines := strings.Split(string(content), "\n")

	// Find IP addresses
	members := []ClusterMember{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		id := parts[0]
		parts = strings.Split(parts[1], " ")
		clusterIP := parts[0]
		privateHostIP := clusterIP
		etcdProxy := false
		for index, x := range parts {
			if index == 0 {
				continue
			}
			switch x {
			case "etcd-proxy":
				etcdProxy = true
			default:
				if strings.HasPrefix(x, privateHostIPPrefix) {
					privateHostIP = x[len(privateHostIPPrefix):]
				} else {
					log.Error("Unknown option '%s' in %s", x, clusterMembersPath)
				}
			}
		}

		members = append(members, ClusterMember{
			MachineID:     id,
			ClusterIP:     clusterIP,
			PrivateHostIP: privateHostIP,
			EtcdProxy:     etcdProxy,
		})
	}

	return members, nil
}

func updateContent(log *logging.Logger, path, content string, fileMode os.FileMode) (bool, error) {
	content = strings.TrimSpace(content)
	os.MkdirAll(filepath.Dir(path), 0755)
	changed, err := util.UpdateFile(log, path, []byte(content), fileMode)
	return changed, maskAny(err)
}

// trimLines trims the spaces away from every given line and leaves out any empty lines.
func trimLines(lines []string) []string {
	result := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
