// Package ssh provides simple wrapper around ssh_config and executing ssh session for TUI
package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
)

type Host struct {
	// Name is the primary pattern used to match this host entry.
	Name string
	// Aliases are other patterns that match this host entry.
	Aliases string
	// User is the username for the SSH connection.
	User string
	// Hostname is the actual remote hostname to connect to.
	Hostname string
	// Port is the port number for the SSH connection.
	Port string
	// ProxyCommand is the command to use to connect to the server.
	ProxyCommand string

	// original is a reference to the ssh_config.Host for other properties
	original *ssh_config.Host
}

func NewHost(host *ssh_config.Host) *Host {
	name := ""
	if len(host.Patterns) > 0 {
		name = host.Patterns[0].String()
	}

	aliases := ""

	if len(host.Patterns) > 1 {
		var aliasPatterns []string
		for _, p := range host.Patterns[1:] {
			aliasPatterns = append(aliasPatterns, p.String())
		}

		aliases = fmt.Sprintf("(%s)", joinStrings(aliasPatterns))
	}

	hostname := getOptVal(host, "hostname")
	if hostname == "" && len(host.Patterns) > 0 {
		hostname = host.Patterns[0].String()
	}

	return &Host{
		Name:         name,
		Aliases:      aliases,
		User:         getOptVal(host, "user"),
		Hostname:     hostname,
		Port:         getOptVal(host, "port"),
		ProxyCommand: getOptVal(host, "proxycommand"),
		original:     host,
	}
}

func joinStrings(ss []string) string {
	b := strings.Builder{}
	b.Grow(len(ss))

	for i, s := range ss {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(s)
	}

	return b.String()
}

func getOptVal(host *ssh_config.Host, opt string) string {
	for _, node := range host.Nodes {
		if kv, ok := node.(*ssh_config.KV); ok {
			if strings.EqualFold(kv.Key, opt) {
				return kv.Value
			}
		}
	}

	return ""
}

func loadSSHConfig(path, home string) ([]*ssh_config.Host, error) {
	var fp string

	if strings.HasPrefix(path, "~/") {
		fp = filepath.Join(home, path[2:])
	} else {
		fp = path
	}

	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		if path == "/etc/ssh/ssh_config" && os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("could not open ssh config file %s: %w", fp, err)
	}

	defer func() { _ = f.Close() }()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("could not decode ssh config file %s: %w", fp, err)
	}

	hosts := make([]*ssh_config.Host, 0, len(cfg.Hosts))

	for _, h := range cfg.Hosts {
		if len(h.Patterns) > 0 && h.Patterns[0].String() == "*" {
			for _, node := range h.Nodes {
				if after, ok := strings.CutPrefix(strings.ToLower(node.String()), "include "); ok {
					incPath := strings.TrimSpace(after)

					includedHosts, err := loadSSHConfig(incPath, home)
					if err != nil {
						return nil, err
					}

					hosts = append(hosts, includedHosts...)
				}
			}
		}

		if len(h.Patterns) == 0 || h.Patterns[0].String() == "*" {
			continue
		}

		hosts = append(hosts, h)
	}

	return hosts, nil
}

func LoadSSHConfig(paths []string) ([]*Host, error) {
	var allSSHHosts []*ssh_config.Host

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	for _, p := range paths {
		hosts, err := loadSSHConfig(p, home)
		if err != nil {
			return nil, err
		}

		allSSHHosts = append(allSSHHosts, hosts...)
	}

	allHosts := make([]*Host, 0, len(allSSHHosts))

	for _, h := range allSSHHosts {
		allHosts = append(allHosts, NewHost(h))
	}

	// TODO: Figure out if we want to group AND include all hosts with the same hostname
	// or just the grouped one.
	// Group hosts by hostname
	groupedHosts := make(map[string][]*Host)
	for _, h := range allHosts {
		groupedHosts[h.Hostname] = append(groupedHosts[h.Hostname], h)
	}

	hosts := make([]*Host, 0, len(groupedHosts))

	for _, group := range groupedHosts {
		if len(group) == 0 {
			continue
		}

		if len(group) == 1 {
			hosts = append(hosts, group[0])
			continue
		}

		// TODO: This assumes the first one is the most important
		// maybe we should use the one with the most config options?
		primary := group[0]

		var aliases []string
		for _, h := range group[1:] {
			aliases = append(aliases, h.Name)
		}

		// also add any existing aliases
		for _, h := range group {
			if h.Aliases != "" {
				aliases = append(aliases, h.Aliases)
			}
		}

		primary.Aliases = fmt.Sprintf("(%s)", joinStrings(aliases))
		hosts = append(hosts, primary)
	}

	return hosts, nil
}

func (h *Host) RunCmdTmpl(tmplstr string) error {
	tmpl, err := template.New("command").Parse(tmplstr)
	if err != nil {
		return fmt.Errorf("could not parse command template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, h); err != nil {
		return fmt.Errorf("error executing command template: %w", err)
	}

	commandLine := buf.String()
	fmt.Printf("Running command: %s\n", commandLine)

	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return errors.New("command is empty")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
