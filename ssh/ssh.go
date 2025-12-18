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

func LoadSSHConfig(paths []string) ([]*Host, error) {
	var hosts []*Host

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	for _, p := range paths {
		var fp string
		if strings.HasPrefix(p, "~/") {
			fp = filepath.Join(home, p[2:])
		} else {
			fp = p
		}

		f, err := os.Open(filepath.Clean(fp))
		if err != nil {
			// ignore the system wide fella if its missing
			if p == "/etc/ssh/ssh_config" && os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("could not open ssh config file %s: %w", fp, err)
		}

		defer func() { _ = f.Close() }()

		cfg, err := ssh_config.Decode(f)
		if err != nil {
			return nil, fmt.Errorf("could not decode ssh config file %s: %w", fp, err)
		}

		for _, h := range cfg.Hosts {
			hosts = append(hosts, NewHost(h))
		}
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
