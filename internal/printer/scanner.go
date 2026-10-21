package printer

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"go.bug.st/serial"
)

// DeviceInfo merepresentasikan detail printer fisik, driver spooler, atau port serial yang terdeteksi pada sistem
type DeviceInfo struct {
	Type        string `json:"type"`        // "spooler" atau "serial"
	Name        string `json:"name"`        // e.g. "EPSON PLQ-20 ESC/P2" atau "COM3"
	Port        string `json:"port"`        // e.g. "COM3", "USB001"
	DriverName  string `json:"driver_name"` // e.g. "Generic / Text Only"
	IsPassbook  bool   `json:"is_passbook"` // true jika nama/driver cocok dengan printer buku tabungan
	Description string `json:"description"`
}

// Keyword printer passbook & dot matrix populer perbankan
var passbookKeywords = []string{
	"plq", "pr2", "olivetti", "compuprint", "sp40", "tally", "dascom", "wincor", "4915",
	"ibm", "passbook", "dot matrix", "text only", "generic", "lq-", "lx-", "fx-", "epson",
	"serial", "com", "ftdi", "ch340", "prolific",
}

// IsPassbookDevice mengecek apakah nama perangkat mengindikasikan printer passbook
func IsPassbookDevice(name, driver, port string) bool {
	combined := strings.ToLower(name + " " + driver + " " + port)
	for _, kw := range passbookKeywords {
		if strings.Contains(combined, kw) {
			return true
		}
	}
	return false
}

// ScanAllDevices mendeteksi printer spooler dan port serial yang terpasang di sistem
func ScanAllDevices() []DeviceInfo {
	var devices []DeviceInfo

	// 1. Pindai printer spooler OS
	spoolers := scanSpoolerPrinters()
	devices = append(devices, spoolers...)

	// 2. Pindai port serial fisik/virtual
	ports, err := serial.GetPortsList()
	if err == nil {
		for _, p := range ports {
			isPb := strings.Contains(strings.ToUpper(p), "COM") || strings.Contains(p, "ttyUSB") || strings.Contains(p, "ttyACM")
			devices = append(devices, DeviceInfo{
				Type:        "serial",
				Name:        fmt.Sprintf("Port Serial %s", p),
				Port:        p,
				DriverName:  "Direct Serial / Virtual COM",
				IsPassbook:  isPb,
				Description: fmt.Sprintf("Koneksi kabel Serial COM / USB RS232 pada port %s", p),
			})
		}
	}

	// 3. Urutkan perangkat: Rekomendasi Passbook printer di paling atas
	var passbooks []DeviceInfo
	var others []DeviceInfo
	for _, d := range devices {
		if d.IsPassbook {
			passbooks = append(passbooks, d)
		} else {
			others = append(others, d)
		}
	}

	return append(passbooks, others...)
}

// scanSpoolerPrinters membaca daftar printer dari Windows Print Spooler atau CUPS Linux
func scanSpoolerPrinters() []DeviceInfo {
	var results []DeviceInfo

	if runtime.GOOS == "windows" {
		// Menggunakan PowerShell CimInstance Win32_Printer
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			"Get-CimInstance Win32_Printer | Select-Object Name, PortName, DriverName | ConvertTo-Json")
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			type psPrinter struct {
				Name       string `json:"Name"`
				PortName   string `json:"PortName"`
				DriverName string `json:"DriverName"`
			}

			// Coba parse array of printers
			var list []psPrinter
			if err := json.Unmarshal(out, &list); err == nil {
				for _, p := range list {
					if strings.TrimSpace(p.Name) == "" {
						continue
					}
					isPb := IsPassbookDevice(p.Name, p.DriverName, p.PortName)
					results = append(results, DeviceInfo{
						Type:        "spooler",
						Name:        p.Name,
						Port:        p.PortName,
						DriverName:  p.DriverName,
						IsPassbook:  isPb,
						Description: fmt.Sprintf("Windows Spooler Printer (Driver: %s, Port: %s)", p.DriverName, p.PortName),
					})
				}
				return results
			}

			// Fallback jika hanya 1 printer (JSON mengembalikan objek tunggal)
			var single psPrinter
			if err := json.Unmarshal(out, &single); err == nil && strings.TrimSpace(single.Name) != "" {
				isPb := IsPassbookDevice(single.Name, single.DriverName, single.PortName)
				results = append(results, DeviceInfo{
					Type:        "spooler",
					Name:        single.Name,
					Port:        single.PortName,
					DriverName:  single.DriverName,
					IsPassbook:  isPb,
					Description: fmt.Sprintf("Windows Spooler Printer (Driver: %s, Port: %s)", single.DriverName, single.PortName),
				})
				return results
			}
		}

		// Fallback WMIC jika PowerShell gagal
		wmicCmd := exec.Command("cmd", "/c", "wmic printer get name,portname,drivername /format:csv")
		wmicOut, err := wmicCmd.Output()
		if err == nil {
			lines := strings.Split(string(wmicOut), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				parts := strings.Split(line, ",")
				if len(parts) >= 4 && parts[0] != "Node" {
					driver := strings.TrimSpace(parts[1])
					name := strings.TrimSpace(parts[2])
					port := strings.TrimSpace(parts[3])
					if name != "" && name != "Name" {
						isPb := IsPassbookDevice(name, driver, port)
						results = append(results, DeviceInfo{
							Type:        "spooler",
							Name:        name,
							Port:        port,
							DriverName:  driver,
							IsPassbook:  isPb,
							Description: fmt.Sprintf("Windows Spooler Printer (Driver: %s, Port: %s)", driver, port),
						})
					}
				}
			}
		}
	} else {
		// Linux / macOS via lpstat
		cmd := exec.Command("lpstat", "-p")
		out, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "printer ") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						pName := parts[1]
						isPb := IsPassbookDevice(pName, "", "")
						results = append(results, DeviceInfo{
							Type:        "spooler",
							Name:        pName,
							Port:        "CUPS",
							DriverName:  "CUPS Queue",
							IsPassbook:  isPb,
							Description: fmt.Sprintf("CUPS Linux Printer Queue (%s)", pName),
						})
					}
				}
			}
		}
	}

	return results
}
