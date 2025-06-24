package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

func BenchmarkParsePortList(b *testing.B) {
	portStr := "21,22,23,25,53,80,110,111,135,139,143,443,993,995,1723,3389,5900,8080"

	for b.Loop() {
		_, err := parsePortList(portStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSortHostsByIP(b *testing.B) {
	hosts := make([]*HostInfo, 1000)
	for i := range 1000 {
		hosts[i] = &HostInfo{
			IP: fmt.Sprintf("192.168.%d.%d", (i/254)+1, (i%254)+1),
		}
	}

	for b.Loop() {
		hostsCopy := make([]*HostInfo, len(hosts))
		copy(hostsCopy, hosts)
		sortHostsByIP(hostsCopy)
	}
}

func BenchmarkFuzzyMatch(b *testing.B) {
	text := "Dell Technologies Inc."
	pattern := "dell"

	for b.Loop() {
		fuzzyMatch(text, pattern)
	}
}

func BenchmarkMatchesSearch(b *testing.B) {
	host := &HostInfo{
		IP:       "192.168.1.100",
		Hostname: "server-web-01.example.com",
		Vendor:   "Dell Technologies Inc.",
		MAC:      "AA:BB:CC:DD:EE:FF",
		Services: []ServiceInfo{
			{Port: 22, Service: "SSH", Version: "OpenSSH 8.0"},
			{Port: 80, Service: "HTTP", Version: "nginx 1.18"},
			{Port: 443, Service: "HTTPS", Version: "nginx 1.18"},
		},
	}
	searchTerm := "dell"

	for b.Loop() {
		matchesSearch(host, searchTerm)
	}
}

func BenchmarkGetUniqueVendors(b *testing.B) {
	hosts := make([]*HostInfo, 500)
	vendors := []string{"Dell", "HP", "Cisco", "Netgear", "D-Link", "TP-Link", "Apple", "Microsoft"}

	for i := range 500 {
		hosts[i] = &HostInfo{
			IP:     fmt.Sprintf("192.168.1.%d", i%254+1),
			Vendor: vendors[i%len(vendors)],
		}
	}

	for b.Loop() {
		getUniqueVendors(hosts)
	}
}

func BenchmarkExportToCSV(b *testing.B) {
	hosts := make([]*HostInfo, 100)
	for i := range 100 {
		hosts[i] = &HostInfo{
			IP:           fmt.Sprintf("192.168.1.%d", i+1),
			MAC:          fmt.Sprintf("AA:BB:CC:DD:EE:%02X", i),
			Vendor:       "Test Vendor",
			Hostname:     fmt.Sprintf("host-%d", i),
			IsReachable:  true,
			ResponseTime: time.Duration(i) * time.Millisecond,
			Services: []ServiceInfo{
				{Port: 22, Service: "SSH"},
				{Port: 80, Service: "HTTP"},
			},
		}
	}

	for i := 0; b.Loop(); i++ {
		filename := fmt.Sprintf("bench_test_%d.csv", i)
		err := exportToCSV(filename, hosts)
		if err != nil {
			b.Fatal(err)
		}
		os.Remove(filename)
	}
}

func BenchmarkIpToInt(b *testing.B) {
	ip := "192.168.1.100"

	for b.Loop() {
		ipToInt(ip)
	}
}

func BenchmarkFilterResults(b *testing.B) {
	model := &UIModel{
		searchInput: textinput.New(),
		results:     make([]*HostInfo, 1000),
	}

	for i := range 1000 {
		model.results[i] = &HostInfo{
			IP:     fmt.Sprintf("192.168.%d.%d", (i/254)+1, (i%254)+1),
			Vendor: fmt.Sprintf("Vendor%d", i%10),
		}
	}

	model.searchInput.SetValue("192.168")

	for b.Loop() {
		filterResults(model)
	}
}

func BenchmarkOptimizedSearch(b *testing.B) {
	host := &HostInfo{
		IP:     "192.168.1.100",
		Vendor: "Dell Technologies",
	}
	searchTerm := "192.168"

	for b.Loop() {
		optimizedSearch(host, searchTerm)
	}
}
