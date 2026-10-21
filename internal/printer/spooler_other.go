//go:build !windows

package printer

import (
	"fmt"
)

// SpoolerPrinter fallback untuk OS selain Windows
type SpoolerPrinter struct {
	printerName string
}

func NewSpoolerPrinter(printerName string) (Printer, error) {
	return &SpoolerPrinter{
		printerName: printerName,
	}, nil
}

func (s *SpoolerPrinter) Name() string {
	return s.printerName
}

func (s *SpoolerPrinter) Ping() (bool, error) {
	return false, fmt.Errorf("mode 'spooler' (Win32 Spooler) hanya didukung di sistem operasi Windows. Gunakan mode 'mock' atau 'serial' untuk Linux/macOS")
}

func (s *SpoolerPrinter) Print(data []byte) error {
	return fmt.Errorf("mode 'spooler' (Win32 Spooler) hanya didukung di sistem operasi Windows")
}

func (s *SpoolerPrinter) Close() error {
	return nil
}
