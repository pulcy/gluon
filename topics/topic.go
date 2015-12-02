package topics

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/juju/errgo"
	"github.com/op/go-logging"

	"arvika.pulcy.com/pulcy/yard/systemd"
)

const (
	clusterMembersPath = "/etc/yard-cluster-members"
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
	privateIPs []string
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

// GetClusterMemberPrivateIPs returns a list of the private IP
// addresses of all the cluster members
func (flags *TopicFlags) GetClusterMemberPrivateIPs() ([]string, error) {
	if flags.privateIPs != nil {
		return flags.privateIPs, nil
	}

	addresses, err := flags.getClusterMembersFromFS()
	if err != nil {
		// Local config failed, try discovery
		addresses, err = flags.getClusterMembersFromDiscovery()
		if err != nil {
			return nil, maskAny(err)
		}
	}

	flags.privateIPs = addresses
	return addresses, nil
}

// getClusterMembersFromDiscovery returns a list of the private IP
// addresses of all the cluster members
func (flags *TopicFlags) getClusterMembersFromDiscovery() ([]string, error) {
	if flags.DiscoveryURL == "" {
		return nil, maskAny(errgo.New("discovery-url missing"))
	}
	resp, err := http.Get(flags.DiscoveryURL)
	if err != nil {
		return nil, maskAny(err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, maskAny(err)
	}

	// Decode response
	var discResp discoveryResponse
	if err := json.Unmarshal(body, &discResp); err != nil {
		return nil, maskAny(err)
	}

	// Find IP addresses
	addresses := []string{}
	for _, n := range discResp.Node.Nodes {
		parts := strings.SplitN(n.Value, "=", 2)
		if len(parts) == 2 {
			url, err := url.Parse(parts[1])
			if err == nil {
				host, _, err := net.SplitHostPort(url.Host)
				if err == nil {
					addresses = append(addresses, host)
				}
			}
		}
	}

	return addresses, nil
}

// getClusterMembersFromFS returns a list of the private IP
// addresses from a local configuration file
func (flags *TopicFlags) getClusterMembersFromFS() ([]string, error) {
	content, err := ioutil.ReadFile(clusterMembersPath)
	if err != nil {
		return nil, maskAny(err)
	}
	lines := strings.Split(string(content), "\n")

	// Find IP addresses
	addresses := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			addresses = append(addresses, line)
		}

	}

	return addresses, nil
}
