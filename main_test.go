package main

import (
	"encoding/csv"
	"os"
	"testing"
	"time"
)

func TestParsePortList(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []int
		expectError bool
	}{
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: false,
		},
		{
			name:        "single port",
			input:       "80",
			expected:    []int{80},
			expectError: false,
		},
		{
			name:        "multiple ports",
			input:       "22,80,443",
			expected:    []int{22, 80, 443},
			expectError: false,
		},
		{
			name:        "ports with spaces",
			input:       "22, 80, 443",
			expected:    []int{22, 80, 443},
			expectError: false,
		},
		{
			name:        "ports with empty values",
			input:       "22,,80,443",
			expected:    []int{22, 80, 443},
			expectError: false,
		},
		{
			name:        "invalid port string",
			input:       "22,abc,443",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "port out of range low",
			input:       "0,80",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "port out of range high",
			input:       "22,65536",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "valid edge cases",
			input:       "1,65535",
			expected:    []int{1, 65535},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePortList(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d ports, got %d", len(tt.expected), len(result))
				return
			}

			for i, port := range result {
				if port != tt.expected[i] {
					t.Errorf("expected port %d at index %d, got %d", tt.expected[i], i, port)
				}
			}
		})
	}
}

func TestSortHostsByIP(t *testing.T) {
	hosts := []*HostInfo{
		{IP: "192.168.1.10"},
		{IP: "192.168.1.2"},
		{IP: "192.168.1.100"},
		{IP: "192.168.1.1"},
		{IP: "10.0.0.1"},
		{IP: "192.168.1.20"},
	}

	sortHostsByIP(hosts)

	expected := []string{
		"10.0.0.1",
		"192.168.1.1",
		"192.168.1.2",
		"192.168.1.10",
		"192.168.1.20",
		"192.168.1.100",
	}

	for i, host := range hosts {
		if host.IP != expected[i] {
			t.Errorf("expected IP %s at index %d, got %s", expected[i], i, host.IP)
		}
	}
}

func TestSortHostsByIPWithInvalidIPs(t *testing.T) {
	hosts := []*HostInfo{
		{IP: "invalid-ip"},
		{IP: "192.168.1.1"},
		{IP: "another-invalid"},
		{IP: "10.0.0.1"},
	}

	sortHostsByIP(hosts)

	if len(hosts) != 4 {
		t.Errorf("expected 4 hosts after sorting, got %d", len(hosts))
	}
}

func TestExportToCSV(t *testing.T) {
	tempFile := "test_export.csv"
	defer func() {
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			t.Logf("failed to remove test file: %v", err)
		}
	}()

	hosts := []*HostInfo{
		{
			IP:           "192.168.1.1",
			MAC:          "AA:BB:CC:DD:EE:FF",
			Vendor:       "Test Vendor",
			Hostname:     "test-host",
			IsReachable:  true,
			ResponseTime: 50 * time.Millisecond,
			Services: []ServiceInfo{
				{Port: 22, Service: "SSH", Version: "2.0"},
				{Port: 80, Service: "HTTP", Version: "1.1"},
			},
		},
		{
			IP:          "192.168.1.2",
			IsReachable: false,
		},
	}

	err := exportToCSV(tempFile, hosts)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	file, err := os.Open(tempFile)
	if err != nil {
		t.Fatalf("failed to open exported file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}

	expectedHeader := []string{"IP Address", "MAC Address", "Vendor", "Hostname", "Is Reachable", "Response Time (ms)", "Open Ports", "Services"}
	if len(records) < 1 {
		t.Fatal("no header found")
	}

	for i, col := range expectedHeader {
		if i >= len(records[0]) || records[0][i] != col {
			t.Errorf("header mismatch at column %d: expected %s, got %s", i, col, records[0][i])
		}
	}

	if len(records) != 3 {
		t.Errorf("expected 3 rows (header + 2 data), got %d", len(records))
	}

	if records[1][0] != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", records[1][0])
	}
	if records[1][4] != "true" {
		t.Errorf("expected reachable=true, got %s", records[1][4])
	}
}
