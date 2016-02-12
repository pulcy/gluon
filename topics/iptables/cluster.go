package iptables

import (
	"os/exec"
	"strings"

	"github.com/pulcy/yard/topics"
)

// UpdatePrivateCluster refresh the PRIVATECLUSTER iptables chain reflecting
// all current members of the cluster.
func UpdatePrivateCluster(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	if err := updatePrivateClusterChain(deps, flags); err != nil {
		return maskAny(err)
	}
	if err := createRules(deps, flags); err != nil {
		return maskAny(err)
	}
	return nil
}

func updatePrivateClusterChain(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	members, err := flags.GetClusterMembers()
	if err != nil {
		return maskAny(err)
	}
	iptables(deps, true, "-N", "PRIVATECLUSTER") // ignore errors
	if err := iptables(deps, false, "-F", "PRIVATECLUSTER"); err != nil {
		return maskAny(err)
	}
	for _, cm := range members {
		if err := iptables(deps, false, "-A", "PRIVATECLUSTER", "-s", cm.PrivateIP, "-j", "ACCEPT"); err != nil {
			return maskAny(err)
		}
	}
	if err := iptables(deps, false, "-A", "PRIVATECLUSTER", "-j", "DROP"); err != nil {
		return maskAny(err)
	}
	return nil
}

// iptables invokes an iptables command
func iptables(deps *topics.TopicDependencies, allowFailure bool, arg ...string) error {
	deps.Logger.Debug("iptables %s", strings.Join(arg, " "))
	cmd := exec.Command("iptables", arg...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if allowFailure {
			deps.Logger.Info("iptables failed: %s", string(out))
		} else {
			deps.Logger.Error("iptables failed: %s", string(out))
			return maskAny(err)
		}
	}
	return nil
}
