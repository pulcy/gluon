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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/op/go-logging"

	"github.com/pulcy/gluon/systemd"
	"github.com/pulcy/gluon/util"
)

const (
	clusterMembersPath     = "/etc/pulcy/cluster-members"
	privateRegistryUrlPath = "/etc/pulcy/private-registry-url"
	fleetMetadataPath      = "/etc/pulcy/fleet-metadata"
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
	GluonImage string

	// Docker
	DockerIP                string
	DockerSubnet            string
	PrivateRegistryUrl      string
	PrivateRegistryUserName string
	PrivateRegistryPassword string

	// IPTables
	PrivateClusterDevice string
	PrivateIP            string // Private IPv4 address of this machine

	// Fleet
	FleetMetadata string

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
	MachineID string
	PrivateIP string
	EtcdProxy bool
}

// SetupDefaults fills given flags with default value
func (flags *ServiceFlags) SetupDefaults() error {
	if flags.PrivateRegistryUrl == "" {
		url, err := ioutil.ReadFile(privateRegistryUrlPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.PrivateRegistryUrl = string(url)
		}
	}
	if flags.FleetMetadata == "" {
		metadata, err := ioutil.ReadFile(fleetMetadataPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.FleetMetadata = string(metadata)
		}
	}
	if flags.PrivateClusterDevice == "" {
		flags.PrivateClusterDevice = "eth1"
	}
	return nil
}

// Save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *ServiceFlags) Save() (bool, error) {
	changes := 0
	if flags.PrivateRegistryUrl != "" {
		if changed, err := updateContent(privateRegistryUrlPath, flags.PrivateRegistryUrl, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if flags.FleetMetadata != "" {
		if changed, err := updateContent(fleetMetadataPath, flags.FleetMetadata, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	return (changes > 0), nil
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
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			continue
		}
		id := parts[0]
		parts = strings.Split(parts[1], " ")
		ip := parts[0]
		etcdProxy := false
		for index, x := range parts {
			if index == 0 {
				continue
			}
			switch x {
			case "etcd-proxy":
				etcdProxy = true
			default:
				log.Error("Unknown option '%s' in %s", x, clusterMembersPath)
			}
		}

		members = append(members, ClusterMember{
			MachineID: id,
			PrivateIP: ip,
			EtcdProxy: etcdProxy,
		})
	}

	return members, nil
}

func updateContent(path, content string, fileMode os.FileMode) (bool, error) {
	content = strings.TrimSpace(content)
	os.MkdirAll(filepath.Dir(path), 0755)
	changed, err := util.UpdateFile(path, []byte(content), fileMode)
	return changed, maskAny(err)
}
