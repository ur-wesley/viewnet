package main

import (
	"testing"
	"time"
)

func TestScanProgress(t *testing.T) {
	progress := GetScanProgress()

	if progress.HostsScanned < 0 {
		t.Error("HostsScanned should not be negative")
	}
	if progress.TotalHosts < 0 {
		t.Error("TotalHosts should not be negative")
	}
	if progress.ActiveHosts < 0 {
		t.Error("ActiveHosts should not be negative")
	}
	if progress.OpenPorts < 0 {
		t.Error("OpenPorts should not be negative")
	}
}

func TestScanResults(t *testing.T) {
	results := GetScanResults()

	if results == nil {
		t.Error("GetScanResults() should not return nil")
	}

	for i, host := range results {
		if host == nil {
			t.Errorf("result at index %d is nil", i)
		}
		if host.IP == "" {
			t.Errorf("result at index %d has empty IP", i)
		}
	}
}

func TestIsScanComplete(t *testing.T) {
	complete := IsScanComplete()

	t.Logf("scan complete: %v", complete)
}

func TestScanProgressTiming(t *testing.T) {
	progress := GetScanProgress()

	if !progress.StartTime.IsZero() {
		if progress.StartTime.After(time.Now()) {
			t.Error("start time should not be in the future")
		}

		if time.Since(progress.StartTime) > time.Hour {
			t.Logf("start time seems old: %v", progress.StartTime)
		}
	}

	if !progress.EndTime.IsZero() && !progress.StartTime.IsZero() {
		if progress.EndTime.Before(progress.StartTime) {
			t.Error("end time should be after start time")
		}
	}
}

func TestScanProgressConsistency(t *testing.T) {
	progress := GetScanProgress()

	if progress.ActiveHosts > progress.TotalHosts {
		t.Errorf("active hosts (%d) should not exceed total hosts (%d)",
			progress.ActiveHosts, progress.TotalHosts)
	}

	if progress.HostsScanned > progress.TotalHosts {
		t.Errorf("scanned hosts (%d) should not exceed total hosts (%d)",
			progress.HostsScanned, progress.TotalHosts)
	}

	if progress.OpenPorts > progress.ActiveHosts*1000 {
		t.Errorf("open ports (%d) seems unreasonably high for active hosts (%d)",
			progress.OpenPorts, progress.ActiveHosts)
	}
}

func TestScanStateThreadSafety(t *testing.T) {

	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			GetScanProgress()
			GetScanResults()
			IsScanComplete()
		}()
	}

	for range 10 {
		<-done
	}

}
