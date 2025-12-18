// Package tui provides the TUI interface for PSSH
package tui

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pix-xip/pssh/ssh"
	"github.com/sahilm/fuzzy"
)

var baseStyle = lipgloss.NewStyle().BorderForeground(lipgloss.Color("240"))

type Model struct {
	hosts         []*ssh.Host
	filteredHosts []*ssh.Host
	textInput     textinput.Model
	table         table.Model
	quitting      bool
	selectedHost  *ssh.Host // host for use in connection after selection
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.textInput.Value() != "" {
				m.textInput.SetValue("")
				m.filterHosts()
				m.table.SetRows(hostsToRows(m.filteredHosts))
			} else {
				m.quitting = true
				return m, tea.Quit
			}
		case "enter":
			selectedRow := m.table.SelectedRow()
			if len(selectedRow) > 0 {
				// Find the actual host object from filteredHosts using the selected name
				for _, h := range m.filteredHosts {
					if h.Name == selectedRow[0] {
						m.selectedHost = h
						break
					}
				}
			}

			return m, tea.Quit // Quit the TUI to run the external command
		}

		// Handle text input and table updates
		oldSearch := m.textInput.Value()
		m.textInput, cmd = m.textInput.Update(msg)
		// Table updates should generally happen after filtering or selection changes
		m.table, _ = m.table.Update(msg)

		if m.textInput.Value() != oldSearch {
			m.filterHosts()
			m.table.SetRows(hostsToRows(m.filteredHosts))
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return "Bye!"
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.textInput.View(),
		baseStyle.Render(m.table.View()),
		"\n Press q to quit",
	)
}

func initialModel(fp string) Model {
	paths := []string{"/etc/ssh/ssh_config", "~/.ssh/config", fp}

	allHosts, err := ssh.LoadSSHConfig(paths)
	if err != nil {
		log.Fatal("an error occurred while loading ssh config", "err", err)
	}

	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Aliases", Width: 15},
		{Title: "User", Width: 10},
		{Title: "Hostname", Width: 25},
		{Title: "Port", Width: 7},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(hostsToRows(allHosts)),
		table.WithFocused(true),
		table.WithHeight(9),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	tbl.SetStyles(s)

	txtInput := textinput.New()
	txtInput.Placeholder = "Search SSH hosts..."
	txtInput.Focus()
	txtInput.CharLimit = 200
	txtInput.Width = 40

	m := Model{
		hosts:     allHosts,
		textInput: txtInput,
		table:     tbl,
	}

	m.filterHosts()

	return m
}

func (m *Model) filterHosts() {
	searchTerm := m.textInput.Value()
	if searchTerm == "" {
		m.filteredHosts = m.hosts
		return
	}

	targets := make([]string, 0, len(m.hosts))
	for _, host := range m.hosts {
		targets = append(targets, fmt.Sprintf("%s %s %s %s %s",
			host.Name,
			host.Aliases,
			host.User,
			host.Hostname,
			host.Port,
		))
	}

	ranks := fuzzy.Find(searchTerm, targets)

	newFiltered := make([]*ssh.Host, 0, len(m.hosts))
	for _, rank := range ranks {
		newFiltered = append(newFiltered, m.hosts[rank.Index])
	}

	m.filteredHosts = newFiltered
}

func hostsToRows(hosts []*ssh.Host) []table.Row {
	rows := make([]table.Row, 0, len(hosts))
	for _, host := range hosts {
		rows = append(rows, table.Row{
			host.Name,
			host.Aliases,
			host.User,
			host.Hostname,
			host.Port,
		})
	}

	return rows
}
