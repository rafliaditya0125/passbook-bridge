package main

import (
	"strings"
	"testing"
)

func TestCleanFontFamilyName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Arial (TrueType)", "Arial"},
		{"Arial Bold (TrueType)", "Arial"},
		{"Arial Bold Italic (TrueType)", "Arial"},
		{"Courier New (TrueType)", "Courier New"},
		{"Segoe UI Semibold (TrueType)", "Segoe UI"},
		{"Times New Roman (TrueType)", "Times New Roman"},
		{"Calibri Light (TrueType)", "Calibri"},
		{"MS Gothic & MS PGothic & MS UI Gothic (TrueType)", "MS Gothic"},
		{"Consolas", "Consolas"},
	}

	for _, tt := range tests {
		got := CleanFontFamilyName(tt.input)
		if got != tt.expected {
			t.Errorf("CleanFontFamilyName(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetSystemFontFamilies(t *testing.T) {
	fonts := GetSystemFontFamilies()
	if len(fonts) == 0 {
		t.Fatalf("Expected at least some system fonts, got 0")
	}

	// Cek apakah alfabetis (case-insensitive)
	for i := 1; i < len(fonts); i++ {
		if strings.ToLower(fonts[i]) < strings.ToLower(fonts[i-1]) {
			t.Errorf("Fonts tidak terurut: %s sebelum %s", fonts[i-1], fonts[i])
			break
		}
	}
}
