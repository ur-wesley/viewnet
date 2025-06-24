package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbletea"
)

func getUniqueVendors(results []*HostInfo) string {
	vendorMap := make(map[string]bool)
	for _, host := range results {
		if host.Vendor != "" && host.Vendor != "Unknown" {
			vendorMap[host.Vendor] = true
		}
	}

	if len(vendorMap) == 0 {
		return "None detected"
	}

	var vendors []string
	for vendor := range vendorMap {
		vendors = append(vendors, vendor)
	}

	if len(vendors) > 5 {
		return fmt.Sprintf("%s and %d more", strings.Join(vendors[:5], ", "), len(vendors)-5)
	}

	return strings.Join(vendors, ", ")
}

func matchesSearch(host *HostInfo, searchTerm string) bool {
	if fuzzyMatch(strings.ToLower(host.IP), searchTerm) {
		return true
	}

	if host.Hostname != "" && fuzzyMatch(strings.ToLower(host.Hostname), searchTerm) {
		return true
	}

	if host.Vendor != "" && fuzzyMatch(strings.ToLower(host.Vendor), searchTerm) {
		return true
	}

	if host.MAC != "" && fuzzyMatch(strings.ToLower(host.MAC), searchTerm) {
		return true
	}

	for _, service := range host.Services {
		if fuzzyMatch(strings.ToLower(service.Service), searchTerm) {
			return true
		}
		if service.Version != "" && fuzzyMatch(strings.ToLower(service.Version), searchTerm) {
			return true
		}
		if service.Banner != "" && fuzzyMatch(strings.ToLower(service.Banner), searchTerm) {
			return true
		}
	}

	return false
}

func matchesFocusedSearch(host *HostInfo, searchTerm string) bool {
	if fuzzyMatch(strings.ToLower(host.IP), searchTerm) {
		return true
	}

	if host.Vendor != "" && fuzzyMatch(strings.ToLower(host.Vendor), searchTerm) {
		return true
	}

	return false
}

func fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	if strings.Contains(text, pattern) {
		return true
	}

	textIndex := 0
	patternIndex := 0

	for textIndex < len(text) && patternIndex < len(pattern) {
		if text[textIndex] == pattern[patternIndex] {
			patternIndex++
		}
		textIndex++
	}

	return patternIndex == len(pattern)
}

func pollForUpdates() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
		return pollMsg{}
	})
}

func animateProgress() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return progress.FrameMsg{}
	})
}

func filterResults(model *UIModel) {
	searchTerm := strings.ToLower(strings.TrimSpace(model.searchInput.Value()))

	if searchTerm == "" {
		model.filteredResults = []*HostInfo{}
		model.scrollOffset = 0
		return
	}

	model.filteredResults = []*HostInfo{}

	for _, host := range model.results {
		var matches bool

		if model.searchOnlyMode && (isIPPattern(searchTerm) || isVendorPattern(searchTerm)) {
			matches = optimizedSearch(host, searchTerm)
		} else if model.searchOnlyMode {
			matches = matchesFocusedSearch(host, searchTerm)
		} else {
			matches = matchesSearch(host, searchTerm)
		}
		if matches {
			model.filteredResults = append(model.filteredResults, host)
		}
	}

	sort.Slice(model.filteredResults, func(i, j int) bool {
		return ipToInt(model.filteredResults[i].IP) < ipToInt(model.filteredResults[j].IP)
	})

	model.scrollOffset = 0
}

func optimizedSearch(host *HostInfo, searchTerm string) bool {
	searchLower := strings.ToLower(searchTerm)

	if isIPPattern(searchTerm) {
		return strings.Contains(strings.ToLower(host.IP), searchLower)
	}

	if isVendorPattern(searchTerm) {
		return host.Vendor != "" && strings.Contains(strings.ToLower(host.Vendor), searchLower)
	}

	return matchesFocusedSearch(host, searchTerm)
}

func isIPPattern(term string) bool {
	for _, char := range term {
		if char != '.' && (char < '0' || char > '9') {
			return false
		}
	}
	return len(term) > 0
}

func isVendorPattern(term string) bool {
	hasLetter := false
	for _, char := range term {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
			break
		}
	}
	return hasLetter
}
