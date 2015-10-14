package iptables

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
	rulesTemplate        = "templates/iptables.rules.tmpl"
	rulesPath            = "/home/core/iptables.rules"
	serviceTemplate      = "templates/iptables.service.tmpl"
	serviceName          = "iptables.service"
	servicePath          = "/etc/systemd/system/" + serviceName
	netfilterTemplate    = "templates/netfilter.service.tmpl"
	netfilterServiceName = "netfilter.service"
	netfilterServicePath = "/etc/systemd/system/" + netfilterServiceName

	fileMode = os.FileMode(0755)
)

type IPTablesTopic struct {
}

func NewTopic() *IPTablesTopic {
	return &IPTablesTopic{}
}

func (t *IPTablesTopic) Name() string {
	return "iptables"
}

func (t *IPTablesTopic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *IPTablesTopic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := createRules(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := createNetfilterService(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := createIptablesService(deps, flags); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Start(netfilterServiceName); err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Start(serviceName); err != nil {
		return maskAny(err)
	}

	return nil
}

func createRules(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", rulesPath)
	memberIPs, err := flags.GetClusterMemberPrivateIPs()
	if err != nil {
		return maskAny(err)
	}
	opts := struct {
		ClusterMemberIPs []string
		DockerSubnet     string
	}{
		ClusterMemberIPs: memberIPs,
		DockerSubnet:     flags.DockerSubnet,
	}
	if err := templates.Render(rulesTemplate, rulesPath, opts, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createNetfilterService(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", netfilterServicePath)
	if err := templates.Render(netfilterTemplate, netfilterServicePath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}

func createIptablesService(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	deps.Logger.Info("creating %s", servicePath)
	if err := templates.Render(serviceTemplate, servicePath, nil, fileMode); err != nil {
		return maskAny(err)
	}

	return nil
}
