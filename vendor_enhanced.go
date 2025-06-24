package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var vendorCache = make(map[string]string)
var ieeeOUICache = make(map[string]string)
var vendorMutex sync.RWMutex
var ouiMutex sync.RWMutex
var cacheFile = "vendor_cache.json"
var ouiCacheFile = "ieee_oui_cache.json"
var ouiDatabaseFile = "oui.txt"

func loadVendorCache() {
	vendorMutex.Lock()
	defer vendorMutex.Unlock()

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &vendorCache)
}

func loadIEEEOUICache() {
	ouiMutex.Lock()
	defer ouiMutex.Unlock()

	data, err := os.ReadFile(ouiCacheFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &ieeeOUICache)
}

func saveVendorCache() {
	vendorMutex.RLock()
	cacheData := make(map[string]string)
	maps.Copy(cacheData, vendorCache)
	vendorMutex.RUnlock()

	data, _ := json.MarshalIndent(cacheData, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
}

func saveIEEEOUICache() {
	ouiMutex.RLock()
	cacheData := make(map[string]string)
	maps.Copy(cacheData, ieeeOUICache)
	ouiMutex.RUnlock()

	data, _ := json.MarshalIndent(cacheData, "", "  ")
	os.WriteFile(ouiCacheFile, data, 0644)
}

func getEnhancedVendor(mac string) string {
	if len(mac) < 8 {
		return ""
	}

	oui := strings.ToUpper(mac[:8])
	vendorMutex.RLock()
	if vendor, exists := vendorCache[oui]; exists {
		vendorMutex.RUnlock()
		return vendor
	}
	vendorMutex.RUnlock()

	vendorMutex.RLock()
	vendorCacheEmpty := len(vendorCache) == 0
	vendorMutex.RUnlock()

	if vendorCacheEmpty {
		loadVendorCache()
	}

	ouiMutex.RLock()
	ouiCacheEmpty := len(ieeeOUICache) == 0
	ouiMutex.RUnlock()

	if ouiCacheEmpty {
		loadIEEEOUICache()
		ouiMutex.RLock()
		stillEmpty := len(ieeeOUICache) == 0
		ouiMutex.RUnlock()
		if stillEmpty {
			downloadAndParseIEEEOUI()
		}
	}

	ouiMutex.RLock()
	vendor, exists := ieeeOUICache[oui]
	ouiMutex.RUnlock()

	if exists {
		vendorMutex.Lock()
		vendorCache[oui] = vendor
		vendorMutex.Unlock()
		return vendor
	}

	vendor = getLocalVendor(mac)
	if vendor != "" && vendor != "Unknown" {
		vendorMutex.Lock()
		vendorCache[oui] = vendor
		vendorMutex.Unlock()
		return vendor
	}

	vendor = getOnlineVendor(oui)
	if vendor != "" {
		vendorMutex.Lock()
		vendorCache[oui] = vendor
		vendorMutex.Unlock()
		go saveVendorCache()
		return vendor
	}

	vendor = "Unknown"
	vendorMutex.Lock()
	vendorCache[oui] = vendor
	vendorMutex.Unlock()
	return vendor
}

func getLocalVendor(mac string) string {
	if len(mac) < 8 {
		return ""
	}

	vendors := getComprehensiveVendorDB()

	oui := strings.ToUpper(mac[:8])
	if vendor, exists := vendors[oui]; exists {
		return vendor
	}

	for prefix, vendor := range vendors {
		if strings.HasPrefix(strings.ToUpper(mac), prefix) {
			return vendor
		}
	}

	return "Unknown"
}

func getOnlineVendor(oui string) string {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	url := fmt.Sprintf("https://api.macvendors.com/%s", strings.ReplaceAll(oui, ":", ""))

	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ""
		}

		vendor := strings.TrimSpace(string(body))
		if vendor != "" && !strings.Contains(vendor, "error") {
			return vendor
		}
	}

	return ""
}

func getComprehensiveVendorDB() map[string]string {
	return map[string]string{
		"00:50:56": "VMware",
		"00:0C:29": "VMware",
		"00:05:69": "VMware",
		"08:00:27": "VirtualBox",
		"52:54:00": "QEMU",
		"00:16:3E": "Xen",
		"00:03:FF": "Microsoft Hyper-V",

		"00:23:24": "Apple",
		"00:26:B9": "Apple",
		"3C:07:54": "Apple",
		"B8:E8:56": "Apple",
		"A4:C3:61": "Apple",
		"BC:52:B7": "Apple",
		"9C:FC:E8": "Apple",
		"78:7B:8A": "Apple",
		"98:FE:94": "Apple",
		"AC:BC:32": "Apple",
		"F0:18:98": "Apple",
		"1C:AB:A7": "Apple",
		"E8:9A:8F": "Apple",
		"28:CF:E9": "Apple",
		"98:B8:E3": "Apple",
		"34:AB:37": "Apple",
		"64:BC:0C": "Apple",
		"5C:97:F3": "Apple",
		"C0:C9:76": "Apple",
		"70:DE:E2": "Apple",
		"88:1D:FC": "Apple",
		"8C:2D:AA": "Apple",
		"48:60:BC": "Apple",
		"D8:96:95": "Apple",
		"B0:CA:68": "Apple",
		"8C:85:90": "Apple",
		"E0:F8:47": "Apple",
		"FC:E9:98": "Apple",
		"DC:56:E7": "Apple",
		"2C:BE:08": "Apple",
		"78:FD:94": "Apple",
		"80:92:9F": "Apple",
		"C8:69:CD": "Apple",
		"7C:6D:62": "Apple",
		"58:55:CA": "Apple",
		"3C:22:FB": "Apple",
		"EC:35:86": "Apple",
		"F4:37:B7": "Apple",
		"90:9C:4A": "Apple",
		"68:96:7B": "Apple",
		"84:38:35": "Apple",
		"04:E5:36": "Apple",
		"C4:8E:8F": "Apple",
		"14:7D:DA": "Apple",
		"28:E0:2C": "Apple",
		"A8:6D:AA": "Apple",
		"68:FE:F7": "Apple",
		"C8:2A:14": "Apple",
		"04:52:C7": "Apple",
		"20:78:F0": "Apple",
		"30:90:AB": "Apple",
		"60:F4:45": "Apple",
		"90:72:40": "Apple",
		"B4:F0:AB": "Apple",
		"C4:B3:01": "Apple",
		"D4:61:9D": "Apple",
		"E0:B9:BA": "Apple",
		"F8:1E:DF": "Apple",
		"0C:74:C2": "Apple",
		"18:34:51": "Apple",
		"24:A0:74": "Apple",
		"30:F7:C5": "Apple",
		"3C:D0:F8": "Apple",
		"48:A1:95": "Apple",
		"54:26:96": "Apple",
		"60:C5:47": "Apple",
		"6C:72:20": "Apple",
		"78:31:C1": "Apple",
		"84:85:06": "Apple",
		"90:B0:ED": "Apple",
		"9C:20:7B": "Apple",
		"A8:20:66": "Apple",
		"B4:18:D1": "Apple",
		"C0:56:27": "Apple",
		"CC:29:F5": "Apple",
		"D8:30:62": "Apple",
		"E4:CE:8F": "Apple",
		"F0:99:BF": "Apple",
		"FC:25:3F": "Apple",

		"00:1B:21": "Intel",
		"00:23:14": "Intel",
		"D4:BE:D9": "Intel",
		"AC:72:89": "Intel",
		"94:65:9C": "Intel",
		"A0:A8:CD": "Intel",
		"B4:96:91": "Intel",
		"00:90:27": "Intel",
		"00:E0:18": "Intel",
		"F4:06:69": "Intel",
		"00:13:E8": "Intel",
		"00:21:86": "Intel",
		"34:13:E8": "Intel",
		"18:56:80": "Intel",
		"E4:F4:C6": "Intel",
		"48:51:B7": "Intel",
		"78:92:9C": "Intel",
		"00:19:D1": "Intel",
		"00:24:D7": "Intel",
		"00:1F:3C": "Intel",
		"7C:7A:91": "Intel",
		"00:15:00": "Intel",
		"00:16:E3": "Intel",
		"00:1A:A0": "Intel",
		"00:1E:68": "Intel",
		"00:22:FB": "Intel",
		"00:25:64": "Intel",
		"3C:A9:F4": "Intel",
		"84:3A:4B": "Intel",
		"C4:34:6B": "Intel",
		"E0:DB:55": "Intel",
		"00:02:B3": "Intel",
		"00:07:E9": "Intel",
		"00:0E:0C": "Intel",
		"00:12:F0": "Intel",
		"00:18:DE": "Intel",
		"00:1C:BF": "Intel",
		"00:20:E0": "Intel",
		"00:24:81": "Intel",
		"68:05:CA": "Intel",
		"8C:A9:82": "Intel",
		"A4:BA:DB": "Intel",
		"00:AA:00": "Intel",
		"00:A0:C9": "Intel",
		"08:11:96": "Intel",

		"DC:A6:32": "Raspberry Pi Foundation",
		"B8:27:EB": "Raspberry Pi",
		"E4:5F:01": "Raspberry Pi",
		"28:CD:C1": "Raspberry Pi",
		"D8:3A:DD": "Raspberry Pi",

		"00:1D:25": "Samsung",
		"BC:14:01": "Samsung",
		"38:AA:3C": "Samsung",
		"94:8B:C1": "Samsung",
		"F4:7B:5E": "Samsung",
		"3C:28:6D": "Samsung",
		"C8:19:F7": "Samsung",
		"90:18:7C": "Samsung",
		"8C:77:12": "Samsung",
		"20:64:32": "Samsung",
		"68:EB:C5": "Samsung",
		"CC:03:FA": "Samsung",
		"EC:1F:72": "Samsung",
		"78:F8:82": "Samsung",
		"40:B0:FA": "Samsung",
		"74:45:CE": "Samsung",
		"D0:59:E4": "Samsung",
		"50:CC:F8": "Samsung",
		"A0:02:DC": "Samsung",
		"E8:50:8B": "Samsung",
		"1C:62:B8": "Samsung",
		"30:07:4D": "Samsung",
		"44:4E:6D": "Samsung",
		"58:67:1A": "Samsung",
		"6C:88:14": "Samsung",
		"80:A9:97": "Samsung",
		"94:E9:79": "Samsung",
		"A8:F2:74": "Samsung",
		"BC:25:E0": "Samsung",
		"D0:7E:35": "Samsung",
		"E4:A4:71": "Samsung",
		"F8:D0:BD": "Samsung",

		"F4:F5:D8": "Google",
		"DA:A1:19": "Google",
		"6C:AD:F8": "Google",
		"AC:37:43": "Google",
		"40:5B:D8": "Google",
		"F8:8F:CA": "Google",
		"44:07:0B": "Google",
		"E0:B7:B1": "Google",
		"54:60:09": "Google",
		"68:C6:3A": "Google",
		"7C:C3:A1": "Google",
		"90:E6:BA": "Google",
		"A4:DA:32": "Google",
		"B8:AD:28": "Google",
		"CC:3A:61": "Google",

		"00:FC:8B": "Amazon",
		"F0:27:2D": "Amazon",
		"68:37:E9": "Amazon",
		"AC:63:BE": "Amazon",
		"50:DC:E7": "Amazon",
		"84:D6:D0": "Amazon",
		"F8:04:2E": "Amazon",
		"CC:9E:A2": "Amazon",
		"18:74:2E": "Amazon",
		"2C:F0:5D": "Amazon",
		"40:B4:CD": "Amazon",
		"54:E4:3A": "Amazon",
		"68:54:ED": "Amazon",
		"7C:BB:8A": "Amazon",
		"90:E7:C4": "Amazon",
		"A4:50:46": "Amazon",
		"B8:8B:83": "Amazon",
		"CC:F7:35": "Amazon",
		"E0:AA:96": "Amazon",
		"F4:39:09": "Amazon",

		"00:50:F2": "Microsoft",
		"0C:8B:FD": "Microsoft",
		"18:60:24": "Microsoft",
		"20:82:C0": "Microsoft",
		"60:45:BD": "Microsoft",
		"7C:1E:52": "Microsoft",
		"A0:48:1C": "Microsoft",
		"34:17:EB": "Microsoft",
		"48:2A:E3": "Microsoft",
		"5C:79:C2": "Microsoft",
		"70:4C:A5": "Microsoft",
		"84:38:38": "Microsoft",
		"98:5F:D3": "Microsoft",
		"AC:22:0B": "Microsoft",
		"C0:38:F9": "Microsoft",
		"D4:81:D7": "Microsoft",
		"E8:92:A4": "Microsoft",
		"FC:15:B4": "Microsoft",

		"00:1B:FC": "ASUS",
		"BC:AE:C5": "ASUS",
		"D8:50:E6": "ASUS",
		"2C:56:DC": "ASUS",
		"AC:9E:17": "ASUS",
		"04:D4:C4": "ASUS",
		"70:4D:7B": "ASUS",
		"38:D5:47": "ASUS",
		"50:46:5D": "ASUS",
		"1C:87:2C": "ASUS",
		"30:5A:3A": "ASUS",
		"44:E9:DD": "ASUS",
		"58:11:22": "ASUS",
		"6C:62:6D": "ASUS",
		"80:91:33": "ASUS",
		"94:DE:80": "ASUS",
		"A8:5E:45": "ASUS",
		"BC:EE:7B": "ASUS",
		"D0:17:C2": "ASUS",
		"E4:70:B8": "ASUS",
		"F8:32:E4": "ASUS",

		"00:23:CD": "TP-Link",
		"14:CC:20": "TP-Link",
		"50:C7:BF": "TP-Link",
		"A4:2B:B0": "TP-Link",
		"C4:E9:84": "TP-Link",
		"EC:08:6B": "TP-Link",
		"F4:EC:38": "TP-Link",
		"1C:61:B4": "TP-Link",
		"98:DA:C4": "TP-Link",
		"B0:4E:26": "TP-Link",
		"4C:ED:FB": "TP-Link",
		"E8:DE:27": "TP-Link",
		"68:FF:7B": "TP-Link",
		"84:16:F9": "TP-Link",
		"AC:84:C6": "TP-Link",
		"18:A6:F7": "TP-Link",
		"2C:3E:CF": "TP-Link",
		"40:5D:82": "TP-Link",
		"54:A0:50": "TP-Link",
		"68:1C:A2": "TP-Link",
		"7C:8B:CA": "TP-Link",
		"90:F6:52": "TP-Link",
		"A4:91:B1": "TP-Link",
		"B8:A3:86": "TP-Link",
		"CC:32:E5": "TP-Link",
		"E0:28:6D": "TP-Link",
		"F4:92:BF": "TP-Link",

		"00:05:5D": "D-Link",
		"00:0D:88": "D-Link",
		"00:15:E9": "D-Link",
		"00:17:9A": "D-Link",
		"00:19:5B": "D-Link",
		"00:1B:11": "D-Link",
		"00:1C:F0": "D-Link",
		"00:1E:58": "D-Link",
		"00:21:91": "D-Link",
		"00:22:B0": "D-Link",
		"00:24:01": "D-Link",
		"00:26:5A": "D-Link",
		"14:D6:4D": "D-Link",
		"C8:D3:A3": "D-Link",
		"CC:B2:55": "D-Link",
		"28:10:7B": "D-Link",
		"3C:1E:04": "D-Link",
		"64:70:02": "D-Link",
		"78:54:2E": "D-Link",
		"8C:BE:BE": "D-Link",
		"A0:AB:1B": "D-Link",
		"B4:B5:2F": "D-Link",
		"C8:BE:19": "D-Link",
		"DC:53:7C": "D-Link",
		"F0:7D:68": "D-Link",

		"00:09:5B": "Netgear",
		"00:0F:B5": "Netgear",
		"00:14:6C": "Netgear",
		"00:18:4D": "Netgear",
		"00:1B:2F": "Netgear",
		"00:1E:2A": "Netgear",
		"00:22:3F": "Netgear",
		"00:24:B2": "Netgear",
		"00:26:F2": "Netgear",
		"20:4E:7F": "Netgear",
		"2C:30:33": "Netgear",
		"A0:04:60": "Netgear",
		"C0:3F:0E": "Netgear",
		"E0:46:9A": "Netgear",
		"04:A1:51": "Netgear",
		"18:1B:EB": "Netgear",
		"2C:B0:5D": "Netgear",
		"40:16:7E": "Netgear",
		"54:04:A6": "Netgear",
		"90:A4:DE": "Netgear",
		"A4:2B:8C": "Netgear",
		"B8:C7:5D": "Netgear",
		"CC:40:D0": "Netgear",
		"E0:91:F5": "Netgear",

		"00:06:25": "Linksys",
		"00:12:17": "Linksys",
		"00:13:10": "Linksys",
		"00:14:BF": "Linksys",
		"00:16:B6": "Linksys",
		"00:18:39": "Linksys",
		"00:1A:70": "Linksys",
		"00:1C:10": "Linksys",
		"00:1D:7E": "Linksys",
		"00:1E:E5": "Linksys",
		"00:20:A6": "Linksys",
		"00:21:29": "Linksys",
		"00:22:6B": "Linksys",
		"00:23:69": "Linksys",
		"00:25:9C": "Linksys",
		"48:F8:B3": "Linksys",
		"C4:41:1E": "Linksys",
		"14:91:82": "Linksys",
		"20:AA:4B": "Linksys",
		"28:C6:8E": "Linksys",
		"50:06:04": "Linksys",
		"64:09:80": "Linksys",
		"78:8A:20": "Linksys",
		"8C:04:BA": "Linksys",
		"A0:21:B7": "Linksys",
		"B4:75:0E": "Linksys",
		"C8:D7:19": "Linksys",
		"DC:EF:09": "Linksys",
		"F0:92:1C": "Linksys",

		"00:04:20": "Slim Devices",
		"00:04:61": "Linksys Group",
		"00:06:91": "Cacheflow",
		"00:07:32": "Cisco Aironet",
		"00:08:5B": "Compex",
		"00:09:2B": "Quantenna",
		"00:0A:27": "Apple",
		"00:0B:86": "Arima",
		"00:0C:6E": "SAT-Tech",
		"00:0D:54": "Dell",
		"00:0E:35": "Cisco Systems",
		"00:0F:1F": "AzureWave",
		"00:10:18": "Broadcom",
		"00:11:09": "Zoom",
		"00:12:0E": "Globalstar",
		"00:13:02": "Cisco Systems",
		"00:14:A4": "Cisco Systems",
		"00:15:62": "Cisco Systems",
		"00:16:01": "Ubee",
		"00:17:13": "Belkin",
		"00:18:8B": "Compex",
		"00:19:07": "Cisco Systems",
		"00:1A:1E": "Giga",
		"00:1B:63": "Apple",
		"00:1C:B3": "Apple",
		"00:1D:4F": "Apple",
		"00:1E:52": "Apple",
		"00:1F:5B": "Apple",
		"00:20:91": "Phast",
		"00:21:E9": "Dell",
		"00:22:58": "Cisco Systems",
		"00:23:DF": "Apple",
		"00:24:36": "Cisco Systems",
		"00:25:00": "Apple",
		"00:26:08": "Apple",
		"00:27:10": "Motorola",
		"00:50:BA": "Cisco Systems",
		"00:60:2F": "Cisco Systems",
		"00:90:4B": "Gemtek",
		"00:A0:C6": "Qualcomm",
		"00:B0:52": "Cisco Systems",
		"00:C0:49": "US Robotics",
		"00:D0:01": "Cisco Systems",
		"00:E0:14": "Cisco Systems",
		"08:00:07": "Apple",
		"10:40:F3": "Dell",
		"14:10:9F": "Cisco Systems",
		"18:03:73": "Cisco Systems",
		"1C:6F:65": "Foxconn",
		"20:02:AF": "Apple",
		"24:A4:3C": "Apple",
		"28:37:37": "Apple",
		"2C:B4:3A": "Apple",
		"30:65:EC": "Apple",
		"34:15:9E": "Apple",
		"38:C9:86": "Apple",
		"3C:15:C2": "Apple",
		"40:30:04": "Apple",
		"44:2A:60": "Apple",
		"48:3B:38": "Apple",
		"4C:56:9D": "Apple",
		"50:DE:06": "Apple",
		"58:B0:35": "Apple",
		"5C:F9:38": "Apple",
		"60:33:4B": "Apple",
		"64:20:0C": "Apple",
		"68:AB:1E": "Apple",
		"6C:19:8F": "Apple",
		"70:73:CB": "Apple",
		"74:E2:F5": "Apple",
		"78:4F:43": "Apple",
		"7C:C7:09": "Apple",
		"80:E6:50": "Apple",
		"84:A1:34": "Apple",
		"88:63:DF": "Apple",
		"8C:7C:92": "Apple",
		"90:84:0D": "Apple",
		"94:E6:F7": "Apple",
		"98:5A:EB": "Apple",
		"9C:04:EB": "Apple",
		"A0:99:9B": "Apple",
		"A4:D1:8C": "Apple",
		"A8:96:8A": "Apple",
		"AC:87:A3": "Apple",
		"B0:65:BD": "Apple",
		"B4:F7:A1": "Apple",
		"BC:67:1C": "Apple",
		"C0:84:7A": "Apple",
		"C4:2C:03": "Apple",
		"C8:E0:EB": "Apple",
		"CC:25:EF": "Apple",
		"D0:03:4B": "Apple",
		"D4:9A:20": "Apple",
		"D8:BB:2C": "Apple",
		"DC:2B:2A": "Apple",
		"E0:33:8E": "Apple",
		"E4:8B:7F": "Apple",
		"E8:40:F2": "Apple",
		"EC:8C:A2": "Apple",
		"F0:B4:79": "Apple",
		"F4:1B:A1": "Apple",
		"F8:A9:D0": "Apple",
		"FC:FC:48": "Apple",
	}
}

func downloadAndParseIEEEOUI() {
	fmt.Println("Downloading IEEE OUI database...")

	if info, err := os.Stat(ouiDatabaseFile); err == nil {
		if time.Since(info.ModTime()) < 30*24*time.Hour {
			if parseLocalOUIFile() {
				return
			}
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get("https://standards-oui.ieee.org/oui/oui.txt")
	if err != nil {
		fmt.Printf("Failed to download OUI database: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Failed to download OUI database: HTTP %d\n", resp.StatusCode)
		return
	}

	file, err := os.Create(ouiDatabaseFile)
	if err != nil {
		fmt.Printf("Failed to create local OUI file: %v\n", err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read OUI database: %v\n", err)
		return
	}

	file.Write(content)

	parseOUIContent(string(content))

	saveIEEEOUICache()

	fmt.Printf("IEEE OUI database downloaded and cached (%d vendors)\n", len(ieeeOUICache))
}

func parseLocalOUIFile() bool {
	content, err := os.ReadFile(ouiDatabaseFile)
	if err != nil {
		return false
	}

	parseOUIContent(string(content))
	saveIEEEOUICache()

	return len(ieeeOUICache) > 0
}

func parseOUIContent(content string) {
	ouiRegex := regexp.MustCompile(`^([0-9A-F]{2}-[0-9A-F]{2}-[0-9A-F]{2})\s+\(hex\)\s+(.+)$`)

	tempCache := make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := ouiRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			oui := strings.ReplaceAll(matches[1], "-", ":")
			vendor := strings.TrimSpace(matches[2])

			vendor = cleanVendorName(vendor)

			tempCache[oui] = vendor
		}
	}

	ouiMutex.Lock()
	maps.Copy(ieeeOUICache, tempCache)
	ouiMutex.Unlock()
}

func cleanVendorName(vendor string) string {
	vendor = strings.TrimSpace(vendor)

	suffixes := []string{", Inc.", " Inc.", ", Corp.", " Corp.", ", Ltd.", " Ltd.",
		", LLC", " LLC", ", Co.", " Co.", ", Inc", " Inc", ", Corp", " Corp"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(vendor, suffix) {
			vendor = strings.TrimSpace(strings.TrimSuffix(vendor, suffix))
			break
		}
	}

	if len(vendor) > 50 {
		vendor = vendor[:50] + "..."
	}

	return vendor
}
