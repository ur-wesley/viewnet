package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TUIComponent interface {
	Update(msg tea.Msg, model *UIModel) tea.Cmd
	View(model *UIModel) string
	Height() int
}

type HeaderComponent struct{}

func NewHeaderComponent() *HeaderComponent {
	return &HeaderComponent{}
}

func (h *HeaderComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	return nil
}

func (h *HeaderComponent) View(model *UIModel) string {
	var content []string

	title := titleStyle.Render("🔍 ViewNet - Network Discovery")
	content = append(content, title)

	var header string
	if model.ipsOnly {
		header = headerStyle.Render(fmt.Sprintf(
			"Target: %s | Mode: IP Discovery Only | Timeout: %dms",
			model.targetSubnet, model.timeout,
		))
	} else if len(model.customPorts) > 0 {
		portsStr := ""
		if len(model.customPorts) <= 10 {
			portStrs := make([]string, len(model.customPorts))
			for i, port := range model.customPorts {
				portStrs[i] = fmt.Sprintf("%d", port)
			}
			portsStr = strings.Join(portStrs, ",")
		} else {
			portsStr = fmt.Sprintf("%d common ports", len(model.customPorts))
		}
		header = headerStyle.Render(fmt.Sprintf(
			"Target: %s | Ports: %s | Timeout: %dms",
			model.targetSubnet, portsStr, model.timeout,
		))
	} else {
		header = headerStyle.Render(fmt.Sprintf(
			"Target: %s | Ports: %d-%d | Timeout: %dms",
			model.targetSubnet, model.startPort, model.endPort, model.timeout,
		))
	}
	content = append(content, header)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (h *HeaderComponent) Height() int {
	return 2
}

type SearchComponent struct{}

func NewSearchComponent() *SearchComponent {
	return &SearchComponent{}
}

func (s *SearchComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if model.searchFocused {
			switch msg.String() {
			case "esc":
				model.searchFocused = false
				model.searchInput.Blur()
				return nil
			case "enter":
				model.searchFocused = false
				model.searchInput.Blur()
				filterResults(model)
				return nil
			case "ctrl+f", "alt+f":
				model.searchOnlyMode = !model.searchOnlyMode
				filterResults(model)
				return nil
			default:
				var cmd tea.Cmd
				model.searchInput, cmd = model.searchInput.Update(msg)
				filterResults(model)
				return cmd
			}
		}
	}
	return nil
}

func (s *SearchComponent) View(model *UIModel) string {
	searchLabel := "🔍 Search: "
	if model.searchFocused {
		modeIndicator := ""
		if model.searchOnlyMode {
			modeIndicator = " [FOCUSED] "
		}
		searchLabel = fmt.Sprintf("🔍 Search%s (ESC to exit, ENTER to confirm, Ctrl+F to toggle focus): ", modeIndicator)
	}
	searchView := searchLabel + model.searchInput.View()
	if !model.searchFocused && model.searchInput.Value() == "" {
		if model.searchOnlyMode {
			searchView += " [FOCUSED MODE] (Press '/' to search by IP or vendor only)"
		} else {
			searchView += " (Press '/' to search all fields, Ctrl+F for focused mode, or use -focused flag)"
		}
	} else if !model.searchFocused && model.searchInput.Value() != "" {
		if model.searchOnlyMode {
			searchView += " [FOCUSED MODE - showing only IP/vendor matches]"
		}
	}
	return searchView
}

func (s *SearchComponent) Height() int {
	return 1
}

type ProgressComponent struct{}

func NewProgressComponent() *ProgressComponent {
	return &ProgressComponent{}
}

func (p *ProgressComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	switch msg := msg.(type) {
	case progress.FrameMsg:
		progressModel, cmd := model.progress.Update(msg)
		model.progress = progressModel.(progress.Model)
		return tea.Batch(cmd, animateProgress())
	case spinner.TickMsg:
		var cmd tea.Cmd
		model.spinner, cmd = model.spinner.Update(msg)
		return cmd
	}
	return nil
}

func (p *ProgressComponent) View(model *UIModel) string {
	if model.state != stateScanning {
		return ""
	}

	progressInfo := fmt.Sprintf(
		"%s Scanning... %d/%d hosts | Active: %d | Ports: %d",
		model.spinner.View(),
		model.scanInfo.HostsScanned,
		model.scanInfo.TotalHosts,
		model.scanInfo.ActiveHosts,
		model.scanInfo.OpenPorts,
	)

	if model.scanInfo.TotalHosts > 0 {
		percent := (float64(model.scanInfo.HostsScanned) / float64(model.scanInfo.TotalHosts)) * 100
		progressInfo += fmt.Sprintf(" (%.1f%%)", percent)
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Render(progressInfo)
}

func (p *ProgressComponent) Height() int {
	return 1
}

type StatsComponent struct{}

func NewStatsComponent() *StatsComponent {
	return &StatsComponent{}
}

func (s *StatsComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	return nil
}

func (s *StatsComponent) View(model *UIModel) string {
	if model.state == stateScanning {
		var elapsed time.Duration
		if !model.scanInfo.StartTime.IsZero() {
			elapsed = time.Since(model.scanInfo.StartTime)
		} else {
			elapsed = 0
		}
		stats := fmt.Sprintf(
			"🕒 Elapsed: %v | ✅ Active Hosts: %d | 🔓 Open Ports: %d",
			elapsed.Round(time.Second),
			model.scanInfo.ActiveHosts,
			model.scanInfo.OpenPorts,
		)
		return statsStyle.Render(stats)
	} else {
		totalHosts := model.scanInfo.TotalHosts
		activeHosts := model.scanInfo.ActiveHosts
		totalPorts := model.scanInfo.OpenPorts
		var elapsed time.Duration
		if !model.scanEndTime.IsZero() && !model.scanInfo.StartTime.IsZero() {
			elapsed = model.scanEndTime.Sub(model.scanInfo.StartTime)
		} else {
			elapsed = time.Since(model.scanInfo.StartTime)
		}

		finalStats := fmt.Sprintf(
			"📊 Scan Complete!\n\n"+
				"🎯 Total Hosts Scanned: %d\n"+
				"✅ Active Hosts Found: %d\n"+
				"🔓 Total Open Ports: %d\n"+
				"⏱️  Duration: %v",
			totalHosts, activeHosts, totalPorts, elapsed.Round(time.Millisecond),
		)
		return statsStyle.Render(finalStats)
	}
}

func (s *StatsComponent) Height() int {
	if s == nil {
		return 1
	}
	return 1
}

type SummaryComponent struct{}

func NewSummaryComponent() *SummaryComponent {
	return &SummaryComponent{}
}

func (s *SummaryComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	return nil
}

func (s *SummaryComponent) View(model *UIModel) string {
	if model.state != stateComplete || len(model.results) == 0 {
		return ""
	}

	summaryHeader := ""
	if len(model.filteredResults) > 0 && len(model.filteredResults) < len(model.results) {
		filteredPorts := 0
		for _, host := range model.filteredResults {
			filteredPorts += len(host.Services)
		}
		if model.searchOnlyMode {
			summaryHeader = fmt.Sprintf("🔍 Focused Search Results:\n"+
				"🔍 Found %d hosts matching your search\n"+
				"🔓 %d open ports in results\n"+
				"💡 Press ESC to clear search",
				len(model.filteredResults), filteredPorts)
		} else {
			summaryHeader = fmt.Sprintf("🔍 Search Results Summary:\n"+
				"🔍 Found %d hosts (of %d total) matching your search\n"+
				"🔓 %d open ports in filtered results\n"+
				"💡 Press ESC to clear search and show all results",
				len(model.filteredResults), len(model.results), filteredPorts)
		}
	} else {
		if model.ipsOnly {
			summaryHeader = fmt.Sprintf("📋 Scan Results Summary:\n"+
				"🖥️  %d active hosts discovered\n"+
				"🌐 IP discovery scan completed",
				len(model.results))
		} else {
			summaryHeader = fmt.Sprintf("📋 Scan Results Summary:\n"+
				"🖥️  %d active hosts discovered\n"+
				"🔓 %d total open ports found",
				len(model.results), model.scanInfo.OpenPorts)
		}
	}

	return summaryHeader
}

func (s *SummaryComponent) Height() int {
	return 4
}

type HelpComponent struct{}

func NewHelpComponent() *HelpComponent {
	return &HelpComponent{}
}

func (h *HelpComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	return nil
}

func (h *HelpComponent) View(model *UIModel) string {
	if model.state == stateScanning {
		return "💡 Press 'q' or 'Ctrl+C' to quit"
	} else {
		return "💡 Navigation: ↑/↓ or j/k to scroll | Page Up/Down | Home/End | / to search | ESC to clear | 'r' to rescan | 'q' to exit"
	}
}

func (h *HelpComponent) Height() int {
	return 1
}
