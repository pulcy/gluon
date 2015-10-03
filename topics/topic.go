package topics

import (
	"github.com/op/go-logging"

	"arvika.pulcy.com/pulcy/yard/systemd"
)

type Topic interface {
	Name() string
	Defaults(flags *TopicFlags) error
	Setup(deps *TopicDependencies, flags *TopicFlags) error
}

type TopicDependencies struct {
	Systemd *systemd.SystemdClient
	Logger  *logging.Logger
}

type TopicFlags struct {
	// Stunnel
	StunnelPemPassphrase string

	// Weave
	WeavePassword string
}
