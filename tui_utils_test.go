package main

import (
	"testing"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		pattern  string
		expected bool
	}{
		{"case insensitive", "hello", "hello", true},

		{"subsequence", "hello world", "hlo", true},
		{"scattered letters", "hello", "hlo", true},
		{"single char", "hello", "h", true},

		{"no match", "hello", "xyz", false},
		{"wrong order", "hello", "leh", false},
		{"missing chars", "hello", "hellox", false},

		{"empty pattern", "hello", "", true},
		{"empty text", "", "hello", false},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzyMatch(tt.text, tt.pattern)
			if result != tt.expected {
				t.Errorf("fuzzyMatch(%q, %q) = %v, expected %v", tt.text, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatchesSearch(t *testing.T) {
	host := &HostInfo{
		IP:       "192.168.1.100",
		Hostname: "test-server",
		Vendor:   "Dell Inc",
		MAC:      "AA:BB:CC:DD:EE:FF",
		Services: []ServiceInfo{
			{Port: 22, Service: "SSH"},
			{Port: 80, Service: "HTTP"},
		},
	}

	tests := []struct {
		name       string
		searchTerm string
		expected   bool
	}{
		{"full IP", "192.168.1.100", true},
		{"partial IP", "192.168", true},
		{"IP fuzzy", "1921681", true},

		{"full hostname", "test-server", true},
		{"partial hostname", "test", true},
		{"hostname fuzzy", "tst", true},

		{"full vendor", "dell inc", true},
		{"partial vendor", "dell", true},
		{"vendor fuzzy", "dl", true},

		{"full MAC", "aa:bb:cc:dd:ee:ff", true},
		{"partial MAC", "aa:bb", true},
		{"MAC without colons", "aabbcc", true},
		{"service name", "ssh", true},
		{"service fuzzy", "ht", true},


		{"no match", "xyz123", false},
		{"wrong IP", "10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesSearch(host, tt.searchTerm)
			if result != tt.expected {
				t.Errorf("matchesSearch(%q) = %v, expected %v", tt.searchTerm, result, tt.expected)
			}
		})
	}
}

func TestMatchesFocusedSearch(t *testing.T) {
	host := &HostInfo{
		IP:       "192.168.1.100",
		Hostname: "test-server",
		Vendor:   "Dell Inc",
		Services: []ServiceInfo{
			{Port: 22, Service: "SSH"},
		},
	}

	tests := []struct {
		name       string
		searchTerm string
		expected   bool
	}{
		{"IP match", "192.168", true},
		{"vendor match", "dell", true},

		{"hostname no match", "test", false},
		{"service no match", "ssh", false},
		{"port no match", "22", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFocusedSearch(host, tt.searchTerm)
			if result != tt.expected {
				t.Errorf("matchesFocusedSearch(%q) = %v, expected %v", tt.searchTerm, result, tt.expected)
			}
		})
	}
}

func TestGetUniqueVendors(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []*HostInfo
		expected string
	}{
		{
			name:     "no vendors",
			hosts:    []*HostInfo{{IP: "1.1.1.1"}},
			expected: "None detected",
		},
		{
			name: "single vendor",
			hosts: []*HostInfo{
				{IP: "1.1.1.1", Vendor: "Dell"},
			},
			expected: "Dell",
		},
		{
			name: "multiple vendors",
			hosts: []*HostInfo{
				{IP: "1.1.1.1", Vendor: "Dell"},
				{IP: "1.1.1.2", Vendor: "HP"},
				{IP: "1.1.1.3", Vendor: "Dell"},
			},
			expected: "Dell, HP",
		},
		{
			name: "unknown vendors filtered",
			hosts: []*HostInfo{
				{IP: "1.1.1.1", Vendor: "Dell"},
				{IP: "1.1.1.2", Vendor: "Unknown"},
				{IP: "1.1.1.3", Vendor: ""},
			},
			expected: "Dell",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueVendors(tt.hosts)
			if result != tt.expected {
				t.Errorf("getUniqueVendors() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestIsIPPattern(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		expected bool
	}{
		{"full IP", "192.168.1.1", true},
		{"partial IP", "192.168", true},
		{"IP with dots", "10.0", true},
		{"not IP", "dell", false},
		{"not IP with numbers", "test123", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIPPattern(tt.term)
			if result != tt.expected {
				t.Errorf("isIPPattern(%q) = %v, expected %v", tt.term, result, tt.expected)
			}
		})
	}
}

func TestIsVendorPattern(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		expected bool
	}{
		{"vendor name", "dell", true},
		{"vendor with space", "dell inc", true},
		{"single char", "d", true},
		{"IP pattern", "192.168.1.1", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVendorPattern(tt.term)
			if result != tt.expected {
				t.Errorf("isVendorPattern(%q) = %v, expected %v", tt.term, result, tt.expected)
			}
		})
	}
}

func TestPollForUpdates(t *testing.T) {
	cmd := pollForUpdates()
	if cmd == nil {
		t.Error("pollForUpdates() returned nil command")
	}
}
