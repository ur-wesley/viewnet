package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
)

type ServiceInfo struct {
	Port         int
	Protocol     string
	Service      string
	Version      string
	Banner       string
	IsOpen       bool
	ResponseTime time.Duration
}

type HostInfo struct {
	IP           string
	MAC          string
	Vendor       string
	Hostname     string
	Services     []ServiceInfo
	IsReachable  bool
	ResponseTime time.Duration
}

type ScanProgress struct {
	CurrentHost  string
	HostsScanned int
	TotalHosts   int
	ActiveHosts  int
	OpenPorts    int
	StartTime    time.Time
	EndTime      time.Time
}

type scanState int

const (
	stateScanning scanState = iota
	stateComplete
)

type UIModel struct {
	state           scanState
	progress        progress.Model
	spinner         spinner.Model
	searchInput     textinput.Model
	scanInfo        ScanProgress
	results         []*HostInfo
	filteredResults []*HostInfo
	targetSubnet    string
	startPort       int
	endPort         int
	timeout         int
	customPorts     []int
	ipsOnly         bool
	quitting        bool
	err             error
	scrollOffset    int
	viewHeight      int
	searchFocused   bool
	searchOnlyMode  bool
	windowWidth     int
	windowHeight    int
	scanEndTime     time.Time
}

type scanErrorMsg struct {
	err error
}

type pollMsg struct{}
