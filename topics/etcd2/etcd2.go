package etcd2

import (
	"fmt"
	"os"
	"strings"

	"github.com/juju/errgo"

	"github.com/pulcy/gluon/topics"
	"github.com/pulcy/gluon/util"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

const (
	confTemplate = "templates/99-etcd2.conf.tmpl"
	confName     = "99-etcd2.conf"
	confPath     = "/etc/systemd/system/etcd2.service.d/" + confName
	serviceName  = "etcd2.service"

	fileMode = os.FileMode(0755)
)

type etcd2Topic struct {
}

func NewTopic() *etcd2Topic {
	return &etcd2Topic{}
}

func (t *etcd2Topic) Name() string {
	return "etcd2"
}

func (t *etcd2Topic) Defaults(flags *topics.TopicFlags) error {
	return nil
}

func (t *etcd2Topic) Setup(deps *topics.TopicDependencies, flags *topics.TopicFlags) error {
	changedConf, err := createEtcd2Conf(deps, flags)
	if err != nil {
		return maskAny(err)
	}

	if err := deps.Systemd.Reload(); err != nil {
		return maskAny(err)
	}

	isActive, err := deps.Systemd.IsActive(serviceName)
	if err != nil {
		return maskAny(err)
	}

	if !isActive || changedConf || flags.Force {
		if err := deps.Systemd.Restart(serviceName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func createEtcd2Conf(deps *topics.TopicDependencies, flags *topics.TopicFlags) (bool, error) {
	if flags.PrivateIP == "" {
		return false, maskAny(fmt.Errorf("PrivateIP empty"))
	}
	deps.Logger.Info("creating %s", confPath)

	members, err := flags.GetClusterMembers()
	if err != nil {
		deps.Logger.Warning("GetClusterMembers failed: %v", err)
	}
	clusterItems := []string{}
	name := ""
	for _, cm := range members {
		clusterItems = append(clusterItems,
			fmt.Sprintf("%s=http://%s:2380", cm.MachineID, cm.PrivateIP))
		if cm.PrivateIP == flags.PrivateIP {
			name = cm.MachineID
		}
	}

	lines := []string{
		"[Service]",
		"Environment=ETCD_LISTEN_PEER_URLS=" + fmt.Sprintf("http://%s:2380", flags.PrivateIP),
		"Environment=ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379,http://0.0.0.0:4001",
		"Environment=ETCD_INITIAL_CLUSTER=" + strings.Join(clusterItems, ","),
		"Environment=ETCD_INITIAL_CLUSTER_STATE=new",
		"Environment=ETCD_INITIAL_ADVERTISE_PEER_URLS=" + fmt.Sprintf("http://%s:2380", flags.PrivateIP),
		"Environment=ETCD_ADVERTISE_CLIENT_URLS=" + fmt.Sprintf("http://%s:2379,http://%s:4001", flags.PrivateIP, flags.PrivateIP),
	}
	if name != "" {
		lines = append(lines, "Environment=ETCD_NAME="+name)
	}

	changed, err := util.UpdateFile(confPath, []byte(strings.Join(lines, "\n")), fileMode)
	return changed, maskAny(err)
}
