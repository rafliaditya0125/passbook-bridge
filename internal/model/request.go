package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FlexAmount adalah tipe fleksibel yang dapat menerima angka JSON atau string angka.
type FlexAmount float64

func (fa *FlexAmount) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*fa = 0
		return nil
	}

	// Jika format number langsung
	if data[0] != '"' {
		var f float64
		if err := json.Unmarshal(data, &f); err != nil {
			return fmt.Errorf("nominal tidak valid: %s", string(data))
		}
		*fa = FlexAmount(f)
		return nil
	}

	// Jika format string, contoh: "100000", "100,000", "100.000,50"
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		*fa = 0
		return nil
	}

	// Bersihkan pemisah ribuan/format umum jika ada
	// Jika ada titik dan koma, misal "100.000,00", ubah ke standard float "100000.00"
	cleanStr := strings.ReplaceAll(s, " ", "")
	if strings.Contains(cleanStr, ".") && strings.Contains(cleanStr, ",") {
		cleanStr = strings.ReplaceAll(cleanStr, ".", "")
		cleanStr = strings.ReplaceAll(cleanStr, ",", ".")
	} else if strings.Contains(cleanStr, ",") && !strings.Contains(cleanStr, ".") {
		// misal "100000,50"
		cleanStr = strings.ReplaceAll(cleanStr, ",", ".")
	}

	val, err := strconv.ParseFloat(cleanStr, 64)
	if err != nil {
		return fmt.Errorf("nominal string tidak valid '%s': %w", s, err)
	}

	*fa = FlexAmount(val)
	return nil
}

// Float64 mengembalikan nilai float64 murni.
func (fa FlexAmount) Float64() float64 {
	return float64(fa)
}

// PrintRequest merepresentasikan payload body dari POST /print
type PrintRequest struct {
	Tanggal        string     `json:"tanggal"`
	KodeTransaksi  string     `json:"kode_transaksi"`
	JenisTransaksi string     `json:"jenis_transaksi"`
	Jumlah         FlexAmount `json:"jumlah"`
	SaldoSesudah   FlexAmount `json:"saldo_sesudah"`
	Keterangan     string     `json:"keterangan"`
	Petugas        string     `json:"petugas"`
	BarisKe        int        `json:"baris_ke,omitempty"` // Opsional sesuai roadmap line skipping
}

// Validate memeriksa apakah field wajib telah terpenuhi dan valid.
func (r *PrintRequest) Validate() error {
	if strings.TrimSpace(r.Tanggal) == "" {
		return fmt.Errorf("field 'tanggal' wajib diisi")
	}
	if strings.TrimSpace(r.KodeTransaksi) == "" {
		return fmt.Errorf("field 'kode_transaksi' wajib diisi")
	}
	if strings.TrimSpace(r.JenisTransaksi) == "" {
		return fmt.Errorf("field 'jenis_transaksi' wajib diisi")
	}

	jt := strings.ToLower(strings.TrimSpace(r.JenisTransaksi))
	validJenis := map[string]bool{
		"setor":       true,
		"tarik":       true,
		"bunga":       true,
		"biaya_admin": true,
	}
	if !validJenis[jt] {
		return fmt.Errorf("field 'jenis_transaksi' harus salah satu dari: setor, tarik, bunga, biaya_admin")
	}

	if r.Jumlah <= 0 {
		return fmt.Errorf("field 'jumlah' wajib diisi dan harus bernilai lebih besar dari 0")
	}
	if r.SaldoSesudah < 0 {
		return fmt.Errorf("field 'saldo_sesudah' tidak boleh negatif")
	}

	return nil
}
