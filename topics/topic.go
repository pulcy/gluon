package topics

import (
	"arvika.pulcy.com/iggi/yard/systemd"
	"github.com/op/go-logging"
)

type Topic interface {
	Name() string
	Setup(deps *TopicDependencies) error
}

type TopicDependencies struct {
	Systemd *systemd.SystemdClient
	Logger  *logging.Logger
}
