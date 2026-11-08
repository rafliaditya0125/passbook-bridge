package escp

import (
	"bytes"

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

	// 2. Aktifkan font condensed agar muat dalam buku tabungan
	buf.Write(CmdCondensedOn)

	// 3. Atur panjang halaman fisik buku jika total baris ditentukan (ESC C n)
	if opts.Template != nil && opts.Template.GetTotalRows() > 0 && opts.Template.GetTotalRows() <= 127 {
		buf.Write([]byte{0x1B, 0x43, byte(opts.Template.GetTotalRows())})
	}

	// Roadmap 1: Line skipping jika baris_ke > 1 ditentukan, atau jika template memiliki TopMargin
	lineOffset := 0
	if req.BarisKe > 1 {
		lineOffset = req.BarisKe - 1
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

	// 4. Baris baru (CR LF)
	buf.Write(CmdCRLF)

	// 5. Eject opsional
	if opts.EnableEject {
		buf.Write(CmdFormFeedEject)
	}

	return buf.Bytes()
}
