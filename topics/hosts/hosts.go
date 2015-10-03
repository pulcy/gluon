package hosts

import (
	"os"

	"github.com/juju/errgo"

	"arvika.pulcy.com/pulcy/yard/templates"
	"arvika.pulcy.com/pulcy/yard/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	hostsTemplate = "templates/hosts.tmpl"
	hostsPath     = "/etc/hosts"

	fileMode = os.FileMode(0664)
)

type HostsTopic struct {
}

func NewTopic() *HostsTopic {
	return &HostsTopic{}
}

func (t *HostsTopic) Name() string {
	return "hosts"
}

func (t *HostsTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *HostsTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createHosts(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createHosts(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", hostsPath)
	if err := templates.Render(hostsTemplate, hostsPath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
