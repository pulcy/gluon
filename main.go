package main

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/spf13/cobra"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
)

var (
	cmdMain = &cobra.Command{
		Use:   "gluon",
		Short: "gluon provisions machines to run jobs",
		Long:  "gluon provisions machines to run jobs",
		Run:   showUsage,
	}
	log = logging.MustGetLogger(cmdMain.Use)
)

func init() {
	logging.SetFormatter(logging.MustStringFormatter("[%{level:-5s}] %{message}"))
}

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

func def(envKey, defaultValue string) string {
	s := os.Getenv(envKey)
	if s == "" {
		s = defaultValue
	}
	return s
}
