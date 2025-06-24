package main

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"private 10.x.x.x", "10.0.0.1", true},
		{"private 172.16.x.x", "172.16.0.1", true},
		{"private 192.168.x.x", "192.168.1.1", true},

		{"public Google DNS", "8.8.8.8", false},
		{"public Cloudflare", "1.1.1.1", false},

		{"localhost", "127.0.0.1", false},
		{"link-local", "169.254.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid IP address: %s", tt.ip)
			}

			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIpToInt(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{"zero IP", "0.0.0.0", 0},
		{"localhost", "127.0.0.1", 2130706433},
		{"private IP", "192.168.1.1", 3232235777},
		{"max IP", "255.255.255.255", 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToInt(tt.ip)
			if result != tt.expected {
				t.Errorf("ipToInt(%s) = %d, expected %d", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIpToIntInvalidIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{"empty string", ""},
		{"invalid format", "not.an.ip"},
		{"too few parts", "192.168.1"},
		{"too many parts", "192.168.1.1.1"},
		{"IPv6", "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToInt(tt.ip)
			if result != 0 {
				t.Errorf("ipToInt(%s) = %d, expected 0 for invalid IP", tt.ip, result)
			}
		})
	}
}

func TestGetMACAddressNew(t *testing.T) {
	mac, vendor := getMACAddressNew("127.0.0.1")

	t.Logf("getMACAddressNew(127.0.0.1) = mac:%s, vendor:%s", mac, vendor)

	if mac != "" && mac != "N/A" {
		if len(mac) < 12 {
			t.Logf("MAC address seems short: %s", mac)
		}
	}
}

func TestGetLocalSubnet(t *testing.T) {
	subnet, err := getLocalSubnet()
	if err != nil {
		t.Logf("getLocalSubnet() error (may be expected in test environment): %v", err)
		return
	}

	_, _, err = net.ParseCIDR(subnet)
	if err != nil {
		t.Errorf("getLocalSubnet() returned invalid CIDR %s: %v", subnet, err)
	}

	t.Logf("detected local subnet: %s", subnet)
}
