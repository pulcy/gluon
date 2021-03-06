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
	"time"

	"github.com/spf13/cobra"

	"github.com/pulcy/gluon/update"
)

const (
	defaultMachineDelay = time.Second * 30
)

var (
	cmdUpdate = &cobra.Command{
		Use: "update",
		Run: runUpdate,
	}
	updateFlags = &update.UpdateFlags{}
)

func init() {
	cmdUpdate.Flags().DurationVar(&updateFlags.MachineDelay, "machine-delay", defaultMachineDelay, "Time between updating 2 machines")
	cmdUpdate.Flags().DurationVar(&updateFlags.RebootExpired, "reboot-expired", 0, "Duration until a reboot is considered failed")
	cmdUpdate.Flags().BoolVar(&updateFlags.Reboot, "reboot", false, "If set, reboot machines after update")
	cmdUpdate.Flags().BoolVar(&updateFlags.AskConfirmation, "confirm", false, "If set, confirmation is needed before continuing with next machine")

	cmdMain.AddCommand(cmdUpdate)
}

func runUpdate(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Exitf("Gluon-image argument needed\n")
	}
	updateFlags.GluonImage = args[0]
	if err := updateFlags.SetupDefaults(log); err != nil {
		Exitf("SetupDefaults failed: %#v\n", err)
	}

	if err := update.UpdateAllMachines(updateFlags, log); err != nil {
		Exitf("Update failed: %#v", err)
	}

	log.Info("Done")
}
