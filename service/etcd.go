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
	"os"
	"strings"

	"github.com/op/go-logging"
)

// ETCD
type Etcd struct {
	ClusterState  string
	UseVaultCA    bool // If set, use vault to create peer (and optional client) TLS certificates
	SecureClients bool // If set, force clients to connect over TLS
	ClientPort    int
}

const (
	defaultEtcdClientPort = 2379
)

// setupDefaults fills given flags with default value
func (flags *Etcd) setupDefaults(log *logging.Logger) error {
	if flags.ClientPort == 0 {
		flags.ClientPort = defaultEtcdClientPort
	}
	if flags.ClusterState == "" {
		raw, err := ioutil.ReadFile(etcdClusterStatePath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			lines := trimLines(strings.Split(string(raw), "\n"))
			flags.ClusterState = strings.TrimSpace(strings.Join(lines, " "))
		}
	}
	return nil
}

// save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *Etcd) save() (bool, error) {
	changes := 0
	if flags.ClusterState != "" {
		if changed, err := updateContent(etcdClusterStatePath, flags.ClusterState, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	return (changes > 0), nil
}

// CreateEndpoint returns the client URL to reach an ETCD server at the given cluster IP.
func (flags *Etcd) CreateEndpoint(ip string) string {
	if flags.SecureClients {
		return fmt.Sprintf("https://%s:%d", ip, flags.ClientPort)
	} else {
		return fmt.Sprintf("http://%s:%d", ip, flags.ClientPort)
	}
}
