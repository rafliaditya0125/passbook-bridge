package main

import (
	"sort"
	"strings"
	"sync"
)

var (
	cachedSystemFonts []string
	systemFontsMutex  sync.Mutex
)

// defaultFallbackFonts digunakan sebagai cadangan jika query sistem operasi tidak menemukan font sama sekali.
var defaultFallbackFonts = []string{
	"Arial",
	"Arial Narrow",
	"Calibri",
	"Cambria",
	"Cascadia Code",
	"Comic Sans MS",
	"Consolas",
	"Courier New",
	"Franklin Gothic Medium",
	"Georgia",
	"Impact",
	"Lucida Console",
	"Lucida Sans",
	"Microsoft Sans Serif",
	"Palatino Linotype",
	"Segoe UI",
	"Tahoma",
	"Times New Roman",
	"Trebuchet MS",
	"Verdana",
}

// GetSystemFontFamilies mengembalikan daftar unik nama font family yang terpasang pada sistem operasi (Windows/Linux/macOS).
func GetSystemFontFamilies() []string {
	systemFontsMutex.Lock()
	defer systemFontsMutex.Unlock()

	if len(cachedSystemFonts) > 0 {
		return cachedSystemFonts
	}

	rawList := getPlatformFontFamilies()
	familyMap := make(map[string]bool)

	for _, f := range rawList {
		cleaned := CleanFontFamilyName(f)
		if cleaned != "" && len(cleaned) > 1 {
			familyMap[cleaned] = true
		}
	}

	// Pastikan font standar selalu ada di daftar jika sistem mengembalikan terlalu sedikit
	if len(familyMap) < 5 {
		for _, def := range defaultFallbackFonts {
			familyMap[def] = true
		}
	}

	var result []string
	for name := range familyMap {
		result = append(result, name)
	}

	// Urutkan secara alfabetikal (case-insensitive)
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})

	cachedSystemFonts = result
	return cachedSystemFonts
}

// CleanFontFamilyName membersihkan format teknis Windows/OpenType (misal " (TrueType)") dan style suffix.
func CleanFontFamilyName(raw string) string {
	// 1. Buang keterangan kurung seperti "(TrueType)", "(OpenType)", dll.
	if idx := strings.Index(raw, "("); idx != -1 {
		raw = raw[:idx]
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// 2. Jika ada simbol '&', ambil nama pertama
	if strings.Contains(raw, "&") {
		parts := strings.Split(raw, "&")
		raw = strings.TrimSpace(parts[0])
	}

	// 3. Bersihkan akhiran style standar Windows
	suffixes := []string{
		" Bold Italic", " Italic Bold",
		" Bold", " Italic", " Oblique",
		" Regular", " Light Italic", " Light",
		" SemiBold Italic", " SemiBold",
		" Semibold Italic", " Semibold",
		" Black Italic", " Black",
		" ExtraBold Italic", " ExtraBold",
		" ExtraLight Italic", " ExtraLight",
		" Medium Italic", " Medium",
		" Thin Italic", " Thin",
		" UltraBold", " UltraLight",
		" Heavy Italic", " Heavy",
	}

	for _, sfx := range suffixes {
		if strings.HasSuffix(strings.ToLower(raw), strings.ToLower(sfx)) {
			raw = strings.TrimSpace(raw[:len(raw)-len(sfx)])
			break
		}
	}

	return strings.TrimSpace(raw)
}
