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

package systemd

import (
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/juju/errgo"
	"github.com/op/go-logging"
)

const (
	jobTimeout = time.Minute
)

type SystemdClient struct {
	Logger *logging.Logger
}

// NewSystemdClient creates a new systemd client
func NewSystemdClient(logger *logging.Logger) *SystemdClient {
	return &SystemdClient{
		Logger: logger,
	}
}

// Reload behaves as `systemctl daemon-reload`
func (sdc *SystemdClient) Reload() error {
	sdc.Logger.Debug("reloading daemon")

	conn, err := dbus.New()
	if err != nil {
		return maskAny(err)
	}

	if err := conn.Reload(); err != nil {
		sdc.Logger.Errorf("reloading daemon failed: %#v", err)
		return maskAny(err)
	}

	return nil
}

// Start behaves as `systemctl start <unit>`
func (sdc *SystemdClient) Start(unit string) error {
	sdc.Logger.Debugf("starting %s", unit)

	conn, err := dbus.New()
	if err != nil {
		return maskAny(err)
	}

	responseChan := make(chan string, 1)
	if _, err := conn.StartUnit(unit, "replace", responseChan); err != nil {
		sdc.Logger.Errorf("starting %s failed: %#v", unit, err)
		return maskAny(err)
	}

	select {
	case res := <-responseChan:
		switch res {
		case "done":
			return nil
		case "failed":
			// We need a start considered to be failed, when the unit is already running.
			return nil
		case "canceled", "timeout", "dependency", "skipped":
			return maskAny(errgo.WithCausef(nil, SystemdError, res))
		default:
			// that should never happen
			sdc.Logger.Errorf("unexpected systemd response: '%s'", res)
			return maskAny(errgo.WithCausef(nil, SystemdError, res))
		}
	case <-time.After(jobTimeout):
		return maskAny(errgo.WithCausef(nil, SystemdError, "job timeout"))
	}

	return nil
}

// Restart behaves as `systemctl restart <unit>`
func (sdc *SystemdClient) Restart(unit string) error {
	sdc.Logger.Debugf("restarting %s", unit)

	conn, err := dbus.New()
	if err != nil {
		return maskAny(err)
	}

	responseChan := make(chan string, 1)
	if _, err := conn.RestartUnit(unit, "replace", responseChan); err != nil {
		sdc.Logger.Errorf("restarting %s failed: %#v", unit, err)
		return maskAny(err)
	}

	select {
	case res := <-responseChan:
		switch res {
		case "done":
			return nil
		case "failed":
			// We need a start considered to be failed, when the unit is already running.
			return nil
		case "canceled", "timeout", "dependency", "skipped":
			return maskAny(errgo.WithCausef(nil, SystemdError, res))
		default:
			// that should never happen
			sdc.Logger.Errorf("unexpected systemd response: '%s'", res)
			return maskAny(errgo.WithCausef(nil, SystemdError, res))
		}
	case <-time.After(jobTimeout):
		return maskAny(errgo.WithCausef(nil, SystemdError, "job timeout"))
	}

	return nil
}

// Stop behaves as `systemctl stop <unit>`
func (sdc *SystemdClient) Stop(unit string) error {
	sdc.Logger.Debugf("stopping %s", unit)

	conn, err := dbus.New()
	if err != nil {
		return maskAny(err)
	}

	responseChan := make(chan string, 1)
	if _, err := conn.StopUnit(unit, "replace", responseChan); err != nil {
		sdc.Logger.Debugf("stopping %s failed: %#v", unit, err)
		return maskAny(err)
	}

	select {
	case res := <-responseChan:
		switch res {
		case "done":
			return nil
		case "canceled":
			// In case the job that is stopped is canceled (because it was running),
			// it is stopped, so all good.
			return nil
		case "timeout", "failed", "dependency", "skipped":
			return maskAny(errgo.WithCausef(nil, SystemdError, res))
		default:
			// that should never happen
			sdc.Logger.Errorf("unexpected systemd response: '%s'", res)
			return maskAny(errgo.WithCausef(nil, SystemdError, res))
		}
	case <-time.After(jobTimeout):
		return maskAny(errgo.WithCausef(nil, SystemdError, "job timeout"))
	}

	return nil
}

// Enable behaves as `systemctl enable <unit>`
func (sdc *SystemdClient) Enable(unit string) error {
	sdc.Logger.Debugf("enabling %s", unit)

	conn, err := dbus.New()
	if err != nil {
		return maskAny(err)
	}

	if _, _, err := conn.EnableUnitFiles([]string{unit}, false, false); err != nil {
		sdc.Logger.Errorf("enabling %s failed: %#v", unit, err)
		return maskAny(err)
	}

	return nil
}

// Disable behaves as `systemctl disable <unit>`
func (sdc *SystemdClient) Disable(unit string) error {
	sdc.Logger.Debugf("disabling %s", unit)

	conn, err := dbus.New()
	if err != nil {
		return maskAny(err)
	}

	if _, err := conn.DisableUnitFiles([]string{unit}, false); err != nil {
		sdc.Logger.Errorf("disabling %s failed: %#v", unit, err)
		return maskAny(err)
	}

	return nil
}

// Exists returns true if the given unit exists, false otherwise.
func (sdc *SystemdClient) Exists(unit string) (bool, error) {
	conn, err := dbus.New()
	if err != nil {
		return false, maskAny(err)
	}

	ustates, err := conn.ListUnits()
	if err != nil {
		return false, maskAny(err)
	}

	for _, ustate := range ustates {
		if ustate.Name == unit {
			return true, nil
		}
	}

	return false, nil
}

// IsActive returns true if the given unit exists and its ActiveState is 'active',
// false otherwise.
func (sdc *SystemdClient) IsActive(unit string) (bool, error) {
	conn, err := dbus.New()
	if err != nil {
		return false, maskAny(err)
	}

	ustates, err := conn.ListUnits()
	if err != nil {
		return false, maskAny(err)
	}

	for _, ustate := range ustates {
		if ustate.Name == unit {
			return ustate.ActiveState == "active", nil
		}
	}

	return false, nil
}
