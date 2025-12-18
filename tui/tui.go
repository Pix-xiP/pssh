package tui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pix-xip/pssh/ssh"
)

func SelectHost(fp string) *ssh.Host {
	m := initialModel(fp)
	p := tea.NewProgram(m)

	final, err := p.Run()
	if err != nil {
		log.Fatal("error running program", "err", err)
	}

	return final.(Model).selectedHost
}
