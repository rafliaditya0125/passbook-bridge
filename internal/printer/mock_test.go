package printer

import (
	"testing"

	"passbook-bridge/internal/config"
)

func TestMockPrinter(t *testing.T) {
	p := NewMockPrinter("TEST-PLQ20")

	if p.Name() != "TEST-PLQ20" {
		t.Errorf("Nama printer salah, got: %s", p.Name())
	}

	connected, err := p.Ping()
	if err != nil || !connected {
		t.Fatalf("Ping gagal pada mock printer: %v", err)
	}

	data := []byte("20/10/26 SET                100.000,00     250.000,00SRI Y       \r\n")
	if err := p.Print(data); err != nil {
		t.Fatalf("Print gagal pada mock printer: %v", err)
	}

	p.SetConnected(false)
	connected, _ = p.Ping()
	if connected {
		t.Errorf("Harusnya connected bernilai false")
	}

	if err := p.Print(data); err == nil {
		t.Errorf("Harusnya print gagal saat offline")
	}
}

func TestNewPrinter_Factory(t *testing.T) {
	cfg := &config.Config{
		PrinterType: "mock",
		PrinterName: "CustomMock",
	}

	p, err := NewPrinter(cfg)
	if err != nil {
		t.Fatalf("Gagal membuat mock printer: %v", err)
	}
	if p.Name() != "CustomMock" {
		t.Errorf("Nama printer tidak sesuai: %s", p.Name())
	}
}
