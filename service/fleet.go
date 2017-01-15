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
)

const (
	fleetMetadataPath = "/etc/pulcy/fleet-metadata"
)

// Fleet
type Fleet struct {
	Enabled                 bool
	Metadata                string
	AgentTTL                string
	DisableEngine           bool
	DisableWatches          bool
	EngineReconcileInterval int
	TokenLimit              int
}

// setupDefaults fills given flags with default value
func (flags *Fleet) setupDefaults(log *logging.Logger) error {
	if flags.Metadata == "" {
		raw, err := ioutil.ReadFile(fleetMetadataPath)
		if err != nil && !os.IsNotExist(err) {
			return maskAny(err)
		} else if err == nil {
			lines := trimLines(strings.Split(string(raw), "\n"))
			flags.Metadata = strings.Join(lines, ",")
		}
	}
	return nil
}

// save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *Fleet) save(log *logging.Logger) (bool, error) {
	changes := 0
	if flags.Metadata != "" {
		parts := strings.Split(flags.Metadata, ",")
		content := strings.Join(parts, "\n")
		if changed, err := updateContent(log, fleetMetadataPath, content, 0644); err != nil {
			return false, maskAny(err)
		} else if changed {
			changes++
		}
	}
	return (changes > 0), nil
}

// IsEnabled returns true if fleet should be installed on the cluster.
func (flags *Fleet) IsEnabled() bool {
	return flags.Enabled
}
