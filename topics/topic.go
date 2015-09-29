package topics

import (
	"arvika.pulcy.com/iggi/yard/systemd"
)

type Topic interface {
	Name() string
	Setup(deps *TopicDependencies) error
}

type TopicDependencies struct {
	Systemd *systemd.SystemdClient
}
