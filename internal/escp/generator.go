package escp

import (
	"bytes"
	"strings"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/model"
)

// ESC/P Command Constants
var (
	CmdInitialize    = []byte{0x1B, 0x40}       // ESC @ : Reset printer ke status awal
	CmdCondensedOn   = []byte{0x0F}             // SI    : Mode font condensed (huruf rapat)
	CmdCRLF          = []byte{0x0D, 0x0A}       // CR LF : Carriage Return & Line Feed
	CmdFormFeedEject = []byte{0x0C}             // FF    : Form Feed / Eject
	CmdPassbookEject = []byte{0x1B, 0x19}       // ESC EM: Passbook Eject command khusus
)

// GeneratorOptions opsi konfigurasi saat men-generate byte ESC/P
type GeneratorOptions struct {
	EnableEject bool
	Template    *config.LayoutTemplate
}

// GeneratePrintCommand merangkai byte ESC/P siap kirim ke printer dot-matrix
func GeneratePrintCommand(req *model.PrintRequest, opts GeneratorOptions) []byte {
	var buf bytes.Buffer

	// 1. Inisialisasi printer
	buf.Write(CmdInitialize)

	// 2. Aktifkan font pitch & style sesuai template
	if opts.Template != nil {
		pitch := opts.Template.GetFontPitchCPI()
		switch pitch {
		case 10:
			buf.Write([]byte{0x1B, 0x50}) // ESC P (10 CPI Pica - Lebar)
		case 12:
			buf.Write([]byte{0x1B, 0x4D}) // ESC M (12 CPI Elite - Sedang)
		case 15:
			buf.Write([]byte{0x1B, 0x67}) // ESC g (15 CPI - Rapat)
		default:
			buf.Write(CmdCondensedOn) // SI (Condensed ~17 CPI standar buku tabungan)
		}

		// Typeface ESC/P: 0=Roman, 1=Sans-serif, 2=Courier
		fFamily := strings.ToLower(opts.Template.GetFontFamily())
		if strings.Contains(fFamily, "courier") || strings.Contains(fFamily, "consolas") {
			buf.Write([]byte{0x1B, 0x6B, 0x02}) // ESC k 2 (Courier)
		} else if strings.Contains(fFamily, "sans") || strings.Contains(fFamily, "arial") || strings.Contains(fFamily, "tahoma") || strings.Contains(fFamily, "segoe") {
			buf.Write([]byte{0x1B, 0x6B, 0x01}) // ESC k 1 (Sans Serif)
		} else if strings.Contains(fFamily, "times") || strings.Contains(fFamily, "roman") {
			buf.Write([]byte{0x1B, 0x6B, 0x00}) // ESC k 0 (Roman)
		}

		// Font Style: Bold / Italic
		fStyle := strings.ToLower(opts.Template.GetFontStyle())
		if strings.Contains(fStyle, "bold") {
			buf.Write([]byte{0x1B, 0x45}) // ESC E (Bold on)
		}
		if strings.Contains(fStyle, "italic") {
			buf.Write([]byte{0x1B, 0x34}) // ESC 4 (Italic on)
		}
	} else {
		buf.Write(CmdCondensedOn)
	}

	// 3. Atur panjang halaman fisik buku jika total baris ditentukan (ESC C n)
	if opts.Template != nil && opts.Template.GetTotalRows() > 0 && opts.Template.GetTotalRows() <= 127 {
		buf.Write([]byte{0x1B, 0x43, byte(opts.Template.GetTotalRows())})
	}

	// Roadmap 1: Line skipping jika target baris > 1 ditentukan, atau jika template memiliki TopMargin
	lineOffset := 0
	targetBaris := req.GetBaris()
	if targetBaris > 1 {
		lineOffset = targetBaris - 1
	} else if opts.Template != nil && opts.Template.TopMargin > 0 {
		lineOffset = opts.Template.TopMargin
	}
	for i := 0; i < lineOffset; i++ {
		buf.Write(CmdCRLF)
	}

	// 3. Format satu baris teks transaksi sesuai template jika tersedia
	var lineText string
	if opts.Template != nil {
		lineText = FormatLineWithTemplate(req, opts.Template)
	} else {
		lineText = FormatLine(req)
	}
	buf.WriteString(lineText)

	// 4. Baris baru (CR LF) dan reset style
	buf.Write(CmdCRLF)
	if opts.Template != nil {
		fStyle := strings.ToLower(opts.Template.GetFontStyle())
		if strings.Contains(fStyle, "bold") {
			buf.Write([]byte{0x1B, 0x46}) // ESC F (Bold off)
		}
		if strings.Contains(fStyle, "italic") {
			buf.Write([]byte{0x1B, 0x35}) // ESC 5 (Italic off)
		}
	}

	// 5. Eject opsional
	if opts.EnableEject {
		buf.Write(CmdFormFeedEject)
	}

	return buf.Bytes()
}
