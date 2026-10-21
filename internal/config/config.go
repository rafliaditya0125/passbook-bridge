package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config merepresentasikan seluruh konfigurasi Passbook Bridge.
type Config struct {
	Port          int    `json:"port"`
	PrinterType   string `json:"printer_type"` // "spooler", "serial", "mock"
	PrinterName   string `json:"printer_name"`
	SerialPort    string `json:"serial_port"`
	BaudRate      int    `json:"baud_rate"`
	LogLevel      string `json:"log_level"`
	EnableEject   bool   `json:"enable_eject"`
	WebhookURL    string `json:"webhook_url,omitempty"`
	TimeoutMs     int    `json:"timeout_ms,omitempty"`
	RetryAttempts int    `json:"retry_attempts,omitempty"`
	CurrencyFormat string `json:"currency_format,omitempty"`
	DateFormat    string `json:"date_format,omitempty"`
	FillChar      string `json:"fill_char,omitempty"`
}

// DefaultConfig mengembalikan konfigurasi default.
func DefaultConfig() *Config {
	return &Config{
		Port:          3001,
		PrinterType:   "mock",
		PrinterName:   "Olivetti PR2 Plus",
		SerialPort:    "COM3",
		BaudRate:      9600,
		LogLevel:      "info",
		EnableEject:   false,
		WebhookURL:    "http://localhost:3000/api/v1/passbook/callback",
		TimeoutMs:     5000,
		RetryAttempts: 3,
		CurrencyFormat: "idr-full",
		DateFormat:    "DD/MM/YYYY",
		FillChar:      "*",
	}
}

// LoadConfig membaca file konfigurasi dari path yang diberikan, atau fallback ke default jika tidak ditemukan.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("gagal membaca file konfigurasi %s: %w", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("format file konfigurasi tidak valid: %w", err)
	}

	if cfg.Port <= 0 {
		cfg.Port = 3001
	}
	if cfg.BaudRate <= 0 {
		cfg.BaudRate = 9600
	}
	if cfg.PrinterType == "" {
		cfg.PrinterType = "mock"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}

// SaveConfig menyimpan konfigurasi ke file JSON.
func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal serialisasi konfigurasi ke JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

