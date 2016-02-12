package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
