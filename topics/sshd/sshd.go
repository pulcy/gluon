package sshd

import (
	"os"

	"github.com/juju/errgo"

	"github.com/pulcy/yard/templates"
	"github.com/pulcy/yard/topics"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate = "templates/sshd_config.tmpl"
	confName     = "sshd_config"
	confPath     = "/etc/ssh/" + confName

	fileMode = os.FileMode(0600)
)

type SshdTopic struct {
}

func NewTopic() *SshdTopic {
	return &SshdTopic{}
}

func (t *SshdTopic) Name() string {
	return "sshd"
}

func (t *SshdTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *SshdTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createSshdConfig(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	return nil
}

func createSshdConfig(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", confPath)
	os.Remove(confPath)
	if err := templates.Render(confTemplate, confPath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
