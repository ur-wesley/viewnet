package main

import (
	"context"
	"net"
	"sort"
	"sync"
	"time"
)

type ScanState struct {
	mu           sync.RWMutex
	isScanning   bool
	scanStart    time.Time
	hostsScanned int
	totalHosts   int
	activeHosts  int
	openPorts    int
	currentHost  string
	results      []*HostInfo
}

var globalScanState = &ScanState{}

func GetScanProgress() ScanProgress {
	globalScanState.mu.RLock()
	defer globalScanState.mu.RUnlock()

	return ScanProgress{
		CurrentHost:  globalScanState.currentHost,
		HostsScanned: globalScanState.hostsScanned,
		TotalHosts:   globalScanState.totalHosts,
		ActiveHosts:  globalScanState.activeHosts,
		OpenPorts:    globalScanState.openPorts,
		StartTime:    globalScanState.scanStart,
	}
}

func GetScanResults() []*HostInfo {
	globalScanState.mu.RLock()
	defer globalScanState.mu.RUnlock()

	results := make([]*HostInfo, len(globalScanState.results))
	copy(results, globalScanState.results)
	return results
}

func IsScanComplete() bool {
	globalScanState.mu.RLock()
	defer globalScanState.mu.RUnlock()
	return !globalScanState.isScanning
}

func StartTUIScan(targetSubnet string, startPort, endPort int, timeout time.Duration, customPorts []int, ipsOnly bool) {
	go func() {
		globalScanState.mu.Lock()
		globalScanState.isScanning = true
		globalScanState.scanStart = time.Now()
		globalScanState.hostsScanned = 0
		globalScanState.activeHosts = 0
		globalScanState.openPorts = 0
		globalScanState.results = []*HostInfo{}
		globalScanState.mu.Unlock()

		_, ipnet, err := net.ParseCIDR(targetSubnet)
		if err != nil {
			globalScanState.mu.Lock()
			globalScanState.isScanning = false
			globalScanState.mu.Unlock()
			return
		}

		var ips []string
		for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}

		globalScanState.mu.Lock()
		globalScanState.totalHosts = len(ips)
		globalScanState.mu.Unlock()
		scanHostsWithTUIUpdates(ips, startPort, endPort, timeout, customPorts, ipsOnly)

		globalScanState.mu.Lock()
		globalScanState.isScanning = false
		globalScanState.mu.Unlock()
	}()
}

func scanHostsWithTUIUpdates(ips []string, startPort, endPort int, timeout time.Duration, customPorts []int, ipsOnly bool) {
	hostWorkers := 10
	portWorkers := 100
	sem := make(chan struct{}, hostWorkers)
	var wg sync.WaitGroup
	ctx := context.Background()

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}

		go func(hostIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			globalScanState.mu.Lock()
			globalScanState.currentHost = hostIP
			globalScanState.mu.Unlock()
			hostInfo := scanHostCustom(ctx, hostIP, startPort, endPort, timeout, portWorkers, customPorts, ipsOnly)

			globalScanState.mu.Lock()
			globalScanState.hostsScanned++
			if hostInfo.IsReachable {
				globalScanState.activeHosts++
				globalScanState.openPorts += len(hostInfo.Services)

				globalScanState.results = append(globalScanState.results, hostInfo)

				sort.Slice(globalScanState.results, func(i, j int) bool {
					return ipToInt(globalScanState.results[i].IP) < ipToInt(globalScanState.results[j].IP)
				})
			}
			globalScanState.mu.Unlock()

		}(ip)
	}

	wg.Wait()
}
