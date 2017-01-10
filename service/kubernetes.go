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

import "github.com/op/go-logging"

// K8s config
type Kubernetes struct {
	KubernetesMasterImage string
	APIServerPort         int
	ServiceClusterIPRange string
	ClusterDNS            string
}

const (
	defaultKubernetesMasterImage = "pulcy/k8s-master:0.1.4"
	defaultServiceClusterIPRange = "10.71.0.0/16"
	defaultAPIServerPort         = 6443
	defaultClusterDNS            = "10.32.0.10"
)

// SetupDefaults fills given flags with default value
func (flags *Kubernetes) SetupDefaults(log *logging.Logger) error {
	if flags.KubernetesMasterImage == "" {
		flags.KubernetesMasterImage = defaultKubernetesMasterImage
	}
	if flags.APIServerPort == 0 {
		flags.APIServerPort = defaultAPIServerPort
	}
	if flags.ServiceClusterIPRange == "" {
		flags.ServiceClusterIPRange = defaultServiceClusterIPRange
	}
	if flags.ClusterDNS == "" {
		flags.ClusterDNS = defaultClusterDNS
	}
	return nil
}

// Save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *Kubernetes) Save() (bool, error) {
	changes := 0
	return (changes > 0), nil
}
