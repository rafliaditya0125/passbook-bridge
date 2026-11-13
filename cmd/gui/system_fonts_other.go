//go:build !windows

package main

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

// getPlatformFontFamilies membaca font yang terinstal pada Linux/macOS via fc-list (fontconfig)
func getPlatformFontFamilies() []string {
	familyMap := make(map[string]bool)

	// Coba jalankan fc-list : family
	cmd := exec.Command("fc-list", ":", "family")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// Beberapa nama dipisah koma pada output fc-list
			for _, fam := range strings.Split(line, ",") {
				fam = strings.TrimSpace(fam)
				if fam != "" {
					familyMap[fam] = true
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
