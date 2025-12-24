package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pix-xip/go-command"
	"github.com/pix-xip/pssh/ssh"
	"github.com/pix-xip/pssh/tui"
)

const (
	defaultSSHConfig = "~/.ssh/config"
)

var Version string

func main() {
	r := command.Root().Help("pssh is a TUI ssh manager").
		Flags(func(fs *flag.FlagSet) {
			fs.String("ssh-config", defaultSSHConfig, "path to ssh config file")
			fs.Bool("loop", false, "loop until SSH connection successfully connects")
		})

	r.Action(RunTui)
	r.SubCommand("version").Action(func(_ context.Context, _ *flag.FlagSet, _ []string) error {
		log.Infof("pssh version %s", Version)
		return nil
	}).Help("display the version")

	if err := r.Execute(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func RunTui(_ context.Context, fs *flag.FlagSet, _ []string) error {
	fp := command.Lookup[string](fs, "ssh-config")
	loop := command.Lookup[bool](fs, "loop")

	selectedHost := tui.SelectHost(fp)
	if selectedHost != nil {
		return runSSH(selectedHost, loop)
	}

	log.SetTimeFormat(time.Kitchen)

	return nil
}

func runSSH(host *ssh.Host, loop bool) error {
	tmpl := "ssh {{.Name}}" // TODO: make this configurable
	// IMPROV: could also add a {{.Comamand}} from CLI to run commnds via connection?

	if !loop {
		return host.RunCmdTmpl(tmpl)
	}

	for {
		err := host.RunCmdTmpl(tmpl)
		if err == nil {
			log.Info("Connection closed.")
			break
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// This is an expected error from ssh, so we can retry.
			log.Infof("Connection failed, retrying in 2 seconds. Press Ctrl+C to cancel.")
			time.Sleep(2 * time.Second)
			continue
		}

		// This is an unexpected error, so we should stop.
		return fmt.Errorf("ssh command failed unexpectedly: %w", err)
	}

	return nil
}
