package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
)

var (
	cmdMain = &cobra.Command{
		Use:   "yard",
		Short: "yard provisions machines to run jobs",
		Long:  "yard provisions machines to run jobs",
		Run:   showUsage,
	}
)

func main() {
	cmdMain.Execute()
}

func showUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func Exitf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	os.Exit(1)
}
