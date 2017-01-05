// Taken from https://gist.github.com/tam7t/1b45125ae4de13b3fc6fd0455954c08e

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
)

func main() {
	err := realMain()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

func realMain() error {
	if len(os.Args) != 4 {
		// Ensure the empty input case is handled correctly
		return nil
	}

	// certdump <filepath> <owner> <data>
	path := os.Args[1]
	owner := os.Args[2]
	data := os.Args[3]

	err := ioutil.WriteFile(path, []byte(data), 0700)
	if err != nil {
		return err
	}
	u, err := user.Lookup(owner)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid := os.Getgid()
	err = os.Chmod(path, 0660)
	if err != nil {
		return err
	}
	err = os.Chown(path, uid, gid)
	if err != nil {
		return err
	}

	return nil
}
