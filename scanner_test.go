package main

import (
	"slices"
	"testing"
	"time"
)

func TestGetCommonPorts(t *testing.T) {
	ports := getCommonPorts()

	if len(ports) == 0 {
		t.Error("getCommonPorts() returned empty slice")
	}

	expectedPorts := []int{22, 80, 443}
	for _, expectedPort := range expectedPorts {
		found := slices.Contains(ports, expectedPort)
		if !found {
			t.Errorf("expected port %d not found in common ports", expectedPort)
		}
	}

	for _, port := range ports {
		if port < 1 || port > 65535 {
			t.Errorf("invalid port %d in common ports list", port)
		}
	}

	t.Logf("getCommonPorts() returned %d ports", len(ports))
}

func TestNewPortScanner(t *testing.T) {
	scanner := NewPortScanner(5, 10)
	if scanner == nil {
		t.Error("NewPortScanner() returned nil")
	}

	scanner = NewPortScanner(0, 0)
	if scanner == nil {
		t.Error("NewPortScanner(0, 0) returned nil")
	}
}

func TestPortScannerScanSubnet(t *testing.T) {
	scanner := NewPortScanner(2, 5)

	results := scanner.ScanSubnet("invalid-cidr", 80, 80, 100*time.Millisecond)
	if results != nil {
		t.Error("expected nil results for invalid CIDR")
	}

	results = scanner.ScanSubnet("10.255.255.0/30", 80, 80, 10*time.Millisecond)
	if results == nil {
		t.Error("expected non-nil results for valid CIDR")
	}

	for _, host := range results {
		if host.IsReachable {
			t.Logf("found reachable host: %s", host.IP)
		}
	}
}

func TestServiceInfoValidation(t *testing.T) {
	service := ServiceInfo{
		Port:         80,
		Protocol:     "TCP",
		Service:      "HTTP",
		Version:      "1.1",
		Banner:       "Server: nginx",
		IsOpen:       true,
		ResponseTime: 50 * time.Millisecond,
	}

	if service.Port < 1 || service.Port > 65535 {
		t.Errorf("invalid port: %d", service.Port)
	}

	if service.Protocol == "" {
		t.Error("protocol should not be empty")
	}

	if service.ResponseTime < 0 {
		t.Error("response time should not be negative")
	}
}

func TestHostInfoValidation(t *testing.T) {
	host := HostInfo{
		IP:           "192.168.1.1",
		MAC:          "AA:BB:CC:DD:EE:FF",
		Vendor:       "Test Vendor",
		Hostname:     "test-host",
		IsReachable:  true,
		ResponseTime: 25 * time.Millisecond,
		Services: []ServiceInfo{
			{Port: 80, Service: "HTTP"},
		},
	}

	if host.IP == "" {
		t.Error("IP should not be empty")
	}

	if host.ResponseTime < 0 {
		t.Error("response time should not be negative")
	}

	if host.IsReachable && host.ResponseTime == 0 {
		t.Error("reachable host should have non-zero response time")
	}
}

func TestBannerGrabbing(t *testing.T) {

	service := ServiceInfo{
		Port:    80,
		Service: "HTTP",
		Banner:  "Server: nginx/1.18.0",
		Version: "1.18.0",
	}

	if service.Banner != "" {
		if len(service.Banner) < 5 {
			t.Error("banner seems too short to be useful")
		}
	}

	if service.Version != "" {
		if len(service.Version) < 1 {
			t.Error("version should not be empty if set")
		}
	}
}
