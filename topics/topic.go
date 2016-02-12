package topics

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/op/go-logging"

	"github.com/pulcy/yard/systemd"
)

const (
	discoveryUrlPath       = "/etc/pulcy/discovery-url"
	clusterMembersPath     = "/etc/pulcy/cluster-members"
	yardPassphrasePath     = "/etc/pulcy/yard-passphrase"
	privateRegistryUrlPath = "/etc/pulcy/private-registry-url"
)

type Topic interface {
	Name() string
	Defaults(flags *TopicFlags) error
	Setup(deps *TopicDependencies, flags *TopicFlags) error
}

type TopicDependencies struct {
	Systemd *systemd.SystemdClient
	Logger  *logging.Logger
}

type TopicFlags struct {
	// yard
	YardPassphrase string
	YardImage      string

	// ETCD discovery URL
	DiscoveryURL string

	// Docker
	DockerIP                string
	DockerSubnet            string
	PrivateRegistryUrl      string
	PrivateRegistryUserName string
	PrivateRegistryPassword string

	// IPTables
	PrivateClusterDevice string

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
}

// SetupDefaults fills given flags with default value
func (flags *TopicFlags) SetupDefaults(yardVersion string) error {
	if flags.DiscoveryURL == "" {
		url, err := ioutil.ReadFile(discoveryUrlPath)
		if err != nil {
			return maskAny(err)
		}
		flags.DiscoveryURL = string(url)
	}
	if flags.YardPassphrase == "" {
		passphrase, err := ioutil.ReadFile(yardPassphrasePath)
		if err != nil {
			return maskAny(err)
		}
		flags.YardPassphrase = string(passphrase)
	}
	if flags.PrivateRegistryUrl == "" {
		url, err := ioutil.ReadFile(privateRegistryUrlPath)
		if err != nil {
			return maskAny(err)
		}
		flags.PrivateRegistryUrl = string(url)
	}
	if flags.YardImage == "" && yardVersion != "" {
		flags.YardImage = fmt.Sprintf("pulcy/yard:%s", yardVersion)
	}
	if flags.PrivateClusterDevice == "" {
		flags.PrivateClusterDevice = "eth1"
	}
	return nil
}

// Save applicable flags to their respective files
func (flags *TopicFlags) Save() error {
	if flags.DiscoveryURL != "" {
		if err := updateContent(discoveryUrlPath, flags.DiscoveryURL, 0644); err != nil {
			return maskAny(err)
		}
	}
	if flags.YardPassphrase != "" {
		if err := updateContent(yardPassphrasePath, flags.YardPassphrase, 0400); err != nil {
			return maskAny(err)
		}
	}
	if flags.PrivateRegistryUrl != "" {
		if err := updateContent(privateRegistryUrlPath, flags.PrivateRegistryUrl, 0644); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

// GetClusterMembers returns a list of the private IP
// addresses of all the cluster members
func (flags *TopicFlags) GetClusterMembers() ([]ClusterMember, error) {
	if flags.clusterMembers != nil {
		return flags.clusterMembers, nil
	}

	members, err := flags.getClusterMembersFromFS()
	if err != nil {
		return nil, maskAny(err)
	}

	flags.clusterMembers = members
	return members, nil
}

// getClusterMembersFromFS returns a list of the private IP
// addresses from a local configuration file
func (flags *TopicFlags) getClusterMembersFromFS() ([]ClusterMember, error) {
	content, err := ioutil.ReadFile(clusterMembersPath)
	if err != nil {
		return nil, maskAny(err)
	}
	lines := strings.Split(string(content), "\n")

	// Find IP addresses
	members := []ClusterMember{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				members = append(members, ClusterMember{
					MachineID: parts[0],
					PrivateIP: parts[1],
				})
			}
		}

	}

	return members, nil
}

func updateContent(path, content string, fileMode os.FileMode) error {
	oldContent, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return maskAny(err)
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	if strings.TrimSpace(string(oldContent)) != strings.TrimSpace(content) {
		ioutil.WriteFile(path, []byte(content), fileMode)
	}
	return nil
}
