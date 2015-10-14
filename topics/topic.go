package topics

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/op/go-logging"

	"arvika.pulcy.com/pulcy/yard/systemd"
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
	// Etcd
	DiscoveryUrl string

	// Docker
	DockerSubnet string

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

	resp, err := http.Get(flags.DiscoveryUrl)
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

	flags.privateIPs = addresses
	return addresses, nil
}
