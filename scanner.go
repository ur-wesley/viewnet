package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var commonPorts = map[int]string{
	21:   "FTP",
	22:   "SSH",
	23:   "Telnet",
	25:   "SMTP",
	53:   "DNS",
	80:   "HTTP",
	110:  "POP3",
	143:  "IMAP",
	443:  "HTTPS",
	993:  "IMAPS",
	995:  "POP3S",
	1433: "MSSQL",
	1521: "Oracle",
	3306: "MySQL",
	3389: "RDP",
	5432: "PostgreSQL",
	5900: "VNC",
	8080: "HTTP-Alt",
	8443: "HTTPS-Alt",
}

func scanPortNew(ctx context.Context, ip string, port int, timeout time.Duration) (*ServiceInfo, error) {
	address := fmt.Sprintf("%s:%d", ip, port)
	start := time.Now()

	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", address)
	responseTime := time.Since(start)

	if err != nil {
		return &ServiceInfo{
			Port:         port,
			Protocol:     "TCP",
			Service:      getServiceNameNew(port),
			IsOpen:       false,
			ResponseTime: responseTime,
		}, err
	}
	defer conn.Close()

	banner := getBannerNew(conn, port, timeout)
	service := getServiceNameNew(port)
	version := extractVersionNew(banner, service)

	return &ServiceInfo{
		Port:         port,
		Protocol:     "TCP",
		Service:      service,
		Version:      version,
		Banner:       banner,
		IsOpen:       true,
		ResponseTime: responseTime,
	}, nil
}

func getServiceNameNew(port int) string {
	if service, exists := commonPorts[port]; exists {
		return service
	}
	return "Unknown"
}

func getBannerNew(conn net.Conn, port int, timeout time.Duration) string {
	conn.SetReadDeadline(time.Now().Add(timeout))

	switch port {
	case 80, 8080:
		conn.Write([]byte("GET / HTTP/1.1\r\nHost: \r\n\r\n"))
	case 21:
	case 22:
	case 25:
	default:
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil || n == 0 {
		return ""
	}

	banner := strings.TrimSpace(string(buffer[:n]))
	re := regexp.MustCompile(`[[:print:]]+`)
	cleanBanner := strings.Join(re.FindAllString(banner, -1), " ")

	if len(cleanBanner) > 100 {
		cleanBanner = cleanBanner[:100] + "..."
	}

	return cleanBanner
}

func extractVersionNew(banner, service string) string {
	if banner == "" {
		return ""
	}

	patterns := map[string]*regexp.Regexp{
		"HTTP": regexp.MustCompile(`(?i)(Apache|nginx|IIS)/([0-9.]+)`),
		"SSH":  regexp.MustCompile(`(?i)OpenSSH[_\s]([0-9.]+)`),
		"FTP":  regexp.MustCompile(`(?i)(FileZilla|vsftpd|ProFTPD)\s+([0-9.]+)`),
		"SMTP": regexp.MustCompile(`(?i)(Postfix|Sendmail|Exchange)\s+([0-9.]+)`),
	}

	if pattern, exists := patterns[service]; exists {
		matches := pattern.FindStringSubmatch(banner)
		if len(matches) > 2 {
			return matches[1] + " " + matches[2]
		}
	}

	genericPattern := regexp.MustCompile(`([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	matches := genericPattern.FindStringSubmatch(banner)
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

func scanHostNew(ctx context.Context, ip string, startPort, endPort int, timeout time.Duration, workers int) *HostInfo {
	hostInfo := &HostInfo{
		IP:       ip,
		Services: make([]ServiceInfo, 0),
	}

	reachable, responseTime := pingHostNew(ip, timeout)
	hostInfo.IsReachable = reachable
	hostInfo.ResponseTime = responseTime

	if !reachable {
		return hostInfo
	}

	hostInfo.Hostname = getHostnameNew(ip)
	hostInfo.MAC, hostInfo.Vendor = getMACAddressNew(ip)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for port := startPort; port <= endPort; port++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			serviceInfo, err := scanPortNew(ctx, ip, p, timeout)
			if err == nil && serviceInfo.IsOpen {
				mu.Lock()
				hostInfo.Services = append(hostInfo.Services, *serviceInfo)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	sort.Slice(hostInfo.Services, func(i, j int) bool {
		return hostInfo.Services[i].Port < hostInfo.Services[j].Port
	})

	return hostInfo
}

func pingHostNew(ip string, timeout time.Duration) (bool, time.Duration) {
	start := time.Now()
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", fmt.Sprintf("%d", int(timeout.Milliseconds())), ip)
	} else {
		timeoutSecs := int(timeout.Seconds())
		if timeoutSecs < 1 {
			timeoutSecs = 1
		}
		cmd = exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", timeoutSecs), ip)
	}
	
	err := cmd.Run()
	responseTime := time.Since(start)

	return err == nil, responseTime
}

func getHostnameNew(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

type PortScanner struct {
	hostWorkers int
	portWorkers int
}

func NewPortScanner(hostWorkers, portWorkers int) *PortScanner {
	return &PortScanner{
		hostWorkers: hostWorkers,
		portWorkers: portWorkers,
	}
}

func (ps *PortScanner) ScanSubnet(subnet string, startPort, endPort int, timeout time.Duration) []*HostInfo {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil
	}

	var ips []string
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	return ps.scanHosts(ips, startPort, endPort, timeout)
}

func (ps *PortScanner) ScanSubnetWithProgress(subnet string, startPort, endPort int, timeout time.Duration, progressChan chan<- ScanProgress) []*HostInfo {
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil
	}

	var ips []string
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	return ps.scanHostsWithProgress(ips, startPort, endPort, timeout, progressChan)
}

func (ps *PortScanner) scanHosts(ips []string, startPort, endPort int, timeout time.Duration) []*HostInfo {
	sem := make(chan struct{}, ps.hostWorkers)
	var wg sync.WaitGroup
	ctx := context.Background()

	results := make(chan *HostInfo, len(ips))

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}

		go func(hostIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			hostInfo := scanHostNew(ctx, hostIP, startPort, endPort, timeout, ps.portWorkers)
			results <- hostInfo
		}(ip)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allHosts []*HostInfo
	for hostInfo := range results {
		allHosts = append(allHosts, hostInfo)
	}

	return allHosts
}

func (ps *PortScanner) scanHostsWithProgress(ips []string, startPort, endPort int, timeout time.Duration, progressChan chan<- ScanProgress) []*HostInfo {
	sem := make(chan struct{}, ps.hostWorkers)
	var wg sync.WaitGroup
	ctx := context.Background()

	var allHosts []*HostInfo
	var hostsLock sync.Mutex

	startTime := time.Now()
	scanned := 0
	activeHosts := 0
	openPorts := 0

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}

		go func(hostIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			hostInfo := scanHostNew(ctx, hostIP, startPort, endPort, timeout, ps.portWorkers)

			hostsLock.Lock()
			allHosts = append(allHosts, hostInfo)
			scanned++
			if hostInfo.IsReachable {
				activeHosts++
				openPorts += len(hostInfo.Services)
			}

			progress := ScanProgress{
				CurrentHost:  hostIP,
				HostsScanned: scanned,
				TotalHosts:   len(ips),
				ActiveHosts:  activeHosts,
				OpenPorts:    openPorts,
				StartTime:    startTime,
			}

			select {
			case progressChan <- progress:
			default:
			}

			hostsLock.Unlock()
		}(ip)
	}

	wg.Wait()
	close(progressChan)

	return allHosts
}

func scanHostCustom(ctx context.Context, ip string, startPort, endPort int, timeout time.Duration, workers int, customPorts []int, ipsOnly bool) *HostInfo {
	hostInfo := &HostInfo{
		IP:       ip,
		Services: make([]ServiceInfo, 0),
	}

	reachable, responseTime := pingHostNew(ip, timeout)
	hostInfo.IsReachable = reachable
	hostInfo.ResponseTime = responseTime

	if !reachable {
		return hostInfo
	}

	hostInfo.Hostname = getHostnameNew(ip)
	hostInfo.MAC, hostInfo.Vendor = getMACAddressNew(ip)

	if ipsOnly {
		return hostInfo
	}

	var portsToScan []int
	if len(customPorts) > 0 {
		portsToScan = customPorts
	} else {
		for port := startPort; port <= endPort; port++ {
			portsToScan = append(portsToScan, port)
		}
	}

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, port := range portsToScan {
		wg.Add(1)
		sem <- struct{}{}

		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			serviceInfo, err := scanPortNew(ctx, ip, p, timeout)
			if err == nil && serviceInfo.IsOpen {
				mu.Lock()
				hostInfo.Services = append(hostInfo.Services, *serviceInfo)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()

	sort.Slice(hostInfo.Services, func(i, j int) bool {
		return hostInfo.Services[i].Port < hostInfo.Services[j].Port
	})

	return hostInfo
}

func getCommonPorts() []int {
	ports := make([]int, 0, len(commonPorts))
	for port := range commonPorts {
		ports = append(ports, port)
	}
	return ports
}
