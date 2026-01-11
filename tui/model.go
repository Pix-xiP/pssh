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
	width         int
	height        int
	selectedHost  *ssh.Host // host for use in connection after selection
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.setTableSize(m.width)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
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

			return m, tea.Quit
		}

		// Handle text input and table updates
		oldSearch := m.textInput.Value()
		m.textInput, cmd = m.textInput.Update(msg)
		// Table updates should generally happen after filtering or selection changes
		m.table, _ = m.table.Update(msg)

		if m.textInput.Value() != oldSearch {
			m.filterHosts()
			m.table.SetRows(hostsToRows(m.filteredHosts))
			m.table.GotoTop()
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return "Bye!"
	}

	if m.width < 100 {
		return "Your terminal is too smol! Please resize to at least 100 columns"
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.textInput.View(),
		baseStyle.Render(m.table.View()),
		"\n Press esc to quit",
	)
}

func (m *Model) setTableSize(width int) {
	nameWidth := int(float64(width) * 0.25)
	aliasesWidth := int(float64(width) * 0.20)
	userWidth := int(float64(width) * 0.1)
	hostnameWidth := int(float64(width) * 0.35)
	portWidth := int(float64(width) * 0.08)

	m.table.SetColumns([]table.Column{
		{Title: "Name", Width: nameWidth},
		{Title: "Aliases", Width: aliasesWidth},
		{Title: "User", Width: userWidth},
		{Title: "Hostname", Width: hostnameWidth},
		{Title: "Port", Width: portWidth},
	})

	// Subtract space for text input (1 line) and footer (2 lines) and table borders (2 lines)
	m.table.SetHeight(m.height - 5)
	m.textInput.Width = width - 4
}

func initialModel(fp string) Model {
	allHosts, err := ssh.LoadSSHConfig([]string{fp})
	if err != nil {
		log.Fatal("an error occurred while loading ssh config", "err", err)
	}

	tbl := table.New(
		table.WithFocused(true),
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

	m := Model{
		hosts:     allHosts,
		textInput: txtInput,
		table:     tbl,
		height:    20,
	}

	m.setTableSize(100)
	m.table.SetRows(hostsToRows(allHosts))
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
