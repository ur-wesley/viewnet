package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModularUIModel struct {
	*UIModel

	header   *HeaderComponent
	search   *SearchComponent
	progress *ProgressComponent
	stats    *StatsComponent
	summary  *SummaryComponent
	table    *TableComponent
	help     *HelpComponent
}

func NewModularUI(targetSubnet string, startPort, endPort, timeout int, focusedSearch bool, initialSearch string, customPorts []int, ipsOnly bool) *ModularUIModel {
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	ti := textinput.New()
	ti.Placeholder = "Search by IP, vendor, MAC, hostname, or service..."
	ti.CharLimit = 100
	ti.Width = 50
	if initialSearch != "" {
		ti.SetValue(initialSearch)
	}
	baseModel := &UIModel{
		state:          stateScanning,
		progress:       p,
		spinner:        s,
		searchInput:    ti,
		targetSubnet:   targetSubnet,
		startPort:      startPort,
		endPort:        endPort,
		timeout:        timeout,
		customPorts:    customPorts,
		ipsOnly:        ipsOnly,
		viewHeight:     20,
		windowWidth:    80,
		windowHeight:   24,
		searchOnlyMode: focusedSearch,
	}

	return &ModularUIModel{
		UIModel:  baseModel,
		header:   NewHeaderComponent(),
		search:   NewSearchComponent(),
		progress: NewProgressComponent(),
		stats:    NewStatsComponent(),
		summary:  NewSummaryComponent(),
		table:    NewTableComponent(),
		help:     NewHelpComponent(),
	}
}

func (m *ModularUIModel) Init() tea.Cmd {
	StartTUIScan(m.targetSubnet, m.startPort, m.endPort, time.Duration(m.timeout)*time.Millisecond, m.customPorts, m.ipsOnly)
	return tea.Batch(
		m.spinner.Tick,
		m.UIModel.progress.Init(),
		pollForUpdates(),
		animateProgress(),
		tea.WindowSize(),
	)
}

func (m *ModularUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		reservedHeight := m.header.Height() + m.help.Height() + 2

		reservedHeight += m.stats.Height()

		if m.state == stateScanning {
			reservedHeight += m.progress.Height()
		} else {
			reservedHeight += m.summary.Height() + m.search.Height()
		}

		m.viewHeight = max(msg.Height-reservedHeight, 5)
		m.UIModel.progress.Width = max(msg.Width-8, 20)

		m.searchInput.Width = max(msg.Width-15, 20)
		m.adjustScrollBounds()
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		if !m.searchFocused {
			switch msg.String() {
			case "q":
				m.quitting = true
				return m, tea.Quit
			case "r":
				if m.state == stateComplete {
					m.state = stateScanning
					m.results = []*HostInfo{}
					m.filteredResults = []*HostInfo{}
					m.scrollOffset = 0
					m.scanEndTime = time.Time{}
					StartTUIScan(m.targetSubnet, m.startPort, m.endPort, time.Duration(m.timeout)*time.Millisecond, m.customPorts, m.ipsOnly)
					return m, tea.Batch(pollForUpdates(), m.spinner.Tick)
				}
				return m, nil
			case "/", "f":
				if m.state == stateComplete {
					m.searchFocused = true
					m.searchInput.Focus()
					return m, nil
				}
			case "ctrl+f":
				if m.state == stateComplete {
					m.searchOnlyMode = !m.searchOnlyMode
					filterResults(m.UIModel)
					return m, nil
				}
			case "esc":
				m.searchInput.SetValue("")
				m.searchOnlyMode = false
				filterResults(m.UIModel)
				return m, nil
			}
		}

		if cmd := m.search.Update(msg, m.UIModel); cmd != nil {
			cmds = append(cmds, cmd)
		}
		if cmd := m.table.Update(msg, m.UIModel); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case pollMsg:
		progress := GetScanProgress()
		m.scanInfo = progress
		if progress.TotalHosts > 0 {
			percentage := float64(progress.HostsScanned) / float64(progress.TotalHosts)
			if percentage > 1.0 {
				percentage = 1.0
			}
			if percentage < 0.0 {
				percentage = 0.0
			}
			m.UIModel.progress.SetPercent(percentage)
		} else {
			m.UIModel.progress.SetPercent(0.0)
		}
		m.results = GetScanResults()
		filterResults(m.UIModel)
		m.adjustScrollBounds()
		if IsScanComplete() && progress.HostsScanned >= progress.TotalHosts {
			m.state = stateComplete
			if m.scanEndTime.IsZero() {
				m.scanEndTime = time.Now()
			}
			return m, nil
		}

		cmds = append(cmds, pollForUpdates())

	case scanErrorMsg:
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit
	}

	if cmd := m.progress.Update(msg, m.UIModel); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ModularUIModel) View() string {
	if m.quitting {
		if m.err != nil {
			return fmt.Sprintf("❌ Error: %v\n", m.err)
		}
		return "👋 Goodbye!\n"
	}
	var sections []string

	sections = append(sections, m.header.View(m.UIModel))

	if m.state == stateScanning {
		sections = append(sections, m.progress.View(m.UIModel))
	}

	sections = append(sections, m.stats.View(m.UIModel))

	if m.state == stateComplete {
		if summary := m.summary.View(m.UIModel); summary != "" {
			sections = append(sections, summary)
		}
		sections = append(sections, m.search.View(m.UIModel))
	}

	sections = append(sections, m.table.View(m.UIModel))
	sections = append(sections, m.help.View(m.UIModel))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *ModularUIModel) adjustScrollBounds() {
	resultsToShow := m.filteredResults
	if len(resultsToShow) == 0 {
		resultsToShow = m.results
	}

	if len(resultsToShow) == 0 {
		m.scrollOffset = 0
		return
	}

	minScroll := 0
	maxScroll := max(len(resultsToShow)-m.viewHeight, 0)

	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	if m.scrollOffset < minScroll {
		m.scrollOffset = minScroll
	}
}
