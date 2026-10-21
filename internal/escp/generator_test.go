package escp

import (
	"bytes"
	"testing"

	"passbook-bridge/internal/model"
)

func TestFormatTanggal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026-10-20 13:45:00", "20/10/26 "},
		{"2026-10-20", "20/10/26 "},
		{"21/10/2026", "21/10/26 "},
	}

	for _, tt := range tests {
		got := FormatTanggal(tt.input)
		if got != tt.expected {
			t.Errorf("FormatTanggal(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
		if len(got) != ColWidthTanggal {
			t.Errorf("Panjang tanggal harus %d, dapat %d", ColWidthTanggal, len(got))
		}
	}
}

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{100000, "100.000,00"},
		{50000, "50.000,00"},
		{250000.5, "250.000,50"},
		{0, "0,00"},
	}

	for _, tt := range tests {
		got := FormatRupiah(tt.input)
		if got != tt.expected {
			t.Errorf("FormatRupiah(%v) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatLine_Setor(t *testing.T) {
	req := &model.PrintRequest{
		Tanggal:        "2026-10-20 13:45:00",
		KodeTransaksi:  "BKM001/10/26",
		JenisTransaksi: "setor",
		Jumlah:         100000,
		SaldoSesudah:   250000,
		Petugas:        "SRI Y",
	}

	line := FormatLine(req)
	if len(line) != TotalLineWidth {
		t.Fatalf("Panjang baris harus %d karakter, dapat %d", TotalLineWidth, len(line))
	}

	// Tanggal (0..9)
	colTgl := line[0:9]
	if colTgl != "20/10/26 " {
		t.Errorf("Kolom tanggal salah: %q", colTgl)
	}

	// Sandi (9..14)
	colSandi := line[9:14]
	if colSandi != "SET  " {
		t.Errorf("Kolom sandi salah: %q", colSandi)
	}

	// Debet (14..27) harus kosong
	colDebet := line[14:27]
	if colDebet != "             " {
		t.Errorf("Kolom debet untuk setor harus kosong, dapat: %q", colDebet)
	}

	// Kredit (27..40)
	colKredit := line[27:40]
	if colKredit != "   100.000,00" {
		t.Errorf("Kolom kredit salah, dapat: %q", colKredit)
	}

	// Saldo (40..55)
	colSaldo := line[40:55]
	if colSaldo != "     250.000,00" {
		t.Errorf("Kolom saldo salah, dapat: %q", colSaldo)
	}

	// Validasi (55..67)
	colValidasi := line[55:67]
	if colValidasi != "SRI Y       " {
		t.Errorf("Kolom validasi salah, dapat: %q", colValidasi)
	}
}

func TestFormatLine_Tarik(t *testing.T) {
	req := &model.PrintRequest{
		Tanggal:        "2026-10-21 10:15:00",
		KodeTransaksi:  "BKK002/10/26",
		JenisTransaksi: "tarik",
		Jumlah:         50000,
		SaldoSesudah:   200000,
		Petugas:        "SRI Y",
	}

	line := FormatLine(req)
	if len(line) != TotalLineWidth {
		t.Fatalf("Panjang baris harus %d karakter, dapat %d", TotalLineWidth, len(line))
	}

	// Sandi (9..14)
	colSandi := line[9:14]
	if colSandi != "TAR  " {
		t.Errorf("Kolom sandi salah: %q", colSandi)
	}

	// Debet (14..27) harus terisi 50.000,00
	colDebet := line[14:27]
	if colDebet != "    50.000,00" {
		t.Errorf("Kolom debet salah, dapat: %q", colDebet)
	}

	// Kredit (27..40) harus kosong
	colKredit := line[27:40]
	if colKredit != "             " {
		t.Errorf("Kolom kredit untuk tarik harus kosong, dapat: %q", colKredit)
	}
}

func TestGeneratePrintCommand(t *testing.T) {
	req := &model.PrintRequest{
		Tanggal:        "2026-10-20 13:45:00",
		KodeTransaksi:  "BKM001/10/26",
		JenisTransaksi: "setor",
		Jumlah:         100000,
		SaldoSesudah:   250000,
		Petugas:        "SRI Y",
	}

	raw := GeneratePrintCommand(req, GeneratorOptions{EnableEject: true})

	// Periksa header ESC @ (0x1B, 0x40)
	if !bytes.HasPrefix(raw, []byte{0x1B, 0x40}) {
		t.Error("Output harus diawali perintah ESC @ (0x1B 0x40)")
	}

	// Periksa mode condensed (0x0F)
	if raw[2] != 0x0F {
		t.Error("Karakter ke-3 harus 0x0F (SI)")
	}

	// Periksa CRLF dan form feed di akhir
	if !bytes.HasSuffix(raw, []byte{0x0D, 0x0A, 0x0C}) {
		t.Error("Output harus diakhiri dengan CR LF FF saat EnableEject = true")
	}
}
