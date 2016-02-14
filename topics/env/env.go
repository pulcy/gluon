package env

import (
	"os"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/templates"
	"github.com/pulcy/gluon/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	bashrcTemplate = "templates/bashrc.tmpl"
	bashrcPath     = "/home/core/.bashrc"

	fileMode = os.FileMode(0644)
)

type EnvTopic struct {
}

func NewTopic() *EnvTopic {
	return &EnvTopic{}
}

func (t *EnvTopic) Name() string {
	return "env"
}

func (t *EnvTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *EnvTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createBashrc(deps, flags); err != nil {
		return maskAny(err)
	}

	return nil
}

func createBashrc(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", bashrcPath)
	os.Remove(bashrcPath)
	if _, err := templates.Render(bashrcTemplate, bashrcPath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
