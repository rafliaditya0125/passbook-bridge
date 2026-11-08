package escp

import (
	"fmt"
	"math"
	"strings"
	"time"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/model"
)

// Lebar kolom sesuai spesifikasi SRS 4.2
const (
	ColWidthTanggal  = 9
	ColWidthSandi    = 5
	ColWidthDebet    = 13
	ColWidthKredit   = 13
	ColWidthSaldo    = 15
	ColWidthValidasi = 12
	TotalLineWidth   = ColWidthTanggal + ColWidthSandi + ColWidthDebet + ColWidthKredit + ColWidthSaldo + ColWidthValidasi // 67 karakter
)

// Map jenis transaksi ke kode sandi buku tabungan
var jenisToSandi = map[string]string{
	"setor":       "SET",
	"tarik":       "TAR",
	"bunga":       "BNG",
	"biaya_admin": "ADM",
}

// FormatTanggal mengubah berbagai variasi tanggal input menjadi format "DD/MM/YY"
func FormatTanggal(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return strings.Repeat(" ", ColWidthTanggal)
	}

	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"02/01/2006",
		"02/01/06",
		"02-01-2006",
		"02-01-06",
	}

	var parsed time.Time
	var err error
	for _, layout := range formats {
		parsed, err = time.Parse(layout, raw)
		if err == nil {
			break
		}
	}

	var formatted string
	if err == nil {
		formatted = parsed.Format("02/01/06")
	} else {
		// Fallback: jika tidak bisa di-parse, ambil hingga 8 char
		if len(raw) > 8 {
			formatted = raw[:8]
		} else {
			formatted = raw
		}
	}

	return LeftAlign(formatted, ColWidthTanggal)
}

// FormatRupiah mengonversi nilai angka menjadi format "100.000,00"
func FormatRupiah(amount float64) string {
	isNeg := amount < 0
	amount = math.Abs(amount)

	intPart := int64(amount)
	decPart := int64(math.Round((amount - float64(intPart)) * 100))
	if decPart >= 100 {
		intPart++
		decPart = 0
	}

	// Format ribuan dengan titik
	intStr := fmt.Sprintf("%d", intPart)
	var formattedInt strings.Builder
	l := len(intStr)
	for i, ch := range intStr {
		if i > 0 && (l-i)%3 == 0 {
			formattedInt.WriteByte('.')
		}
		formattedInt.WriteRune(ch)
	}

	res := fmt.Sprintf("%s,%02d", formattedInt.String(), decPart)
	if isNeg {
		res = "-" + res
	}
	return res
}

// LeftAlign memformat teks rata kiri dengan padding spasi di kanan
func LeftAlign(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// RightAlign memformat teks rata kanan dengan padding spasi di kiri
func RightAlign(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// CenterAlign memformat teks rata tengah dengan padding spasi di kiri dan kanan
func CenterAlign(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	pad := width - len(s)
	leftPad := pad / 2
	rightPad := pad - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// FormatLine menyusun baris transaksi tunggal dengan lebar 67 karakter sesuai SRS 4.1 & 4.2
func FormatLine(req *model.PrintRequest) string {
	colTanggal := FormatTanggal(req.Tanggal)

	jt := strings.ToLower(strings.TrimSpace(req.JenisTransaksi))
	sandi, ok := jenisToSandi[jt]
	if !ok {
		sandi = LeftAlign(strings.ToUpper(req.JenisTransaksi), 3)
	}
	colSandi := LeftAlign(sandi, ColWidthSandi)

	var debetStr, kreditStr string
	formattedAmount := FormatRupiah(req.Jumlah.Float64())

	switch jt {
	case "tarik", "biaya_admin":
		debetStr = RightAlign(formattedAmount, ColWidthDebet)
		kreditStr = strings.Repeat(" ", ColWidthKredit)
	case "setor", "bunga":
		fallthrough
	default:
		debetStr = strings.Repeat(" ", ColWidthDebet)
		kreditStr = RightAlign(formattedAmount, ColWidthKredit)
	}

	colSaldo := RightAlign(FormatRupiah(req.SaldoSesudah.Float64()), ColWidthSaldo)

	petugas := strings.TrimSpace(req.Petugas)
	colValidasi := LeftAlign(petugas, ColWidthValidasi)

	// Gabungkan seluruh kolom
	line := colTanggal + colSandi + debetStr + kreditStr + colSaldo + colValidasi
	return line
}

// FormatLineWithTemplate menyusun baris transaksi sesuai template dinamis yang dirancang di GUI
func FormatLineWithTemplate(req *model.PrintRequest, tmpl *config.LayoutTemplate) string {
	if tmpl == nil || len(tmpl.Columns) == 0 {
		return FormatLine(req)
	}

	maxCols := tmpl.MaxColumns
	if maxCols <= 0 {
		maxCols = 80
	}

	// Buffer karakter terisi spasi sepanjang maxCols
	buf := make([]rune, maxCols)
	for i := range buf {
		buf[i] = ' '
	}

	jt := strings.ToLower(strings.TrimSpace(req.JenisTransaksi))
	formattedAmount := FormatRupiah(req.Jumlah.Float64())

	for _, col := range tmpl.Columns {
		if col.ColWidth <= 0 {
			continue
		}

		var rawVal string
		switch strings.ToLower(col.FieldKey) {
		case "no", "nomor", "baris":
			if req.BarisKe > 0 {
				rawVal = fmt.Sprintf("%02d", req.BarisKe)
			} else {
				rawVal = "01"
			}
		case "tanggal", "tgl":
			rawVal = strings.TrimSpace(FormatTanggal(req.Tanggal))
		case "kode_transaksi", "sandi", "kode/sandi":
			sandi, ok := jenisToSandi[jt]
			if !ok {
				if req.KodeTransaksi != "" {
					sandi = req.KodeTransaksi
				} else {
					sandi = strings.ToUpper(req.JenisTransaksi)
				}
			}
			rawVal = sandi
		case "debet":
			if jt == "tarik" || jt == "biaya_admin" {
				rawVal = formattedAmount
			}
		case "kredit":
			if jt == "setor" || jt == "bunga" {
				rawVal = formattedAmount
			}
		case "saldo", "saldo_sesudah":
			rawVal = FormatRupiah(req.SaldoSesudah.Float64())
		case "petugas", "validasi", "teller", "paraf", "ket/paraf":
			rawVal = strings.TrimSpace(req.Petugas)
		case "keterangan", "uraian":
			rawVal = strings.TrimSpace(req.Keterangan)
		case "jenis_transaksi":
			rawVal = strings.TrimSpace(req.JenisTransaksi)
		default:
			rawVal = ""
		}

		var formatted string
		switch col.Align {
		case "right":
			formatted = RightAlign(rawVal, col.ColWidth)
		case "center":
			formatted = CenterAlign(rawVal, col.ColWidth)
		default:
			formatted = LeftAlign(rawVal, col.ColWidth)
		}

		// Sisipkan ke buffer pada posisi ColStart (+ LeftMargin jika ada)
		start := col.ColStart + tmpl.LeftMargin
		if start < 0 {
			start = 0
		}
		runes := []rune(formatted)
		for i := 0; i < len(runes) && (start+i) < len(buf); i++ {
			buf[start+i] = runes[i]
		}
	}

	// Trim spasi berlebih di paling kanan untuk efisiensi cetak dot matrix
	res := strings.TrimRight(string(buf), " ")
	return res
}

