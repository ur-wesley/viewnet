package main

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

func getLocalSubnet() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			var mask net.IPMask

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				mask = v.Mask
			case *net.IPAddr:
				ip = v.IP
				if ip.To4() != nil {
					mask = net.CIDRMask(24, 32)
				} else {
					mask = net.CIDRMask(64, 128)
				}
			}

			if ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				if isPrivateIP(ip) {
					network := &net.IPNet{IP: ip.Mask(mask), Mask: mask}
					return network.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, rangeStr := range privateRanges {
		_, privateNet, _ := net.ParseCIDR(rangeStr)
		if privateNet.Contains(ip) {
			return true
		}
	}
	return false
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

func getMACAddressNew(ip string) (string, string) {
	var cmd *exec.Cmd
	var output []byte
	var err error
	
	if runtime.GOOS == "windows" {
		cmd = exec.Command("arp", "-a", ip)
		output, err = cmd.Output()
		if err != nil {
			return "", ""
		}
	} else {
		cmd = exec.Command("ip", "neighbor", "show", ip)
		output, err = cmd.Output()
		
		if err != nil || len(strings.TrimSpace(string(output))) == 0 {
			cmd = exec.Command("arp", "-n", ip)
			output, _ = cmd.Output()
		}
		
		if len(strings.TrimSpace(string(output))) == 0 {
			if isIPInLocalSubnet(ip) {
				arpingCmd := exec.Command("arping", "-c", "1", "-w", "1", ip)
				arpingCmd.Run()
				
				cmd = exec.Command("ip", "neighbor", "show", ip)
				output, _ = cmd.Output()
			}
		}
	}
	
	if len(output) == 0 {
		return "", ""
	}

	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}[:-]){5}[0-9a-fA-F]{2}`)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip) || runtime.GOOS != "windows" {
			if runtime.GOOS != "windows" && strings.Contains(line, "lladdr") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == "lladdr" && i+1 < len(fields) {
						mac := strings.ToUpper(strings.ReplaceAll(fields[i+1], "-", ":"))
						vendor := getEnhancedVendor(mac)
						return mac, vendor
					}
				}
			}
			
			fields := strings.Fields(line)
			for _, field := range fields {
				if macRegex.MatchString(field) {
					mac := strings.ToUpper(strings.ReplaceAll(field, "-", ":"))
					vendor := getEnhancedVendor(mac)
					return mac, vendor
				}
			}
		}
	}

	return "", ""
}

func ipToInt(ip string) uint32 {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return 0
	}

	var result uint32
	for i, part := range parts {
		val := 0
		fmt.Sscanf(part, "%d", &val)
		result |= uint32(val) << (8 * (3 - i))
	}
	return result
}

func isIPInLocalSubnet(targetIP string) bool {
	target := net.ParseIP(targetIP)
	if target == nil {
		return false
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				if ipNet.Contains(target) {
					return true
				}
			}
		}
	}
	return false
}
