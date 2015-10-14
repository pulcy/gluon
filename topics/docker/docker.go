package docker

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
	serviceTemplate = "templates/docker.service.tmpl"
	serviceName     = "docker.service"
	servicePath     = "/etc/systemd/system/" + serviceName

	fileMode = os.FileMode(0755)
)

type DockerTopic struct {
}

func NewTopic() *DockerTopic {
	return &DockerTopic{}
}

func (t *DockerTopic) Name() string {
	return "docker"
}

func (t *DockerTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *DockerTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createDockerService(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Start(serviceName); err != nil {
		return maskAny(err)
	}

	return nil
}

func createDockerService(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", servicePath)
	opts := struct {
		DockerIP string
	}{
		DockerIP: flags.DockerIP,
	}
	if err := templates.Render(serviceTemplate, servicePath, opts, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
