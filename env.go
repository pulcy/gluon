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

package main

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

const (
	gluonEnvPath = "/etc/pulcy/gluon.env"
)

var (
	loadEnvOnce sync.Once
)

// LoadEnv loads the environment file /etc/pulcy/gluon.env into the environment of this process.
func LoadEnv() {
	loadEnvOnce.Do(func() {
		if content, err := ioutil.ReadFile(gluonEnvPath); os.IsNotExist(err) {
			return
		} else if err != nil {
			Exitf("Failed to read %s: %#v", gluonEnvPath, err)
		} else {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				// Trim comments
				parts := strings.SplitN(line, "#", 2)
				line = strings.TrimSpace(parts[0])
				// Split in key=value
				parts = strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					os.Setenv(key, value)
				}
			}
		}
	})
}
