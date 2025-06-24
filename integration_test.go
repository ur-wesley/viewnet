package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestIntegrationNonInteractiveMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	csvFile := "test_integration.csv"
	defer func() {
		os.Remove(csvFile)
	}()

	hosts := []*HostInfo{
		{
			IP:           "127.0.0.1",
			MAC:          "00:00:00:00:00:00",
			Vendor:       "Test Vendor",
			Hostname:     "localhost",
			IsReachable:  true,
			ResponseTime: 1 * time.Millisecond,
			Services: []ServiceInfo{
				{Port: 22, Service: "SSH", Version: "2.0"},
			},
		},
	}

	err := exportToCSV(csvFile, hosts)
	if err != nil {
		t.Fatalf("CSV export failed: %v", err)
	}

	_, err = os.Stat(csvFile)
	if err != nil {
		t.Fatalf("CSV file was not created: %v", err)
	}

	content, err := os.ReadFile(csvFile)
	if err != nil {
		t.Fatalf("failed to read CSV file: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "IP Address") {
		t.Error("CSV header not found")
	}

	if !strings.Contains(contentStr, "127.0.0.1") {
		t.Error("test data not found in CSV")
	}
}

func TestIntegrationPortListParsing(t *testing.T) {
	portStr := "22,80,443"
	ports, err := parsePortList(portStr)
	if err != nil {
		t.Fatalf("failed to parse port list: %v", err)
	}

	expected := []int{22, 80, 443}
	if len(ports) != len(expected) {
		t.Fatalf("expected %d ports, got %d", len(expected), len(ports))
	}

	for i, port := range ports {
		if port != expected[i] {
			t.Errorf("expected port %d, got %d", expected[i], port)
		}
	}
}

func TestIntegrationSearchFiltering(t *testing.T) {
	hosts := []*HostInfo{
		{IP: "192.168.1.1", Vendor: "Dell", Hostname: "server1"},
		{IP: "192.168.1.2", Vendor: "HP", Hostname: "server2"},
		{IP: "10.0.0.1", Vendor: "Cisco", Hostname: "router"},
	}

	tests := []struct {
		name     string
		search   string
		expected int
	}{
		{"search by vendor", "dell", 1},
		{"search by IP range", "192.168", 2},
		{"search by hostname", "server", 2},
		{"search all", "", 3},
		{"no matches", "notfound", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matchCount int
			for _, host := range hosts {
				if tt.search == "" || matchesSearch(host, strings.ToLower(tt.search)) {
					matchCount++
				}
			}

			if matchCount != tt.expected {
				t.Errorf("search %q: expected %d matches, got %d", tt.search, tt.expected, matchCount)
			}
		})
	}
}

func TestIntegrationIPSorting(t *testing.T) {
	hosts := []*HostInfo{
		{IP: "192.168.1.100"},
		{IP: "192.168.1.1"},
		{IP: "192.168.1.10"},
		{IP: "10.0.0.1"},
		{IP: "172.16.0.1"},
		{IP: "192.168.1.2"},
	}

	sortHostsByIP(hosts)

	expected := []string{
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"192.168.1.2",
		"192.168.1.10",
		"192.168.1.100",
	}

	for i, expectedIP := range expected {
		if hosts[i].IP != expectedIP {
			t.Errorf("position %d: expected %s, got %s", i, expectedIP, hosts[i].IP)
		}
	}
}

func TestIntegrationVendorDetection(t *testing.T) {
	hosts := []*HostInfo{
		{Vendor: "Dell"},
		{Vendor: "HP"},
		{Vendor: "Dell"},
		{Vendor: "Unknown"},
		{Vendor: ""},
		{Vendor: "Cisco"},
	}

	vendors := getUniqueVendors(hosts)

	if !strings.Contains(vendors, "Dell") {
		t.Error("vendors should contain Dell")
	}
	if !strings.Contains(vendors, "HP") {
		t.Error("vendors should contain HP")
	}
	if !strings.Contains(vendors, "Cisco") {
		t.Error("vendors should contain Cisco")
	}
	if strings.Contains(vendors, "Unknown") {
		t.Error("vendors should not contain Unknown")
	}
}

func TestIntegrationFuzzySearch(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		expected bool
	}{
		{"Dell Inc.", "dell", true},
		{"Hewlett-Packard", "hp", true},
		{"Cisco Systems", "cisco", true},

		{"192.168.1.100", "192", true},
		{"192.168.1.100", "1681", true},

		{"server-web-01", "srv", true},
		{"mail.example.com", "mail", true},
	}

	for _, tt := range tests {
		t.Run(tt.text+"/"+tt.pattern, func(t *testing.T) {
			result := fuzzyMatch(strings.ToLower(tt.text), strings.ToLower(tt.pattern))
			if result != tt.expected {
				t.Errorf("fuzzyMatch(%q, %q) = %v, expected %v", tt.text, tt.pattern, result, tt.expected)
			}
		})
	}
}
