package printer

import (
	"testing"
)

func TestIsPassbookDevice(t *testing.T) {
	tests := []struct {
		name       string
		driver     string
		port       string
		isPassbook bool
	}{
		{"EPSON PLQ-20 ESC/P2", "Generic / Text Only", "USB001", true},
		{"Olivetti PR2 Plus", "Olivetti Driver", "COM3", true},
		{"Compuprint SP40 Plus", "Compuprint", "COM1", true},
		{"Tally Dascom 1125", "Tally", "LPT1", true},
		{"HP LaserJet Pro M404", "HP PCL6", "192.168.1.50", false},
		{"Canon Pixma G2010", "Canon Inkjet", "USB002", false},
		{"USB Serial Port", "FTDI", "COM3", true},
	}

	for _, tt := range tests {
		got := IsPassbookDevice(tt.name, tt.driver, tt.port)
		if got != tt.isPassbook {
			t.Errorf("IsPassbookDevice(%q, %q, %q) = %v, want %v", tt.name, tt.driver, tt.port, got, tt.isPassbook)
		}
	}
}

func TestScanAllDevices(t *testing.T) {
	devices := ScanAllDevices()
	// ScanAllDevices should not panic
	t.Logf("Detected devices count: %d", len(devices))
}
