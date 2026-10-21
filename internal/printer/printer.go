package printer

import (
	"fmt"
	"strings"

	"passbook-bridge/internal/config"
)

// Printer mendefinisikan interface umum untuk berkomunikasi dengan printer dot-matrix
type Printer interface {
	// Name mengembalikan nama atau identitas printer yang aktif
	Name() string
	// Ping memeriksa apakah perangkat printer terhubung dan siap
	Ping() (connected bool, err error)
	// Print mengirimkan byte raw ESC/P ke printer
	Print(data []byte) error
	// Close menutup port atau koneksi ke printer
	Close() error
}

// NewPrinter adalah factory function untuk membuat instance printer berdasarkan konfigurasi
func NewPrinter(cfg *config.Config) (Printer, error) {
	printerType := strings.ToLower(strings.TrimSpace(cfg.PrinterType))

	switch printerType {
	case "mock", "console", "test":
		return NewMockPrinter(cfg.PrinterName), nil
	case "serial", "com":
		return NewSerialPrinter(cfg.SerialPort, cfg.BaudRate, cfg.PrinterName), nil
	case "spooler", "windows":
		return NewSpoolerPrinter(cfg.PrinterName)
	default:
		return nil, fmt.Errorf("printer_type '%s' tidak didukung (pilihan: spooler, serial, mock)", cfg.PrinterType)
	}
}
