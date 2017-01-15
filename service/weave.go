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
	"strings"

	"github.com/op/go-logging"

	"github.com/pulcy/gluon/util"
)

const (
	weaveSeedPath          = "/etc/pulcy/weave-seed"
	weaveIPRangePath       = "/etc/pulcy/weave-iprange"
	weaveIPInitPath        = "/etc/pulcy/weave-ipinit"
	defaultWeaveIPRange    = "10.32.0.0/12"
	defaultWeaveRktSubnet  = "10.22.0.0/16"
	defaultWeaveRktGateway = "10.22.0.1"
)

// Weave config
type Weave struct {
	Seed       string
	Hostname   string // Weave DNS of exposed host
	IPRange    string // Value to `--ipalloc-range` (e.g. 10.32.0.0/16)
	IPInit     string // Value for `--ipalloc-init` (default empty)
	RktSubnet  string // Subnet used by rkt (e.g. 10.22.0.0/16)
	RktGateway string // Gateway used by rkt (e.g. 10.22.0.1)
}

// setupDefaults fills given flags with default value
func (flags *Weave) setupDefaults(log *logging.Logger, serviceFlags *ServiceFlags) error {
	if flags.Seed == "" {
		seed, err := ioutil.ReadFile(weaveSeedPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.Seed = string(seed)
		} else {
			members, err := serviceFlags.GetClusterMembers(log)
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
			flags.Seed = strings.Join(seeds, ",")
		}
	}
	if flags.IPRange == "" {
		content, err := ioutil.ReadFile(weaveIPRangePath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.IPRange = strings.TrimSpace(string(content))
		} else {
			flags.IPRange = defaultWeaveIPRange
		}
	}
	if flags.IPInit == "" {
		content, err := ioutil.ReadFile(weaveIPInitPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			flags.IPInit = strings.TrimSpace(string(content))
		}
	}
	if flags.RktSubnet == "" {
		flags.RktSubnet = defaultWeaveRktSubnet
	}
	if flags.RktGateway == "" {
		flags.RktGateway = defaultWeaveRktGateway
	}
	return nil
}

// Save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *Weave) save(log *logging.Logger) (bool, error) {
	changes := 0
	if flags.Seed != "" {
		if changed, err := updateContent(log, weaveSeedPath, flags.Seed, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	if flags.IPRange != "" {
		if changed, err := updateContent(log, weaveIPRangePath, flags.IPRange, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	return (changes > 0), nil
}
