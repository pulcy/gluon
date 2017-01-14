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

// Vault config
type Vault struct {
	VaultImage string
}

const (
	defaultVaultImage = "pulcy/vault:0.7.2"
)

// setupDefaults fills given flags with default value
func (flags *Vault) setupDefaults(log *logging.Logger) error {
	if flags.VaultImage == "" {
		flags.VaultImage = defaultVaultImage
	}
	return nil
}

// save applicable flags to their respective files
// Returns true if anything has changed, false otherwise
func (flags *Vault) save(log *logging.Logger) (bool, error) {
	changes := 0
	return (changes > 0), nil
}
