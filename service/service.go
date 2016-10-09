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
	etcdClusterStatePath   = "/etc/pulcy/etcd-cluster-state"
	fleetMetadataPath      = "/etc/pulcy/fleet-metadata"
	gluonImagePath         = "/etc/pulcy/gluon-image"
	weaveSeedPath          = "/etc/pulcy/weave-seed"
	privateHostIPPrefix    = "private-host-ip="
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
	Docker struct {
		DockerIP                string
		DockerSubnet            string
		PrivateRegistryUrl      string
		PrivateRegistryUserName string
		PrivateRegistryPassword string
	}

	// Network
	Network struct {
		PrivateClusterDevice string
		ClusterIP            string // IP address of member used for internal cluster traffic (e.g. etcd)
	}

	// ETCD
	Etcd struct {
		ClusterState string
	}

	// Fleet
	Fleet struct {
		Metadata                string
		AgentTTL                string
		DisableEngine           bool
		DisableWatches          bool
		EngineReconcileInterval int
		TokenLimit              int
	}

	// Weave
	Weave struct {
		Seed     string
		Hostname string // Weave DNS of exposed host
	}

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
	if flags.Docker.PrivateRegistryUrl == "" {
		url, err := ioutil.ReadFile(privateRegistryUrlPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.Docker.PrivateRegistryUrl = string(url)
		}
	}
	if flags.Fleet.Metadata == "" {
		raw, err := ioutil.ReadFile(fleetMetadataPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			lines := trimLines(strings.Split(string(raw), "\n"))
			flags.Fleet.Metadata = strings.Join(lines, ",")
		}
	}
	if flags.Etcd.ClusterState == "" {
		raw, err := ioutil.ReadFile(etcdClusterStatePath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			lines := trimLines(strings.Split(string(raw), "\n"))
			flags.Etcd.ClusterState = strings.TrimSpace(strings.Join(lines, " "))
		}
	}
	if flags.Network.PrivateClusterDevice == "" {
		flags.Network.PrivateClusterDevice = "eth1"
	}
	if flags.GluonImage == "" {
		content, err := ioutil.ReadFile(gluonImagePath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.GluonImage = strings.TrimSpace(string(content))
		}
	}
	if flags.Weave.Seed == "" {
		seed, err := ioutil.ReadFile(weaveSeedPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.Weave.Seed = string(seed)
		} else {
			members, err := flags.GetClusterMembers(log)
			if err != nil {
				return maskAny(err)
			}
			var seeds []string
			for _, m := range members {
				if !m.EtcdProxy {
					name, err := util.WeaveNameFromMachineID(m.MachineID)
					if err != nil {
						return maskAny(err)
					}
					seeds = append(seeds, name)
				}
			}
			flags.Weave.Seed = strings.Join(seeds, ",")
		}
	}
	return nil
}

// Save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *ServiceFlags) Save() (bool, error) {
	changes := 0
	if flags.Docker.PrivateRegistryUrl != "" {
		if changed, err := updateContent(privateRegistryUrlPath, flags.Docker.PrivateRegistryUrl, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if flags.Fleet.Metadata != "" {
		parts := strings.Split(flags.Fleet.Metadata, ",")
		content := strings.Join(parts, "\n")
		if changed, err := updateContent(fleetMetadataPath, content, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if flags.Etcd.ClusterState != "" {
		if changed, err := updateContent(etcdClusterStatePath, flags.Etcd.ClusterState, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if flags.GluonImage != "" {
		if changed, err := updateContent(gluonImagePath, flags.GluonImage, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if flags.Weave.Seed != "" {
		if changed, err := updateContent(weaveSeedPath, flags.Weave.Seed, 0644); err != nil {
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

func updateContent(path, content string, fileMode os.FileMode) (bool, error) {
	content = strings.TrimSpace(content)
	os.MkdirAll(filepath.Dir(path), 0755)
	changed, err := util.UpdateFile(path, []byte(content), fileMode)
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
