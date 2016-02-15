package main

import (
	"strings"

	"github.com/spf13/cobra"
)

const (
	hdr = ` ____          _                 ____  _
|  _ \  _   _ | |  ___  _   _   / ___|| | _   _   ___   _ __
| |_) || | | || | / __|| | | | | |  _ | || | | | / _ \ | '_ \
|  __/ | |_| || || (__ | |_| | | |_| || || |_| || (_) || | | |
|_|     \__,_||_| \___| \__, |  \____||_| \__,_| \___/ |_| |_|
                        |___/
`
)

var (
	cmdVersion = &cobra.Command{
		Use: "version",
		Run: showVersion,
	}
)

func init() {
	cmdMain.AddCommand(cmdVersion)
}

func showVersion(cmd *cobra.Command, args []string) {
	for _, line := range strings.Split(hdr, "\n") {
		log.Info(line)
	}
	log.Info("%s %s, build %s\n", cmdMain.Use, projectVersion, projectBuild)
}
