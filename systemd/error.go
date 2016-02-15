package systemd

import (
	"github.com/juju/errgo"
)

var (
	SystemdError = errgo.New("systemd error")
	maskAny      = errgo.MaskFunc(errgo.Any)
)
