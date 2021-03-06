package update

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/juju/errgo"
	logging "github.com/op/go-logging"
	"github.com/pulcy/gluon/service"
)

type UpdateFlags struct {
	service.ServiceFlags
	MachineDelay    time.Duration
	RebootExpired   time.Duration
	UserName        string
	Reboot          bool
	AskConfirmation bool
}

func (flags *UpdateFlags) SetupDefaults(log *logging.Logger) error {
	if err := flags.ServiceFlags.SetupDefaults(log); err != nil {
		return maskAny(err)
	}
	if flags.MachineDelay == 0 {
		flags.MachineDelay = time.Second * 30
	}
	if flags.RebootExpired == 0 {
		flags.RebootExpired = time.Minute * 2
	}
	if flags.UserName == "" {
		flags.UserName = "core"
	}
	return nil
}

func UpdateAllMachines(flags *UpdateFlags, log *logging.Logger) error {
	// Get all members
	members, err := flags.GetClusterMembers(log)
	if err != nil {
		return maskAny(err)
	}

	// Pull image on all machines
	log.Infof("Pulling gluon image on %d machines", len(members))
	var pullGroup errgroup.Group
	for _, m := range members {
		m := m
		pullGroup.Go(func() error {
			return maskAny(pullImage(m, *flags, log))
		})
	}
	if err := pullGroup.Wait(); err != nil {
		return maskAny(err)
	}

	// Update all machines, one at a time
	for index, m := range members {
		if index > 0 {
			log.Infof("Waiting %s...", flags.MachineDelay)
			time.Sleep(flags.MachineDelay)
		}
		if err := updateMachine(m, *flags, log); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

func pullImage(member service.ClusterMember, flags UpdateFlags, log *logging.Logger) error {
	cmd := fmt.Sprintf("docker pull %s", flags.GluonImage)
	if _, err := runRemoteCommand(member, flags.UserName, log, cmd, "", false); err != nil {
		return maskAny(err)
	}
	return nil
}

func updateMachine(member service.ClusterMember, flags UpdateFlags, log *logging.Logger) error {
	askConfirmation := flags.AskConfirmation
	log.Infof("Updating %s...", member.ClusterIP)

	// Extract gluon binary
	cmd := fmt.Sprintf("docker run --rm -v /home/core/bin/:/destination/ %s", flags.GluonImage)
	if _, err := runRemoteCommand(member, flags.UserName, log, cmd, "", false); err != nil {
		return maskAny(err)
	}

	// Update image version on disk
	if _, err := runRemoteCommand(member, flags.UserName, log, "sudo tee /etc/pulcy/gluon-image", flags.GluonImage, false); err != nil {
		return maskAny(err)
	}

	// Setup new gluon version
	if _, err := runRemoteCommand(member, flags.UserName, log, "sudo systemctl restart gluon", "", false); err != nil {
		return maskAny(err)
	}

	// Reboot if needed
	if flags.Reboot {
		log.Infof("Rebooting %s...", member.ClusterIP)
		runRemoteCommand(member, flags.UserName, log, "sudo reboot -f", "", true)
		time.Sleep(time.Second * 15)
		if err := waitUntilMachineUp(member, flags, log); err != nil {
			return maskAny(err)
		}
		if !member.EtcdProxy {
			log.Warningf("Core machine %s is back up, check services", member.ClusterIP)
			askConfirmation = true
		} else {
			log.Infof("Machine %s is back up", member.ClusterIP)
		}
	}

	if askConfirmation {
		confirm("Can we continue?")
	}

	return nil
}

func runRemoteCommand(member service.ClusterMember, userName string, log *logging.Logger, command, stdin string, quiet bool) (string, error) {
	hostAddress := member.ClusterIP
	cmd := exec.Command("ssh", "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", userName+"@"+hostAddress, command)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	if err := cmd.Run(); err != nil {
		if !quiet {
			log.Errorf("SSH failed: %s %s", cmd.Path, strings.Join(cmd.Args, " "))
		}
		return "", errgo.NoteMask(err, stdErr.String())
	}

	out := stdOut.String()
	out = strings.TrimSuffix(out, "\n")
	return out, nil
}

func waitUntilMachineUp(member service.ClusterMember, flags UpdateFlags, log *logging.Logger) error {
	start := time.Now()
	for {
		if _, err := runRemoteCommand(member, flags.UserName, log, "cat /etc/machine-id", "", true); err == nil {
			return nil
		}
		if time.Since(start) > flags.RebootExpired {
			return maskAny(fmt.Errorf("Machine %s took too long to reboot", member.ClusterIP))
		}
		time.Sleep(time.Second * 2)
	}
}

func confirm(question string) error {
	prefix := ""
	for {
		var line string
		fmt.Printf("%s%s [yes|no]", prefix, question)
		bufStdin := bufio.NewReader(os.Stdin)
		lineRaw, _, err := bufStdin.ReadLine()
		if err != nil {
			return err
		}
		line = string(lineRaw)

		switch line {
		case "yes", "y":
			return nil
		}
		prefix = "Please enter 'yes' to confirm."
	}
}
