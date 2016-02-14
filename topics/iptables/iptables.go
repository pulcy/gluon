package iptables

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
	changedRules, err := createRules(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedNetfilterService, err := createNetfilterService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	changedIptableService, err := createIptablesService(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if flags.Force || changedRules || changedNetfilterService || changedIptableService {
		if err := deps.Systemd.Reload(); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(netfilterServiceName); err != nil {
			return maskAny(err)
		}
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createRules(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	deps.Logger.Info("creating %s", rulesPath)
	members, err := flags.GetClusterMembers(deps.Logger)
	if err != nil {
		return false, maskAny(err)
	}
	opts := struct {
		ClusterMemberIPs     []string
		DockerSubnet         string
		PrivateClusterDevice string
	}{
		ClusterMemberIPs:     []string{},
		DockerSubnet:         flags.DockerSubnet,
		PrivateClusterDevice: flags.PrivateClusterDevice,
	}
	for _, cm := range members {
		opts.ClusterMemberIPs = append(opts.ClusterMemberIPs, cm.PrivateIP)
	}
	changed, err := templates.Render(rulesTemplate, rulesPath, opts, fileMode)
	return changed, maskAny(err)
}

func createNetfilterService(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	deps.Logger.Info("creating %s", netfilterServicePath)
	changed, err := templates.Render(netfilterTemplate, netfilterServicePath, nil, fileMode)
	return changed, maskAny(err)
}

func createIptablesService(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	deps.Logger.Info("creating %s", servicePath)
	changed, err := templates.Render(serviceTemplate, servicePath, nil, fileMode)
	return changed, maskAny(err)
}
