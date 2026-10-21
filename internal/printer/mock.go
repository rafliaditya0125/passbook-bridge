package printer

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// MockPrinter mengimplementasikan antarmuka Printer untuk testing dan simulasi lokal
type MockPrinter struct {
	name      string
	connected bool
	mu        sync.Mutex
	logFile   string
}

// NewMockPrinter membuat printer tiruan untuk development/simulasi
func NewMockPrinter(name string) *MockPrinter {
	if name == "" {
		name = "EPSON PLQ-20 (Simulasi Mock)"
	}
	return &MockPrinter{
		name:      name,
		connected: true,
		logFile:   "simulated_passbook.log",
	}
}

func (m *MockPrinter) Name() string {
	return m.name
}

func (m *MockPrinter) Ping() (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected, nil
}

func (m *MockPrinter) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

func (m *MockPrinter) Print(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return fmt.Errorf("printer mock offline / tidak terhubung")
	}

	// Filter out non-printable control characters for display
	var printable bytes.Buffer
	for _, b := range data {
		if b >= 32 && b <= 126 {
			printable.WriteByte(b)
		} else if b == '\n' {
			printable.WriteByte('\n')
		}
	}

	cleanText := printable.String()
	slog.Info("🖨️ [MOCK PRINTER] Menerima data cetak",
		"bytes_total", len(data),
		"simulated_output", cleanText,
	)

	// Simpan ke log file simulasi agar dapat diinspeksi
	entry := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), cleanText)
	f, err := os.OpenFile(m.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		_, _ = f.WriteString(entry)
	}

	return nil
}

func (m *MockPrinter) Close() error {
	return nil
}
