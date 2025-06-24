package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TableComponent struct{}

func NewTableComponent() *TableComponent {
	return &TableComponent{}
}

func (t *TableComponent) Update(msg tea.Msg, model *UIModel) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !model.searchFocused {
			switch msg.String() {
			case "up", "k":
				if model.scrollOffset > 0 {
					model.scrollOffset--
				}
				return nil
			case "down", "j":
				resultsToShow := model.filteredResults
				if len(resultsToShow) == 0 {
					resultsToShow = model.results
				}
				if len(resultsToShow) > 0 {
					maxScroll := max(len(resultsToShow)-model.viewHeight, 0)
					if model.scrollOffset < maxScroll {
						model.scrollOffset++
					}
				}
				return nil
			case "home":
				model.scrollOffset = 0
				return nil
			case "end":
				resultsToShow := model.filteredResults
				if len(resultsToShow) == 0 {
					resultsToShow = model.results
				}
				if len(resultsToShow) > 0 {
					maxScroll := max(len(resultsToShow)-model.viewHeight, 0)
					model.scrollOffset = maxScroll
				}
				return nil
			case "pageup":
				pageSize := max(model.viewHeight/2, 1)
				model.scrollOffset -= pageSize
				minScroll := -4
				if model.scrollOffset < minScroll {
					model.scrollOffset = minScroll
				}
				return nil
			case "pagedown":
				resultsToShow := model.filteredResults
				if len(resultsToShow) == 0 {
					resultsToShow = model.results
				}
				if len(resultsToShow) > 0 {
					pageSize := max(model.viewHeight/2, 1)
					maxScroll := max(len(resultsToShow)-model.viewHeight, 0)
					model.scrollOffset += pageSize
					if model.scrollOffset > maxScroll {
						model.scrollOffset = maxScroll
					}
				}
				return nil
			}
		}
	}
	return nil
}

func (t *TableComponent) View(model *UIModel) string {
	resultsToShow := model.filteredResults
	if len(resultsToShow) == 0 {
		resultsToShow = model.results
	}

	if len(resultsToShow) == 0 {
		if model.state == stateScanning {
			return "🔍 Scanning for hosts... (results will appear as they're discovered)"
		} else {
			return "🔍 No active hosts found."
		}
	}

	var content []string

	resultHeader := ""
	if len(model.filteredResults) > 0 && len(model.filteredResults) < len(model.results) {
		resultHeader = "🖥️  Filtered Host Details:"
	} else if model.state == stateScanning {
		resultHeader = fmt.Sprintf("🖥️  Live Results (%d discovered):", len(resultsToShow))
	} else {
		resultHeader = "🖥️  Host Details:"
	}
	content = append(content, resultHeader)

	availableHeight := model.viewHeight - 1
	if availableHeight <= 0 {
		availableHeight = 5
	}

	t.renderTableRows(model, resultsToShow, model.scrollOffset, availableHeight, &content)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (t *TableComponent) renderTableRows(model *UIModel, resultsToShow []*HostInfo, startIndex, maxRows int, content *[]string) {
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= len(resultsToShow) {
		return
	}

	endIdx := min(startIndex+maxRows, len(resultsToShow))

	if model.windowWidth > 80 {
		headerRow := t.renderTableHeader(model.windowWidth)
		*content = append(*content, headerRow)
		maxRows--
		if maxRows <= 0 {
			return
		}
		endIdx = min(startIndex+maxRows, len(resultsToShow))
	}

	for i := startIndex; i < endIdx && i < len(resultsToShow); i++ {
		host := resultsToShow[i]
		row := t.renderTableRow(host, model.windowWidth)
		*content = append(*content, row)
	}

	if len(resultsToShow) > model.viewHeight {
		scrollInfo := t.renderScrollInfo(startIndex, endIdx, len(resultsToShow))
		*content = append(*content, scrollInfo)
	}
}

func (t *TableComponent) Height() int {
	return -1
}

func (t *TableComponent) renderTableHeader(width int) string {
	if width < 80 {
		return ""
	}

	ipWidth := 16
	macWidth := 18
	vendorWidth := 22

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1)

	header := fmt.Sprintf("%-*s │ %-*s │ %-*s │ %s",
		ipWidth, "IP ADDRESS",
		macWidth, "MAC ADDRESS",
		vendorWidth, "VENDOR",
		"OPEN PORTS")

	return headerStyle.Render(header)
}

func (t *TableComponent) renderTableRow(host *HostInfo, width int) string {
	if width < 80 {
		return t.renderHostCard(host)
	}

	ipWidth := 16
	macWidth := 18
	vendorWidth := 22
	portsWidth := max(width-ipWidth-macWidth-vendorWidth-6, 20)

	ipCell := host.IP
	if len(ipCell) > ipWidth-1 {
		ipCell = ipCell[:ipWidth-4] + "..."
	}

	macCell := host.MAC
	if macCell == "" {
		macCell = "N/A"
	}
	if len(macCell) > macWidth-1 {
		macCell = macCell[:macWidth-4] + "..."
	}

	vendorCell := host.Vendor
	if vendorCell == "" || vendorCell == "Unknown" {
		vendorCell = "Unknown"
	}
	if len(vendorCell) > vendorWidth-1 {
		vendorCell = vendorCell[:vendorWidth-4] + "..."
	}

	var portList []string
	for _, service := range host.Services {
		portStr := fmt.Sprintf("%d", service.Port)
		if service.Service != "" && service.Service != "unknown" {
			portStr += "/" + service.Service
		}
		portList = append(portList, portStr)
	}
	portsCell := strings.Join(portList, ", ")
	if len(portsCell) > portsWidth-1 {
		portsCell = portsCell[:portsWidth-4] + "..."
	}

	row := fmt.Sprintf("%-*s │ %-*s │ %-*s │ %s",
		ipWidth, ipCell,
		macWidth, macCell,
		vendorWidth, vendorCell,
		portsCell)

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeft(true).
		Padding(0, 1)

	if host.IsReachable {
		style = style.BorderForeground(lipgloss.Color("10"))
	} else {
		style = style.BorderForeground(lipgloss.Color("9"))
	}

	return style.Render(row)
}

func (t *TableComponent) renderHostCard(host *HostInfo) string {
	hostHeader := fmt.Sprintf("🖥️  %s", host.IP)
	if host.Hostname != "" {
		hostHeader += fmt.Sprintf(" (%s)", host.Hostname)
	}
	hostHeader += fmt.Sprintf(" - %v", host.ResponseTime.Round(time.Millisecond))

	var macInfo string
	if host.MAC != "" {
		macInfo = fmt.Sprintf("   📱 MAC: %s", host.MAC)
		if host.Vendor != "" && host.Vendor != "Unknown" {
			macInfo += fmt.Sprintf(" (%s)", host.Vendor)
		}
	}

	var services []string
	for _, service := range host.Services {
		serviceText := fmt.Sprintf("   🔓 %d/%s", service.Port, service.Service)
		if service.Version != "" {
			serviceText += fmt.Sprintf(" (%s)", service.Version)
		}
		if service.Banner != "" && len(service.Banner) < 50 {
			serviceText += fmt.Sprintf(" - %s", service.Banner)
		}
		services = append(services, openPortStyle.Render(serviceText))
	}

	var hostContent []string
	hostContent = append(hostContent, hostHeader)
	if macInfo != "" {
		hostContent = append(hostContent, macInfo)
	}
	hostContent = append(hostContent, services...)

	style := upHostStyle
	if !host.IsReachable {
		style = downHostStyle
	}

	content := lipgloss.JoinVertical(lipgloss.Left, hostContent...)
	return style.Render(content)
}

func (t *TableComponent) renderScrollInfo(startIdx, endIdx, totalResults int) string {
	scrollInfo := ""
	showingFrom := startIdx + 1
	showingTo := min(endIdx, totalResults)

	if startIdx > 0 && endIdx < totalResults {
		scrollInfo = fmt.Sprintf("📜 Showing %d-%d of %d hosts | ↑ More above | ↓ More below",
			showingFrom, showingTo, totalResults)
	} else if startIdx > 0 {
		scrollInfo = fmt.Sprintf("📜 Showing %d-%d of %d hosts | ↑ More above",
			showingFrom, showingTo, totalResults)
	} else if endIdx < totalResults {
		scrollInfo = fmt.Sprintf("📜 Showing %d-%d of %d hosts | ↓ More below",
			showingFrom, showingTo, totalResults)
	} else {
		scrollInfo = fmt.Sprintf("📜 Showing all %d hosts", totalResults)
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(scrollInfo)
}
