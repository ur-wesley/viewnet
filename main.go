package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
)

func parsePortList(portStr string) ([]int, error) {
	if portStr == "" {
		return nil, nil
	}

	var ports []int
	portStrings := strings.SplitSeq(portStr, ",")

	for portStr := range portStrings {
		portStr = strings.TrimSpace(portStr)
		if portStr == "" {
			continue
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port '%s': %v", portStr, err)
		}

		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("port %d out of range (1-65535)", port)
		}

		ports = append(ports, port)
	}

	return ports, nil
}

func sortHostsByIP(hosts []*HostInfo) {
	sort.Slice(hosts, func(i, j int) bool {
		ipI := net.ParseIP(hosts[i].IP)
		ipJ := net.ParseIP(hosts[j].IP)
		if ipI == nil || ipJ == nil {
			return hosts[i].IP < hosts[j].IP
		}

		ipI = ipI.To4()
		ipJ = ipJ.To4()
		if ipI == nil || ipJ == nil {
			return hosts[i].IP < hosts[j].IP
		}

		for k := range 4 {
			if ipI[k] != ipJ[k] {
				return ipI[k] < ipJ[k]
			}
		}
		return false
	})
}

func exportToCSV(filename string, hosts []*HostInfo) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"IP Address", "MAC Address", "Vendor", "Hostname", "Is Reachable", "Response Time (ms)", "Open Ports", "Services"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, host := range hosts {
		var portList []string
		var serviceList []string

		for _, service := range host.Services {
			portList = append(portList, fmt.Sprintf("%d", service.Port))
			serviceDetail := fmt.Sprintf("%d/%s", service.Port, service.Service)
			if service.Version != "" {
				serviceDetail += fmt.Sprintf(" (%s)", service.Version)
			}
			serviceList = append(serviceList, serviceDetail)
		}

		row := []string{
			host.IP,
			host.MAC,
			host.Vendor,
			host.Hostname,
			fmt.Sprintf("%t", host.IsReachable),
			fmt.Sprintf("%.2f", float64(host.ResponseTime.Nanoseconds())/1000000.0),
			strings.Join(portList, ";"),
			strings.Join(serviceList, ";"),
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func runNonInteractiveMode(targetSubnet string, startPort, endPort, timeoutMs int, customPorts []int, ipsOnly bool, csvFile string) {
	timeout := time.Duration(timeoutMs) * time.Millisecond

	fmt.Printf("🔍 ViewNet - Non-Interactive Mode\n")
	fmt.Printf("Target: %s | ", targetSubnet)

	if ipsOnly {
		fmt.Printf("Mode: IP Discovery Only")
	} else if len(customPorts) > 0 {
		fmt.Printf("Ports: %v", customPorts)
	} else {
		fmt.Printf("Ports: %d-%d", startPort, endPort)
	}
	fmt.Printf(" | Timeout: %dms\n", timeoutMs)
	fmt.Printf("Output: %s\n\n", csvFile)

	_, ipnet, err := net.ParseCIDR(targetSubnet)
	if err != nil {
		fmt.Printf("❌ Error parsing subnet: %v\n", err)
		os.Exit(1)
	}

	var ips []string
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	fmt.Printf("📊 Scanning %d hosts...\n", len(ips))

	startTime := time.Now()
	var results []*HostInfo

	if ipsOnly || len(customPorts) > 0 {
		results = scanHostsNonInteractive(ips, startPort, endPort, timeout, customPorts, ipsOnly)
	} else {
		scanner := NewPortScanner(10, 50)
		results = scanner.ScanSubnet(targetSubnet, startPort, endPort, timeout)
	}

	duration := time.Since(startTime)

	activeHosts := 0
	totalPorts := 0
	for _, host := range results {
		if host.IsReachable {
			activeHosts++
		}
		totalPorts += len(host.Services)
	}

	fmt.Printf("✅ Scan completed in %v\n", duration.Round(time.Millisecond))
	fmt.Printf("📈 Results: %d active hosts, %d open ports\n", activeHosts, totalPorts)

	sortHostsByIP(results)

	if err := exportToCSV(csvFile, results); err != nil {
		fmt.Printf("❌ Error exporting to CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📄 Results exported to %s\n", csvFile)
}

func scanHostsNonInteractive(ips []string, startPort, endPort int, timeout time.Duration, customPorts []int, ipsOnly bool) []*HostInfo {
	hostWorkers := 10
	hostChan := make(chan string, len(ips))
	resultChan := make(chan *HostInfo, len(ips))
	var wg sync.WaitGroup

	for range hostWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range hostChan {
				ctx, cancel := context.WithTimeout(context.Background(), timeout*10)
				host := scanHostCustom(ctx, ip, startPort, endPort, timeout, 20, customPorts, ipsOnly)
				cancel()

				if host != nil {
					resultChan <- host
				}
			}
		}()
	}

	for _, ip := range ips {
		hostChan <- ip
	}
	close(hostChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []*HostInfo
	for host := range resultChan {
		results = append(results, host)
	}

	return results
}

func main() {
	subnet := flag.String("subnet", "", "CIDR to scan (auto-detects local subnet if empty)")
	startPort := flag.Int("start", 1, "start port")
	endPort := flag.Int("end", 1024, "end port")
	portList := flag.String("p", "", "comma-separated list of ports (e.g., 22,80,443)")
	ipsOnly := flag.Bool("ips", false, "scan for active IPs only (no port scanning)")
	timeoutMs := flag.Int("timeout", 200, "ms per port")
	focusedSearch := flag.Bool("focused", false, "enable focused search mode (IP and vendor only)")
	searchTerm := flag.String("s", "", "search term for IP or vendor (speeds up search)")
	csvOutput := flag.String("csv", "", "output results to CSV file (e.g., results.csv)")
	flag.Parse()
	var customPorts []int
	var err error
	if *portList != "" {
		customPorts, err = parsePortList(*portList)
		if err != nil {
			fmt.Printf("❌ Error parsing port list: %v\n", err)
			os.Exit(1)
		}
	} else if !*ipsOnly {
		customPorts = getCommonPorts()
	}

	targetSubnet := *subnet
	args := flag.Args()
	if len(args) > 0 {
		targetSubnet = args[0]
		if !strings.Contains(targetSubnet, "/") {
			targetSubnet = targetSubnet + "/32"
		}
	} else if targetSubnet == "" {
		detected, err := getLocalSubnet()
		if err != nil {
			fmt.Printf("❌ Error detecting local subnet: %v\n", err)
			fmt.Printf("Please specify a subnet manually using -subnet flag or as an argument\n")
			os.Exit(1)
		}
		targetSubnet = detected
	}
	if *csvOutput != "" {
		runNonInteractiveMode(targetSubnet, *startPort, *endPort, *timeoutMs, customPorts, *ipsOnly, *csvOutput)
		return
	}

	model := NewModularUI(targetSubnet, *startPort, *endPort, *timeoutMs, *focusedSearch, *searchTerm, customPorts, *ipsOnly)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
	if m, ok := finalModel.(*ModularUIModel); ok && m.quitting && m.err != nil {
		fmt.Printf("❌ Error: %v\n", m.err)
		os.Exit(1)
	}

	if *csvOutput != "" {
		results := GetScanResults()
		if len(results) > 0 {
			sortHostsByIP(results)

			if err := exportToCSV(*csvOutput, results); err != nil {
				fmt.Printf("❌ Error exporting to CSV: %v\n", err)
			} else {
				fmt.Printf("✅ Results exported to %s\n", *csvOutput)
			}
		} else {
			fmt.Printf("⚠️  No results to export\n")
		}
	}
}
