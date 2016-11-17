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

package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDirectoryOf checks if the directory of the given file path exists and if not creates it.
// If such a path does exist, it checks if it is a directory, if not an error is returned.
func EnsureDirectoryOf(filePath string, perm os.FileMode) error {
	dirPath := filepath.Dir(filePath)
	return maskAny(EnsureDirectory(dirPath, perm))
}

// EnsureDirectory checks if a directory with given path exists and if not creates it.
// If such a path does exist, it checks if it is a directory, if not an error is returned.
func EnsureDirectory(dirPath string, perm os.FileMode) error {
	st, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// Path does not exist, create it
		return maskAny(os.MkdirAll(dirPath, perm))
	} else if err != nil {
		return maskAny(err)
	}
	if !st.IsDir() {
		return maskAny(fmt.Errorf("'%s' is not a directory", dirPath))
	}
	return nil
}

// UpdateFile compares the given content with the context of the file at the given filePath and
// if the content is different, the file is updated.
// If the file does not exist, it is created.
// Returns: true if the file is created or updated, false otherwise.
func UpdateFile(filePath string, content []byte, perm os.FileMode) (bool, error) {
	if err := EnsureDirectoryOf(filePath, perm); err != nil {
		return false, maskAny(err)
	}
	notFound := false
	var oldContent []byte
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		notFound = true
	} else if err != nil {
		return false, maskAny(err)
	} else {
		oldContent, err = ioutil.ReadFile(filePath)
		if os.IsNotExist(err) {
			notFound = true
		} else if err != nil {
			return false, maskAny(err)
		}
	}
	if !notFound && bytes.Equal(oldContent, content) {
		// No need to make changes, but check filemode
		if info.Mode() != perm {
			if err := os.Chmod(filePath, perm); err != nil {
				return false, maskAny(err)
			}
		}
		return false, nil
	}
	// Not found or content changed, update it
	if err := ioutil.WriteFile(filePath, content, perm); err != nil {
		return true, maskAny(err)
	}
	return true, nil
}

type KeyValuePair struct {
	Key   string
	Value string
}

// AppendEnvironmentFile ensures that all given key-value pairs are up to date in the given file.
// If the file does not exist, it is created.
// Returns: true if the file is created or updated, false otherwise.
func AppendEnvironmentFile(filePath string, kv []KeyValuePair, perm os.FileMode) (bool, error) {
	if err := EnsureDirectoryOf(filePath, perm); err != nil {
		return false, maskAny(err)
	}
	updateNeeded := false
	var oldContent []string
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		updateNeeded = true
	} else if err != nil {
		return false, maskAny(err)
	} else {
		oldContentRaw, err := ioutil.ReadFile(filePath)
		if os.IsNotExist(err) {
			updateNeeded = true
		} else if err != nil {
			return false, maskAny(err)
		} else {
			oldContent = strings.Split(string(oldContentRaw), "\n")
		}
	}

	for _, pair := range kv {
		found := false
		kvLine := fmt.Sprintf("%s=%s", pair.Key, pair.Value)
		for idx, line := range oldContent {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			if strings.TrimSpace(parts[1]) != pair.Key {
				continue
			}
			// Key found, Value equal?
			if strings.TrimSpace(parts[2]) == pair.Value {
				continue
			}
			// Update line
			oldContent[idx] = kvLine
			updateNeeded = true
			found = true
		}
		if !found {
			oldContent = append(oldContent, kvLine)
			updateNeeded = true
		}
	}

	if !updateNeeded {
		// No need to make changes, but check filemode
		if info.Mode() != perm {
			if err := os.Chmod(filePath, perm); err != nil {
				return false, maskAny(err)
			}
		}
		return false, nil
	}
	// Not found or content changed, update it
	newContent := strings.Join(oldContent, "\n")
	if err := ioutil.WriteFile(filePath, []byte(newContent), perm); err != nil {
		return true, maskAny(err)
	}
	return true, nil
}
