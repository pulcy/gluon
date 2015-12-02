package topics

import (
	"io/ioutil"
	"strings"

	"github.com/op/go-logging"

	"arvika.pulcy.com/pulcy/yard/systemd"
)

const (
	clusterMembersPath = "/etc/pulcy/cluster-members"
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
