//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// getPlatformFontFamilies membaca font yang terinstal langsung dari registry Windows
// (HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts dan HKCU\Software\Microsoft\Windows NT\CurrentVersion\Fonts)
func getPlatformFontFamilies() []string {
	familyMap := make(map[string]bool)

	// 1. Baca registry HKEY_LOCAL_MACHINE
	readFontsFromWindowsRegistry(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, familyMap)

	// 2. Baca registry HKEY_CURRENT_USER (User fonts di Windows 10/11)
	readFontsFromWindowsRegistry(registry.CURRENT_USER, `Software\Microsoft\Windows NT\CurrentVersion\Fonts`, familyMap)

	// 3. Fallback scan nama file di folder C:\Windows\Fonts jika registry kosong
	if len(familyMap) == 0 {
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		fontsDir := filepath.Join(windir, "Fonts")
		if entries, err := os.ReadDir(fontsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				ext := strings.ToLower(filepath.Ext(name))
				if ext == ".ttf" || ext == ".otf" || ext == ".ttc" {
					baseName := strings.TrimSuffix(name, filepath.Ext(name))
					familyMap[baseName] = true
				}
			}
		}
	}

	var list []string
	for f := range familyMap {
		list = append(list, f)
	}
	return list
}

func readFontsFromWindowsRegistry(rootKey registry.Key, subKey string, targetMap map[string]bool) {
	k, err := registry.OpenKey(rootKey, subKey, registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer k.Close()

	valNames, err := k.ReadValueNames(-1)
	if err != nil {
		return
	}

	for _, valName := range valNames {
		valName = strings.TrimSpace(valName)
		if valName == "" {
			continue
		}
		// Seringkali value name mengandung multiple nama yang digabung '&', misal "MS Gothic & MS PGothic"
		if strings.Contains(valName, "&") {
			parts := strings.Split(valName, "&")
			for _, p := range parts {
				cleaned := CleanFontFamilyName(p)
				if cleaned != "" {
					targetMap[cleaned] = true
				}
			}
		} else {
			cleaned := CleanFontFamilyName(valName)
			if cleaned != "" {
				targetMap[cleaned] = true
			}
		}
	}
}
