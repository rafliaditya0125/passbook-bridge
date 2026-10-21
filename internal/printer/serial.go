package printer

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.bug.st/serial"
)

// GetAvailablePorts mengembalikan daftar port serial fisik/virtual yang terdeteksi pada sistem
func GetAvailablePorts() ([]string, error) {
	return serial.GetPortsList()
}

// SerialPrinter mengimplementasikan antarmuka Printer untuk port Serial/Virtual COM
type SerialPrinter struct {
	portName string
	baudRate int
	name     string
	port     serial.Port
	mu       sync.Mutex
}

// NewSerialPrinter membuat instance SerialPrinter
func NewSerialPrinter(portName string, baudRate int, name string) *SerialPrinter {
	if name == "" {
		name = "Passbook Printer (Serial/COM)"
	}
	if baudRate <= 0 {
		baudRate = 9600
	}
	return &SerialPrinter{
		portName: portName,
		baudRate: baudRate,
		name:     name,
	}
}

func (s *SerialPrinter) Name() string {
	return s.name
}

// Ping memeriksa apakah port serial tersedia di sistem
func (s *SerialPrinter) Ping() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ports, err := serial.GetPortsList()
	if err != nil {
		return false, fmt.Errorf("gagal mendeteksi daftar port serial: %w", err)
	}

	for _, p := range ports {
		if p == s.portName {
			return true, nil
		}
	}

	return false, fmt.Errorf("port serial '%s' tidak ditemukan di sistem", s.portName)
}

func (s *SerialPrinter) Print(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode := &serial.Mode{
		BaudRate: s.baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(s.portName, mode)
	if err != nil {
		return fmt.Errorf("gagal membuka port serial '%s': %w", s.portName, err)
	}
	defer port.Close()

	// Set read/write timeout 5 detik
	if err := port.SetReadTimeout(5 * time.Second); err != nil {
		slog.Warn("Gagal menyetel read timeout port serial", "error", err)
	}

	n, err := port.Write(data)
	if err != nil {
		return fmt.Errorf("gagal menulis ke port serial: %w", err)
	}

	if n < len(data) {
		return fmt.Errorf("hanya berhasil menulis %d dari %d bytes ke port", n, len(data))
	}

	return nil
}

func (s *SerialPrinter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.port != nil {
		err := s.port.Close()
		s.port = nil
		return err
	}
	return nil
}
