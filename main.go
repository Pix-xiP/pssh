package main

import (
	"context"
	"flag"

	"github.com/charmbracelet/log"
	"github.com/pix-xip/go-command"
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

	selectedHost := tui.SelectHost(fp)
	if selectedHost != nil {
		tmpl := "ssh {{.Name}}" // TODO: make this configurable
		// IMPROV: could also add a {{.Comamand}} from CLI to run commnds via connection?

		err := selectedHost.RunCmdTmpl(tmpl)
		if err != nil {
			return err
		}
	}

	return nil
}
