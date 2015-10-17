package main

import (
	"fmt"
	"github.com/spf13/cobra"
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
	fmt.Printf("%s %s, build %s\n", cmdMain.Use, projectVersion, projectBuild)
}
