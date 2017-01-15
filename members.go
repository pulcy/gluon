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
	"fmt"

	"github.com/juju/errgo"
	"github.com/pulcy/gluon/service"
	"github.com/spf13/cobra"
)

var (
	cmdMember = &cobra.Command{
		Use: "member",
		Run: showUsage,
	}
	cmdMemberList = &cobra.Command{
		Use: "list",
		Run: runMemberList,
	}

	maskAny = errgo.MaskFunc(errgo.Any)
)

func init() {
	cmdMain.AddCommand(cmdMember)
	cmdMember.AddCommand(cmdMemberList)
}

func runMemberList(cmd *cobra.Command, args []string) {
	if err := listMembers(); err != nil {
		Exitf("Failed to list members: %#v\n", err)
	}
}

func listMembers() error {
	flags := service.ServiceFlags{}
	if err := flags.SetupDefaults(log); err != nil {
		return maskAny(err)
	}
	// Get all members
	members, err := flags.GetClusterMembers(log)
	if err != nil {
		return maskAny(err)
	}
	for _, m := range members {
		fmt.Println(m.ClusterIP)
	}
	return nil
}
