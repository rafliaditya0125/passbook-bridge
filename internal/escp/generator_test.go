package escp

import (
	"bytes"
	"testing"

	"passbook-bridge/internal/config"
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

func TestGeneratePrintCommand_FontOptions(t *testing.T) {
	req := &model.PrintRequest{
		BarisKe:        5,
		Tanggal:        "2026-11-08",
		KodeTransaksi:  "K01",
		JenisTransaksi: "setor",
		Jumlah:         500000,
		SaldoSesudah:   500000,
		Petugas:        "teller",
	}

	tmpl := &config.LayoutTemplate{
		ID:           "test_font_escp",
		FontFamily:   "Courier New",
		FontStyle:    "Bold",
		FontPitchCPI: 12, // 12 CPI -> ESC M
		TotalRows:    34, // ESC C 34
		Columns: []config.ColumnDef{
			{ID: "c1", FieldKey: "tanggal", ColStart: 0, ColWidth: 10},
		},
	}

	raw := GeneratePrintCommand(req, GeneratorOptions{Template: tmpl})

	// Cek ESC M (0x1B, 0x4D) untuk 12 CPI
	if !bytes.Contains(raw, []byte{0x1B, 0x4D}) {
		t.Errorf("Expected 12 CPI command ESC M (1B 4D)")
	}

	// Cek ESC k 2 (0x1B, 0x6B, 0x02) untuk Courier typeface
	if !bytes.Contains(raw, []byte{0x1B, 0x6B, 0x02}) {
		t.Errorf("Expected Courier typeface command ESC k 2 (1B 6B 02)")
	}

	// Cek ESC E (0x1B, 0x45) untuk Bold on
	if !bytes.Contains(raw, []byte{0x1B, 0x45}) {
		t.Errorf("Expected Bold on command ESC E (1B 45)")
	}

	// Cek ESC F (0x1B, 0x46) untuk Bold off di akhir
	if !bytes.Contains(raw, []byte{0x1B, 0x46}) {
		t.Errorf("Expected Bold off command ESC F (1B 46)")
	}

	// Cek ESC C 34 (0x1B, 0x43, 0x22) untuk panjang halaman 34 baris
	if !bytes.Contains(raw, []byte{0x1B, 0x43, 34}) {
		t.Errorf("Expected Page Length command ESC C 34 (1B 43 22)")
	}
}

func TestFormatLineWithTemplate_Max100ColumnsAndValidasiAt80(t *testing.T) {
	req := &model.PrintRequest{
		BarisKe:        5,
		Tanggal:        "2026-11-12",
		KodeTransaksi:  "BKM-0001",
		JenisTransaksi: "setor",
		Jumlah:         50000,
		SaldoSesudah:   250000,
		Petugas:        "Teller 1",
	}

	tmpl := &config.LayoutTemplate{
		ID:         "test-100-cols",
		Name:       "Test 100 Kolom",
		MaxColumns: 100,
		Columns: []config.ColumnDef{
			{ID: "c_no", FieldKey: "no", ColStart: 0, ColWidth: 4, Align: "left"},
			{ID: "c_tgl", FieldKey: "tanggal", ColStart: 4, ColWidth: 11, Align: "left"},
			{ID: "c_sandi", FieldKey: "kode_transaksi", ColStart: 16, ColWidth: 6, Align: "center"},
			{ID: "c_debet", FieldKey: "debet", ColStart: 23, ColWidth: 12, Align: "right"},
			{ID: "c_kredit", FieldKey: "kredit", ColStart: 36, ColWidth: 12, Align: "right"},
			{ID: "c_saldo", FieldKey: "saldo_sesudah", ColStart: 49, ColWidth: 14, Align: "right"},
			{ID: "c_validasi", FieldKey: "petugas", ColStart: 80, ColWidth: 14, Align: "left"},
		},
	}

	line := FormatLineWithTemplate(req, tmpl)

	// Validasi harus ada di kolom 80 dan tidak nempel dengan saldo di 49..63
	if len(line) < 88 {
		t.Fatalf("Panjang baris hasil cetak minimal 88 karakter (Validasi di col 80+), dapat: %d |%s|", len(line), line)
	}

	// Cek bahwa di antara col 63 (akhir saldo) dan col 80 (awal validasi) adalah spasi (tidak nempel)
	betweenSaldoAndValidasi := line[63:80]
	for i, ch := range betweenSaldoAndValidasi {
		if ch != ' ' {
			t.Errorf("Karakter antara saldo dan validasi harus spasi (indeks %d: %q)", 63+i, ch)
		}
	}

	// Cek teks validasi pada posisi 80
	validasiText := line[80:88]
	if validasiText != "Teller 1" {
		t.Errorf("Kolom validasi pada posisi 80 salah: got %q, expected 'Teller 1'", validasiText)
	}
}

func TestFormatLineWithTemplate_BankBJB_NoStickyOverlap(t *testing.T) {
	req := &model.PrintRequest{
		BarisKe:        14,
		Tanggal:        "12-11-2026",
		KodeTransaksi:  "K14",
		JenisTransaksi: "setor",
		Jumlah:         7000,
		SaldoSesudah:   7000,
		Petugas:        "adm",
		Keterangan:     "Setoran Target Baris 14",
	}

	tmpl := &config.LayoutTemplate{
		ID:         "tmpl-1792500970",
		Name:       "Bank BJB",
		MaxColumns: 100,
		Columns: []config.ColumnDef{
			{ID: "c_tgl", FieldKey: "tanggal", ColStart: 0, ColWidth: 12, Align: "left"},
			{ID: "c_sandi", FieldKey: "kode_transaksi", ColStart: 13, ColWidth: 5, Align: "left"},
			{ID: "c_debet", FieldKey: "debet", ColStart: 20, ColWidth: 15, Align: "right"},
			{ID: "c_kredit", FieldKey: "kredit", ColStart: 40, ColWidth: 15, Align: "right"},
			{ID: "c_saldo", FieldKey: "saldo_sesudah", ColStart: 60, ColWidth: 15, Align: "right"},
			{ID: "c_validasi", FieldKey: "petugas", ColStart: 80, ColWidth: 15, Align: "left"},
		},
	}

	line := FormatLineWithTemplate(req, tmpl)

	// Cek panjang baris
	if len(line) < 83 {
		t.Fatalf("Panjang baris harus minimal 83 karakter (adm di col 80..82), got: %d |%s|", len(line), line)
	}

	// Saldo (60..75) -> "       7.000,00" -> berakhir di indeks 74
	// Di antara indeks 75 dan 80 HARUS spasi (minimal 5 spasi pemisah)
	betweenSaldoAndValidasi := line[75:80]
	if betweenSaldoAndValidasi != "     " {
		t.Errorf("Jeda antara saldo dan validasi harus 5 spasi, got: %q", betweenSaldoAndValidasi)
	}

	// Validasi pada indeks 80 harus "adm"
	validasiText := line[80:83]
	if validasiText != "adm" {
		t.Errorf("Kolom validasi pada indeks 80 harus 'adm', got: %q", validasiText)
	}
}

func TestGeneratePrintCommand_TopMargin_Row1AndSubsequent(t *testing.T) {
	tmpl := &config.LayoutTemplate{
		ID:         "test_top_margin",
		Name:       "Test Top Margin",
		TopMargin:  3,
		MaxColumns: 80,
		Columns: []config.ColumnDef{
			{ID: "c1", FieldKey: "keterangan", ColStart: 0, ColWidth: 20},
		},
	}

	tests := []struct {
		name               string
		topMargin          int
		baris              int
		expectedLineOffset int
	}{
		{
			name:               "TopMargin 3, Baris 1 (geser baris pertama)",
			topMargin:          3,
			baris:              1,
			expectedLineOffset: 3, // 3 + (1-1) = 3
		},
		{
			name:               "TopMargin 3, Baris 2 (geser baris kedua)",
			topMargin:          3,
			baris:              2,
			expectedLineOffset: 4, // 3 + (2-1) = 4
		},
		{
			name:               "TopMargin 3, Baris 14 (geser baris ke-14)",
			topMargin:          3,
			baris:              14,
			expectedLineOffset: 16, // 3 + (14-1) = 16
		},
		{
			name:               "TopMargin 0, Baris 1 (tanpa margin, baris 1)",
			topMargin:          0,
			baris:              1,
			expectedLineOffset: 0, // 0 + 0 = 0
		},
		{
			name:               "TopMargin 0, Baris 5 (tanpa margin, baris 5)",
			topMargin:          0,
			baris:              5,
			expectedLineOffset: 4, // 0 + (5-1) = 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentTmpl := *tmpl
			currentTmpl.TopMargin = tt.topMargin

			req := &model.PrintRequest{
				BarisKe:    model.FlexInt(tt.baris),
				Keterangan: "TEST_LINE_OFFSET",
			}

			raw := GeneratePrintCommand(req, GeneratorOptions{
				Template: &currentTmpl,
			})

			targetText := "TEST_LINE_OFFSET"
			idx := bytes.Index(raw, []byte(targetText))
			if idx == -1 {
				t.Fatalf("Target text %q tidak ditemukan dalam raw output", targetText)
			}

			crlfBeforeLine := bytes.Count(raw[:idx], []byte{0x0D, 0x0A})
			if crlfBeforeLine != tt.expectedLineOffset {
				t.Errorf("CRLF sebelum teks = %d, expected %d", crlfBeforeLine, tt.expectedLineOffset)
			}
		})
	}
}
