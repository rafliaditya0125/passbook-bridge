package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/config"
)

// Sample transactions for realistic bank passbook preview (16 baris sesuai format mutasi standar buku tabungan)
var samplePassbookRows = []struct {
	Baris      int
	Tanggal    string
	Kode       string
	Keterangan string
	Debet      int64
	Kredit     int64
	Saldo      int64
	Paraf      string
}{
	{Baris: 1, Tanggal: "05-10-2021", Kode: "K01", Keterangan: "Saldo Awal", Debet: 0, Kredit: 5000000, Saldo: 5000000, Paraf: "adm"},
	{Baris: 2, Tanggal: "05-10-2021", Kode: "K02", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 100000, Saldo: 5100000, Paraf: "adm"},
	{Baris: 3, Tanggal: "05-10-2021", Kode: "K03", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 10000, Saldo: 5110000, Paraf: "adm"},
	{Baris: 4, Tanggal: "05-10-2021", Kode: "K04", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 30000, Saldo: 5140000, Paraf: "adm"},
	{Baris: 5, Tanggal: "05-10-2021", Kode: "K05", Keterangan: "Penarikan Tunai", Debet: 500000, Kredit: 0, Saldo: 4640000, Paraf: "adm"},
	{Baris: 6, Tanggal: "05-10-2021", Kode: "K06", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 10000, Saldo: 4650000, Paraf: "adm"},
	{Baris: 7, Tanggal: "05-10-2021", Kode: "K07", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 10000, Saldo: 4660000, Paraf: "adm"},
	{Baris: 8, Tanggal: "05-10-2021", Kode: "K08", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 10000, Saldo: 4670000, Paraf: "adm"},
	{Baris: 9, Tanggal: "05-10-2021", Kode: "K09", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 100000, Saldo: 4770000, Paraf: "adm"},
	{Baris: 10, Tanggal: "06-10-2021", Kode: "K10", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 50000, Saldo: 4820000, Paraf: "adm"},
	{Baris: 11, Tanggal: "07-10-2021", Kode: "K11", Keterangan: "Bunga Tabungan", Debet: 0, Kredit: 15000, Saldo: 4835000, Paraf: "sys"},
	{Baris: 12, Tanggal: "07-10-2021", Kode: "K12", Keterangan: "Pajak Bunga", Debet: 3000, Kredit: 0, Saldo: 4832000, Paraf: "sys"},
	{Baris: 13, Tanggal: "08-10-2021", Kode: "K13", Keterangan: "Biaya Adm", Debet: 5000, Kredit: 0, Saldo: 4827000, Paraf: "sys"},
	{Baris: 14, Tanggal: "08-10-2021", Kode: "K14", Keterangan: "Setoran Target", Debet: 0, Kredit: 70000, Saldo: 4897000, Paraf: "adm"},
	{Baris: 15, Tanggal: "09-10-2021", Kode: "K15", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 50000, Saldo: 4947000, Paraf: "adm"},
	{Baris: 16, Tanggal: "10-10-2021", Kode: "K16", Keterangan: "Setoran Tunai", Debet: 0, Kredit: 100000, Saldo: 5047000, Paraf: "adm"},
}

// GetDisplayRows mengembalikan jumlah baris yang ditampilkan pada grid (sesuai template, minimal 16, maksimal 34)
func (w *PassbookCanvasWidget) GetDisplayRows() int {
	if w.tmpl != nil && w.tmpl.GetTotalRows() > 0 {
		r := w.tmpl.GetTotalRows()
		if r < 16 {
			return 16
		}
		if r > 34 {
			return 34
		}
		return r
	}
	return 34
}

// MakeColumnsSticky menata seluruh posisi kolom agar menempel rapat (sticky/contiguous) seperti spreadsheet Excel.
// Setiap kolom ke-j otomatis mulai tepat di akhir kolom sebelumnya (col_start[j] = col_start[j-1] + col_width[j-1]).
func MakeColumnsSticky(tmpl *config.LayoutTemplate) {
	if tmpl == nil || len(tmpl.Columns) == 0 {
		return
	}
	curr := tmpl.Columns[0].ColStart
	if curr < 0 {
		curr = 0
	}
	for i := range tmpl.Columns {
		tmpl.Columns[i].ColStart = curr
		if tmpl.Columns[i].ColWidth < 2 {
			tmpl.Columns[i].ColWidth = 2
		}
		curr += tmpl.Columns[i].ColWidth
	}
}

func getExcelColLetter(idx int) string {
	if idx < 0 {
		return "A"
	}
	if idx < 26 {
		return string(rune('A' + idx))
	}
	first := idx / 26
	second := idx % 26
	return string(rune('A'+first-1)) + string(rune('A'+second))
}

func formatIDRCurrency(val int64) string {
	if val == 0 {
		return "0,00"
	}
	sign := ""
	if val < 0 {
		sign = "-"
		val = -val
	}
	s := fmt.Sprintf("%d", val)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if len(s) > 0 {
		parts = append([]string{s}, parts...)
	}
	return sign + strings.Join(parts, ".") + ",00"
}

func isMonospaceFont(name string) bool {
	f := strings.ToLower(name)
	return strings.Contains(f, "courier") ||
		strings.Contains(f, "consolas") ||
		strings.Contains(f, "mono") ||
		strings.Contains(f, "lucida console") ||
		strings.Contains(f, "draft") ||
		strings.Contains(f, "esc/p")
}

// getSafeTextStyle mengembalikan TextStyle yang kompatibel dengan Fyne theme renderer
func getSafeTextStyle(isMono, isBold, isItalic bool) fyne.TextStyle {
	if isMono {
		// Fyne default theme tidak menyediakan font file Monospace-Bold atau Monospace-Italic
		return fyne.TextStyle{Monospace: true}
	}
	return fyne.TextStyle{
		Bold:   isBold,
		Italic: isItalic,
	}
}

func scaleFont(base float32, s float32) float32 {
	sz := base * s
	if sz < 7.0 {
		return 7.0
	}
	return sz
}

// PassbookCanvasWidget adalah custom interactive widget Fyne untuk mendesain layout buku tabungan berbasis spreadsheet Excel.
// Mendukung:
// 1. Tampilan ala spreadsheet Excel lengkap dengan header kolom [A, B, C...], nomor baris [01, 02...], dan gridlines sel
// 2. Drag pembatas kolom untuk resize lebar kolom (Excel Sticky resize)
// 3. Drag header kolom untuk menggeser posisi koordinat Tab X secara bebas
// 4. Drag sudut ⊞ (Select All Box) untuk menggeser Margin Kertas (Atas / Kiri)
// 5. Klik nomor baris untuk memilih baris awal target cetak (misal Baris 14)
// 6. Zoom in/out dan Pan kertas preview
type PassbookCanvasWidget struct {
	widget.BaseWidget

	tmpl           *config.LayoutTemplate
	selectedColIdx int
	targetRow      int
	stickyMode     bool

	// Zoom and Pan
	scale          float32
	panX           float32
	panY           float32
	initialPanX    float32
	initialPanY    float32
	OnScaleChanged func(scale float32)

	// Spreadsheet Drag Interaction State
	dragMode          int // 0: None, 1: Move Margins (Top/Left), 2: Sticky Excel Resize (border), 3: Move Left Margin, 4: Column Header Drag (Tab X), 5: Pan Viewport
	dragBorderIdx     int // 0: Left edge, 1..N: Border antara kolom i-1 dan kolom i
	dragTargetIdx     int // Kolom yang sedang digeser
	dragStartX        float32
	dragStartY        float32
	initialColWidths  []int
	initialColStarts  []int
	initialLeftMargin int
	initialTopMargin  int

	// Hover state
	hoverBorderIdx int // -1 if none, 0 for left edge, 1..N for column right borders
	hoverHeaderIdx int // -1 if none, 0..N for column headers
	hoverCorner    bool
	hoverRowIdx    int // -1 if none, 1..N
	currentCursor  desktop.Cursor

	// Callbacks
	OnSelected         func(colIdx int)
	OnTargetRowChanged func(row int)
	OnChanged          func()
}

func NewPassbookCanvasWidget(tmpl *config.LayoutTemplate, onSelected func(int), onChanged func()) *PassbookCanvasWidget {
	w := &PassbookCanvasWidget{
		tmpl:           tmpl,
		scale:          1.0,
		selectedColIdx: 0,
		targetRow:      14, // Default baris ke-14 sesuai standar simulasi
		stickyMode:     false,
		hoverBorderIdx: -1,
		hoverHeaderIdx: -1,
		hoverRowIdx:    -1,
		currentCursor:  desktop.DefaultCursor,
		OnSelected:     onSelected,
		OnChanged:      onChanged,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *PassbookCanvasWidget) SetTemplate(tmpl *config.LayoutTemplate) {
	w.tmpl = tmpl
	if w.selectedColIdx >= len(tmpl.Columns) {
		w.selectedColIdx = 0
	}
	w.Refresh()
}

func (w *PassbookCanvasWidget) SetSelectedColumn(idx int) {
	if idx >= 0 && idx < len(w.tmpl.Columns) {
		w.selectedColIdx = idx
		w.Refresh()
	}
}

func (w *PassbookCanvasWidget) SetTargetRow(row int) {
	if row >= 1 && row <= 34 {
		w.targetRow = row
		w.Refresh()
		if w.OnTargetRowChanged != nil {
			w.OnTargetRowChanged(w.targetRow)
		}
	}
}

func (w *PassbookCanvasWidget) GetTargetRow() int {
	if w.targetRow <= 0 {
		w.targetRow = 14
	}
	return w.targetRow
}

func (w *PassbookCanvasWidget) SetStickyMode(enabled bool) {
	w.stickyMode = enabled
	if enabled && w.tmpl != nil {
		MakeColumnsSticky(w.tmpl)
		w.Refresh()
		if w.OnChanged != nil {
			w.OnChanged()
		}
	}
}

func (w *PassbookCanvasWidget) IsStickyMode() bool {
	return w.stickyMode
}

func (w *PassbookCanvasWidget) AdjustSelectedWidth(delta int) {
	if w.tmpl == nil || w.selectedColIdx < 0 || w.selectedColIdx >= len(w.tmpl.Columns) {
		return
	}
	maxLimit := 100
	if w.tmpl.MaxColumns > 0 {
		maxLimit = w.tmpl.MaxColumns
	}
	col := &w.tmpl.Columns[w.selectedColIdx]
	newW := col.ColWidth + delta
	if newW < 2 {
		newW = 2
	} else if newW > maxLimit {
		newW = maxLimit
	}
	col.ColWidth = newW
	if w.stickyMode {
		MakeColumnsSticky(w.tmpl)
	}
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged()
	}
}

func (w *PassbookCanvasWidget) AdjustSelectedStart(delta int) {
	if w.tmpl == nil || w.selectedColIdx < 0 || w.selectedColIdx >= len(w.tmpl.Columns) {
		return
	}
	maxLimit := 100
	if w.tmpl.MaxColumns > 0 {
		maxLimit = w.tmpl.MaxColumns
	}
	col := &w.tmpl.Columns[w.selectedColIdx]
	newS := col.ColStart + delta
	if newS < 0 {
		newS = 0
	} else if newS > maxLimit {
		newS = maxLimit
	}
	col.ColStart = newS
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged()
	}
}

// Zoom and Pan methods
func (w *PassbookCanvasWidget) GetScale() float32 {
	if w.scale <= 0 {
		w.scale = 1.0
	}
	return w.scale
}

func (w *PassbookCanvasWidget) SetScale(s float32) {
	if s < 0.6 {
		s = 0.6
	} else if s > 2.2 {
		s = 2.2
	}
	w.scale = s
	w.Refresh()
	if w.OnScaleChanged != nil {
		w.OnScaleChanged(w.scale)
	}
}

func (w *PassbookCanvasWidget) ZoomIn() {
	w.SetScale(w.GetScale() + 0.15)
}

func (w *PassbookCanvasWidget) ZoomOut() {
	w.SetScale(w.GetScale() - 0.15)
}

func (w *PassbookCanvasWidget) ResetZoom() {
	w.scale = 1.0
	w.panX = 0
	w.panY = 0
	w.Refresh()
	if w.OnScaleChanged != nil {
		w.OnScaleChanged(w.scale)
	}
}

func (w *PassbookCanvasWidget) ResetPan() {
	w.panX = 0
	w.panY = 0
	w.Refresh()
}

// Implementasi desktop.Cursorable
func (w *PassbookCanvasWidget) Cursor() desktop.Cursor {
	return w.currentCursor
}

// Implementasi desktop.Hoverable
func (w *PassbookCanvasWidget) MouseIn(e *fyne.PointEvent) {
	w.updateCursorAndHover(e.Position)
}

func (w *PassbookCanvasWidget) MouseMoved(e *fyne.PointEvent) {
	if w.dragMode == 0 {
		w.updateCursorAndHover(e.Position)
	}
}

func (w *PassbookCanvasWidget) MouseOut() {
	if w.dragMode == 0 {
		w.hoverBorderIdx = -1
		w.hoverHeaderIdx = -1
		w.hoverCorner = false
		w.hoverRowIdx = -1
		w.currentCursor = desktop.DefaultCursor
	}
}

// Implementasi fyne.Scrollable (Mouse wheel zoom)
func (w *PassbookCanvasWidget) Scrolled(ev *fyne.ScrollEvent) {
	if ev.Scrolled.DY > 0 {
		w.ZoomIn()
	} else if ev.Scrolled.DY < 0 {
		w.ZoomOut()
	}
}

func (w *PassbookCanvasWidget) getGeometry() (paperX, paperY, paperW, paperH, printX, printW, charW, originY float32) {
	s := w.scale
	if s <= 0 {
		s = 1.0
	}

	paperX = 14 + w.panX
	paperY = 12 + w.panY

	maxCols := 100
	if w.tmpl != nil && w.tmpl.MaxColumns > 0 {
		maxCols = w.tmpl.MaxColumns
	} else if w.tmpl != nil && w.tmpl.FontPitchCPI > 0 {
		switch w.tmpl.FontPitchCPI {
		case 10:
			maxCols = 58 // 10 CPI Pica (Font Lebar)
		case 12:
			maxCols = 70 // 12 CPI Elite (Font Standar)
		case 15:
			maxCols = 85 // 15 CPI (Font Rapat)
		case 17:
			maxCols = 100 // 17 CPI Condensed (Standar Blangko Buku Tabungan)
		default:
			maxCols = 100
		}
	}

	rowHdrW := 28 * s
	charW = 7.2 * s
	printW = float32(maxCols) * charW
	paperW = printW + 32*s + rowHdrW
	printX = paperX + 16*s + rowHdrW

	topM := 0
	if w.tmpl != nil && w.tmpl.TopMargin > 0 {
		topM = w.tmpl.TopMargin
	}
	originY = paperY + 44*s + float32(topM)*22*s

	displayRows := w.GetDisplayRows()
	tableHdrHeight := 28 * s
	rowHeight := 22 * s
	tableTotalHeight := tableHdrHeight + float32(displayRows)*rowHeight
	paperH = (originY - paperY) + tableTotalHeight + 24*s
	return
}

func (w *PassbookCanvasWidget) updateCursorAndHover(pos fyne.Position) {
	if w.tmpl == nil || len(w.tmpl.Columns) == 0 {
		w.currentCursor = desktop.DefaultCursor
		return
	}

	s := w.scale
	if s <= 0 {
		s = 1.0
	}

	_, _, _, _, printX, _, charW, originY := w.getGeometry()
	leftM := w.tmpl.LeftMargin

	displayRows := w.GetDisplayRows()
	tableHdrHeight := 28 * s
	rowHeight := 22 * s
	rowHdrW := 28 * s
	tableTotalHeight := tableHdrHeight + float32(displayRows)*rowHeight

	tableLeftX := printX + float32(leftM+w.tmpl.Columns[0].ColStart)*charW
	rowHdrX := tableLeftX - rowHdrW

	// 1. Check Excel Top-Left Corner Box ⊞ (Geser Margin X / Y)
	if pos.X >= rowHdrX && pos.X <= rowHdrX+rowHdrW && pos.Y >= originY && pos.Y <= originY+tableHdrHeight {
		w.hoverCorner = true
		w.hoverBorderIdx = -1
		w.hoverHeaderIdx = -1
		w.hoverRowIdx = -1
		w.currentCursor = desktop.CrosshairCursor
		return
	}
	w.hoverCorner = false

	// 2. Check Row Headers (Kolom nomor baris sebelah kiri)
	if pos.X >= rowHdrX && pos.X <= rowHdrX+rowHdrW && pos.Y >= originY+tableHdrHeight && pos.Y <= originY+tableTotalHeight {
		rowIdx := int((pos.Y-(originY+tableHdrHeight))/rowHeight) + 1
		w.hoverRowIdx = rowIdx
		w.hoverBorderIdx = -1
		w.hoverHeaderIdx = -1
		w.currentCursor = desktop.PointerCursor
		return
	}
	w.hoverRowIdx = -1

	// 3. Check Spreadsheet Vertical Borders (Border 0: Left edge, Border 1..N: Column boundaries)
	detectedBorder := -1
	hitRadius := float64(9 * s)
	if hitRadius < 6 {
		hitRadius = 6
	}

	if pos.Y >= originY-4 && pos.Y <= originY+tableTotalHeight+4 {
		// Border 0: Sisi kiri tabel
		if math.Abs(float64(pos.X-tableLeftX)) <= hitRadius {
			detectedBorder = 0
		} else {
			// Border 1..N: Garis pembatas antar kolom
			for i, col := range w.tmpl.Columns {
				borderX := printX + float32(leftM+col.ColStart+col.ColWidth)*charW
				if math.Abs(float64(pos.X-borderX)) <= hitRadius {
					detectedBorder = i + 1
					break
				}
			}
		}
	}

	w.hoverBorderIdx = detectedBorder
	if detectedBorder >= 0 {
		w.hoverHeaderIdx = -1
		w.currentCursor = desktop.HResizeCursor
		return
	}

	// 4. Check Column Headers (Klik untuk pilih, drag untuk geser Tab X)
	if pos.Y >= originY && pos.Y <= originY+tableHdrHeight {
		for i, col := range w.tmpl.Columns {
			colStartX := printX + float32(col.ColStart+leftM)*charW
			colEndX := colStartX + float32(col.ColWidth)*charW
			if pos.X >= colStartX && pos.X <= colEndX {
				w.hoverHeaderIdx = i
				w.currentCursor = desktop.PointerCursor
				return
			}
		}
	}
	w.hoverHeaderIdx = -1

	// 5. In table cells
	if pos.Y >= originY+tableHdrHeight && pos.Y <= originY+tableTotalHeight && pos.X >= tableLeftX {
		lastCol := w.tmpl.Columns[len(w.tmpl.Columns)-1]
		tableRightX := printX + float32(leftM+lastCol.ColStart+lastCol.ColWidth)*charW
		if pos.X <= tableRightX {
			w.currentCursor = desktop.PointerCursor
			return
		}
	}

	w.currentCursor = desktop.DefaultCursor
}

// Implementasi fyne.Tappable
func (w *PassbookCanvasWidget) Tapped(e *fyne.PointEvent) {
	if w.tmpl == nil || len(w.tmpl.Columns) == 0 {
		return
	}

	s := w.scale
	if s <= 0 {
		s = 1.0
	}

	_, _, _, _, printX, _, charW, originY := w.getGeometry()
	leftM := w.tmpl.LeftMargin

	displayRows := w.GetDisplayRows()
	tableHdrHeight := 28 * s
	rowHeight := 22 * s
	rowHdrW := 28 * s
	tableTotalHeight := tableHdrHeight + float32(displayRows)*rowHeight

	tableLeftX := printX + float32(leftM+w.tmpl.Columns[0].ColStart)*charW
	rowHdrX := tableLeftX - rowHdrW

	// Klik pada Kolom Nomor Baris di sebelah kiri -> Ubah Target Baris Cetak
	if e.Position.X >= rowHdrX && e.Position.X <= rowHdrX+rowHdrW && e.Position.Y >= originY+tableHdrHeight && e.Position.Y <= originY+tableTotalHeight {
		clickedRow := int((e.Position.Y-(originY+tableHdrHeight))/rowHeight) + 1
		if clickedRow >= 1 && clickedRow <= displayRows {
			w.SetTargetRow(clickedRow)
			return
		}
	}

	// Klik pada Sel atau Header Kolom -> Pilih Kolom & Set Target Baris jika klik pada baris data
	if e.Position.Y >= originY && e.Position.Y <= originY+tableTotalHeight {
		for i, col := range w.tmpl.Columns {
			colStartX := printX + float32(col.ColStart+leftM)*charW
			colEndX := colStartX + float32(col.ColWidth)*charW

			if e.Position.X >= colStartX && e.Position.X <= colEndX {
				w.selectedColIdx = i
				if w.OnSelected != nil {
					w.OnSelected(i)
				}

				if e.Position.Y >= originY+tableHdrHeight {
					clickedRow := int((e.Position.Y-(originY+tableHdrHeight))/rowHeight) + 1
					if clickedRow >= 1 && clickedRow <= displayRows {
						w.SetTargetRow(clickedRow)
					}
				}

				w.Refresh()
				return
			}
		}
	}
}

// Implementasi fyne.Draggable
func (w *PassbookCanvasWidget) Dragged(e *fyne.DragEvent) {
	if w.tmpl == nil || len(w.tmpl.Columns) == 0 {
		return
	}

	s := w.scale
	if s <= 0 {
		s = 1.0
	}

	_, _, _, _, printX, _, charW, originY := w.getGeometry()
	leftM := w.tmpl.LeftMargin

	displayRowsDrag := w.GetDisplayRows()
	tableHdrHeight := 28 * s
	rowHeight := 22 * s
	rowHdrW := 28 * s
	tableTotalHeight := tableHdrHeight + float32(displayRowsDrag)*rowHeight

	tableLeftX := printX + float32(leftM+w.tmpl.Columns[0].ColStart)*charW
	rowHdrX := tableLeftX - rowHdrW

	hitRadius := float64(12 * s)
	if hitRadius < 8 {
		hitRadius = 8
	}

	// Inisialisasi awal saat pertama kali mouse di-drag
	if w.dragMode == 0 {
		w.dragStartX = e.Position.X
		w.dragStartY = e.Position.Y
		w.initialTopMargin = w.tmpl.TopMargin
		w.initialLeftMargin = w.tmpl.LeftMargin
		w.initialPanX = w.panX
		w.initialPanY = w.panY

		w.initialColWidths = make([]int, len(w.tmpl.Columns))
		w.initialColStarts = make([]int, len(w.tmpl.Columns))
		for i := range w.tmpl.Columns {
			w.initialColWidths[i] = w.tmpl.Columns[i].ColWidth
			w.initialColStarts[i] = w.tmpl.Columns[i].ColStart
		}

		// A. Check Excel Corner Box ⊞ (Geser seluruh margin tabel X & Y)
		if (e.Position.X >= rowHdrX-4 && e.Position.X <= rowHdrX+rowHdrW+4 && e.Position.Y >= originY-4 && e.Position.Y <= originY+tableHdrHeight+4) || w.hoverCorner {
			w.dragMode = 1
			return
		}

		// B. Check Borders & Headers
		if e.Position.Y >= originY-6 && e.Position.Y <= originY+tableTotalHeight+6 {
			// Border 0: Sisi kiri tabel
			if math.Abs(float64(e.Position.X-tableLeftX)) <= hitRadius {
				w.dragMode = 3 // Move Left Margin
				w.dragBorderIdx = 0
				return
			}

			// Border 1..N: Pembatas kolom (Excel Resize)
			hitBorder := -1
			for i, col := range w.tmpl.Columns {
				borderX := printX + float32(leftM+col.ColStart+col.ColWidth)*charW
				if math.Abs(float64(e.Position.X-borderX)) <= hitRadius {
					hitBorder = i + 1
					break
				}
			}

			if hitBorder >= 1 {
				w.dragMode = 2 // Sticky Excel Resize
				w.dragBorderIdx = hitBorder
				w.selectedColIdx = hitBorder - 1
				if w.OnSelected != nil {
					w.OnSelected(w.selectedColIdx)
				}
				w.Refresh()
				return
			}

			// C. Klik di body Header Kolom: Drag untuk GESER POSISI TAB X KOLOM
			if e.Position.Y >= originY && e.Position.Y <= originY+tableHdrHeight {
				for i, col := range w.tmpl.Columns {
					colStartX := printX + float32(col.ColStart+leftM)*charW
					colEndX := colStartX + float32(col.ColWidth)*charW
					if e.Position.X >= colStartX && e.Position.X <= colEndX {
						w.dragMode = 4 // Column Header Drag (Tab X)
						w.dragTargetIdx = i
						w.selectedColIdx = i
						if w.OnSelected != nil {
							w.OnSelected(i)
						}
						w.Refresh()
						return
					}
				}
			}
		}

		// D. Klik di luar elemen tabel: Drag untuk PAN VIEWPORT ("Geser Kertas")
		w.dragMode = 5
		return
	}

	// Active dragging handlers
	switch w.dragMode {
	case 1: // Geser Margin Keseluruhan Tabel (X & Y)
		deltaCols := int(math.Round(float64(e.Position.X-w.dragStartX) / float64(charW)))
		newLeftM := w.initialLeftMargin + deltaCols
		if newLeftM < 0 {
			newLeftM = 0
		} else if newLeftM > 35 {
			newLeftM = 35
		}

		deltaRows := int(math.Round(float64(e.Position.Y-w.dragStartY) / float64(rowHeight)))
		newTopM := w.initialTopMargin + deltaRows
		if newTopM < 0 {
			newTopM = 0
		} else if newTopM > 35 {
			newTopM = 35
		}

		if newLeftM != w.tmpl.LeftMargin || newTopM != w.tmpl.TopMargin {
			w.tmpl.LeftMargin = newLeftM
			w.tmpl.TopMargin = newTopM
			w.Refresh()
			if w.OnChanged != nil {
				w.OnChanged()
			}
		}

	case 2: // EXCEL BORDER RESIZE: Geser Garis Pembatas Kolom
		colIdx := w.dragBorderIdx - 1
		if colIdx >= 0 && colIdx < len(w.tmpl.Columns) {
			deltaCols := int(math.Round(float64(e.Position.X-w.dragStartX) / float64(charW)))
			newWidth := w.initialColWidths[colIdx] + deltaCols

			otherWidths := 0
			for i := range w.initialColWidths {
				if i != colIdx {
					otherWidths += w.tmpl.Columns[i].ColWidth
				}
			}

			maxAllowed := w.tmpl.MaxColumns - w.tmpl.LeftMargin - otherWidths
			if maxAllowed < 2 {
				maxAllowed = 2
			}
			if newWidth > maxAllowed {
				newWidth = maxAllowed
			}
			if newWidth < 2 {
				newWidth = 2
			}

			if newWidth != w.tmpl.Columns[colIdx].ColWidth {
				w.tmpl.Columns[colIdx].ColWidth = newWidth
				if w.stickyMode {
					MakeColumnsSticky(w.tmpl)
				}
				w.Refresh()
				if w.OnChanged != nil {
					w.OnChanged()
				}
			}
		}

	case 3: // Geser Border Paling Kiri Tabel (Menyesuaikan LeftMargin)
		deltaCols := int(math.Round(float64(e.Position.X-w.dragStartX) / float64(charW)))
		newLeft := w.initialLeftMargin + deltaCols
		if newLeft < 0 {
			newLeft = 0
		} else if newLeft > 35 {
			newLeft = 35
		}

		if newLeft != w.tmpl.LeftMargin {
			w.tmpl.LeftMargin = newLeft
			w.Refresh()
			if w.OnChanged != nil {
				w.OnChanged()
			}
		}

	case 4: // Drag Kolom Header: Geser koordinat ColStart kolom secara bebas (Tab Stop)
		curIdx := w.dragTargetIdx
		if curIdx >= 0 && curIdx < len(w.tmpl.Columns) {
			deltaCols := int(math.Round(float64(e.Position.X-w.dragStartX) / float64(charW)))
			newStart := w.initialColStarts[curIdx] + deltaCols
			if newStart < 0 {
				newStart = 0
			}
			maxColsLimit := 100
			if w.tmpl.MaxColumns > 0 {
				maxColsLimit = w.tmpl.MaxColumns
			}
			maxStart := maxColsLimit - w.tmpl.Columns[curIdx].ColWidth
			if maxStart < 0 {
				maxStart = 0
			}
			if newStart > maxStart {
				newStart = maxStart
			}
			if newStart != w.tmpl.Columns[curIdx].ColStart {
				w.tmpl.Columns[curIdx].ColStart = newStart
				w.Refresh()
				if w.OnChanged != nil {
					w.OnChanged()
				}
			}
		}

	case 5: // PAN VIEWPORT ("Geser Kertas")
		w.panX = w.initialPanX + (e.Position.X - w.dragStartX)
		w.panY = w.initialPanY + (e.Position.Y - w.dragStartY)
		w.Refresh()
	}
}

func (w *PassbookCanvasWidget) DragEnd() {
	w.dragMode = 0
	w.hoverBorderIdx = -1
	w.hoverHeaderIdx = -1
	w.hoverCorner = false
	w.hoverRowIdx = -1
	w.Refresh()
	if w.OnChanged != nil {
		w.OnChanged()
	}
}

func (w *PassbookCanvasWidget) MinSize() fyne.Size {
	w.ExtendBaseWidget(w)
	return fyne.NewSize(450, 350)
}

// Widget Renderer
func (w *PassbookCanvasWidget) CreateRenderer() fyne.WidgetRenderer {
	r := &passbookCanvasRenderer{
		widget: w,
	}
	r.rebuildObjects()
	return r
}

type passbookCanvasRenderer struct {
	widget  *PassbookCanvasWidget
	objects []fyne.CanvasObject
}

func (r *passbookCanvasRenderer) Destroy() {}

func (r *passbookCanvasRenderer) MinSize() fyne.Size {
	return fyne.NewSize(450, 350)
}

func (r *passbookCanvasRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *passbookCanvasRenderer) Layout(size fyne.Size) {
	// Objek canvas diposisikan secara absolut dalam rebuildObjects
}

func (r *passbookCanvasRenderer) Refresh() {
	r.rebuildObjects()
	canvas.Refresh(r.widget)
}

func (r *passbookCanvasRenderer) rebuildObjects() {
	w := r.widget
	tmpl := w.tmpl
	if tmpl == nil || len(tmpl.Columns) == 0 {
		r.objects = nil
		return
	}

	s := w.scale
	if s <= 0 {
		s = 1.0
	}

	var objs []fyne.CanvasObject

	// 0. Background Area Kerja (Workspace Canvas Background - Modern Slate)
	widgetSz := w.Size()
	if widgetSz.Width < 450 {
		widgetSz.Width = 450
	}
	if widgetSz.Height < 350 {
		widgetSz.Height = 350
	}
	canvasBg := canvas.NewRectangle(color.NRGBA{R: 241, G: 245, B: 249, A: 255})
	canvasBg.Resize(widgetSz)
	canvasBg.Move(fyne.NewPos(0, 0))
	objs = append(objs, canvasBg)

	paperX, paperY, paperW, paperH, printX, _, charW, originY := w.getGeometry()
	leftM := tmpl.LeftMargin

	// 1. Kertas Putih Blangko Buku Tabungan (Clean Sheet Viewport)
	paperShadow := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 25})
	paperShadow.Resize(fyne.NewSize(paperW, paperH))
	paperShadow.Move(fyne.NewPos(paperX+2*s, paperY+3*s))
	objs = append(objs, paperShadow)

	paperSheet := canvas.NewRectangle(color.White)
	paperSheet.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
	paperSheet.StrokeWidth = 1.5
	paperSheet.CornerRadius = 4 * s
	paperSheet.Resize(fyne.NewSize(paperW, paperH))
	paperSheet.Move(fyne.NewPos(paperX, paperY))
	objs = append(objs, paperSheet)

	// 2. Excel Ribbon / Formula Bar (Informasi Koordinat & Sel Aktif)
	ribbonH := 24 * s
	ribbonBg := canvas.NewRectangle(color.NRGBA{R: 248, G: 250, B: 252, A: 255})
	ribbonBg.StrokeColor = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
	ribbonBg.StrokeWidth = 1.0
	ribbonBg.CornerRadius = 3 * s
	ribbonBg.Resize(fyne.NewSize(paperW-24*s, ribbonH))
	ribbonBg.Move(fyne.NewPos(paperX+12*s, paperY+8*s))
	objs = append(objs, ribbonBg)

	// Name Box (Indikator Kolom Excel Aktif [A], [B] ...)
	selCol := tmpl.Columns[0]
	if w.selectedColIdx >= 0 && w.selectedColIdx < len(tmpl.Columns) {
		selCol = tmpl.Columns[w.selectedColIdx]
	}
	colLetter := getExcelColLetter(w.selectedColIdx)

	nameBoxBg := canvas.NewRectangle(color.NRGBA{R: 237, G: 242, B: 247, A: 255})
	nameBoxBg.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
	nameBoxBg.StrokeWidth = 1.0
	nameBoxBg.Resize(fyne.NewSize(88*s, ribbonH-4*s))
	nameBoxBg.Move(fyne.NewPos(paperX+14*s, paperY+10*s))
	objs = append(objs, nameBoxBg)

	nameBoxText := canvas.NewText(fmt.Sprintf("[%s] %s", colLetter, selCol.DisplayName), theme.PrimaryColor())
	nameBoxText.TextStyle = fyne.TextStyle{Bold: true}
	nameBoxText.TextSize = scaleFont(9.5, s)
	nameBoxText.Move(fyne.NewPos(paperX+18*s, paperY+12*s))
	objs = append(objs, nameBoxText)

	// Info formula / properti kolom aktif & spesifikasi buku tabungan
	colsCount := tmpl.MaxColumns
	if colsCount <= 0 {
		colsCount = 100
	}
	modeStr := "Rapat"
	if !w.stickyMode {
		modeStr = "Bebas"
	}
	fontStr := fmt.Sprintf("%s %dpt %s (%d CPI)", tmpl.GetFontFamily(), tmpl.GetFontSize(), tmpl.GetFontStyle(), tmpl.GetFontPitchCPI())
	infoStr := fmt.Sprintf("📄 %d Kolom × %d Baris  •  🔤 %s  •  Tab X: %d  •  Lebar: %d ch  •  Rata: %s  •  Mode: %s  •  Target: Baris %02d",
		colsCount, tmpl.GetTotalRows(), fontStr, selCol.ColStart, selCol.ColWidth, selCol.Align, modeStr, w.GetTargetRow())
	infoText := canvas.NewText(infoStr, color.NRGBA{R: 71, G: 85, B: 105, A: 255})
	infoText.TextSize = scaleFont(8.5, s)
	infoText.Move(fyne.NewPos(paperX+110*s, paperY+13*s))
	objs = append(objs, infoText)

	// Hitung geometri tabel secara dinamis sesuai total baris buku
	displayRows := w.GetDisplayRows()
	tableHdrHeight := 28 * s
	rowHeight := 22 * s
	rowHdrW := 28 * s
	tableTotalHeight := tableHdrHeight + float32(displayRows)*rowHeight

	tableLeftX := printX + float32(leftM+tmpl.Columns[0].ColStart)*charW
	rowHdrX := tableLeftX - rowHdrW
	lastCol := tmpl.Columns[len(tmpl.Columns)-1]
	tableRightX := printX + float32(leftM+lastCol.ColStart+lastCol.ColWidth)*charW
	tableTotalWidth := tableRightX - tableLeftX

	// 3. Excel Top-Left Corner Box ⊞ (Kotak Sudut Pilih Semua / Geser Margin)
	cornerBox := canvas.NewRectangle(color.NRGBA{R: 241, G: 245, B: 249, A: 255})
	if w.hoverCorner || w.dragMode == 1 {
		cornerBox.FillColor = color.NRGBA{R: 219, G: 234, B: 254, A: 255}
		cornerBox.StrokeColor = theme.PrimaryColor()
		cornerBox.StrokeWidth = 1.5
	} else {
		cornerBox.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
		cornerBox.StrokeWidth = 1.0
	}
	cornerBox.Resize(fyne.NewSize(rowHdrW, tableHdrHeight))
	cornerBox.Move(fyne.NewPos(rowHdrX, originY))
	objs = append(objs, cornerBox)

	cornerIcon := canvas.NewText("⊞", color.NRGBA{R: 100, G: 116, B: 139, A: 255})
	if w.hoverCorner || w.dragMode == 1 {
		cornerIcon.Color = theme.PrimaryColor()
	}
	cornerIcon.TextStyle = fyne.TextStyle{Bold: true}
	cornerIcon.TextSize = scaleFont(11, s)
	cornerIcon.Move(fyne.NewPos(rowHdrX+8*s, originY+6*s))
	objs = append(objs, cornerIcon)

	// 4. Excel Column Headers [A, B, C...] (Spreadsheet Header Row)
	for i, col := range tmpl.Columns {
		colStartX := printX + float32(col.ColStart+leftM)*charW
		colWidthPx := float32(col.ColWidth) * charW
		isSelected := (i == w.selectedColIdx)
		isDraggingThisCol := (w.dragMode == 4 && w.dragTargetIdx == i)
		isHovered := (w.hoverHeaderIdx == i)
		cLetter := getExcelColLetter(i)

		hdrBox := canvas.NewRectangle(color.NRGBA{R: 241, G: 245, B: 249, A: 255})
		if isDraggingThisCol {
			hdrBox.FillColor = color.NRGBA{R: 191, G: 219, B: 254, A: 255}
			hdrBox.StrokeColor = theme.PrimaryColor()
			hdrBox.StrokeWidth = 2
		} else if isSelected {
			hdrBox.FillColor = color.NRGBA{R: 219, G: 234, B: 254, A: 255}
			hdrBox.StrokeColor = theme.PrimaryColor()
			hdrBox.StrokeWidth = 1.5
		} else if isHovered {
			hdrBox.FillColor = color.NRGBA{R: 243, G: 244, B: 246, A: 255}
			hdrBox.StrokeColor = color.NRGBA{R: 148, G: 163, B: 184, A: 255}
			hdrBox.StrokeWidth = 1.0
		} else {
			hdrBox.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
			hdrBox.StrokeWidth = 1.0
		}
		hdrBox.Resize(fyne.NewSize(colWidthPx, tableHdrHeight))
		hdrBox.Move(fyne.NewPos(colStartX, originY))
		objs = append(objs, hdrBox)

		// Huruf Kolom Excel [A], [B]...
		letterLabel := canvas.NewText(cLetter, color.NRGBA{R: 100, G: 116, B: 139, A: 255})
		if isSelected || isDraggingThisCol {
			letterLabel.Color = theme.PrimaryColor()
		}
		letterLabel.TextStyle = fyne.TextStyle{Bold: true}
		letterLabel.TextSize = scaleFont(8.5, s)
		letterLabel.Move(fyne.NewPos(colStartX+4*s, originY+3*s))
		objs = append(objs, letterLabel)

		// Nama Tampilan Kolom (Header Label)
		dispName := col.DisplayName
		hdrText := canvas.NewText(dispName, color.NRGBA{R: 15, G: 23, B: 42, A: 255})
		hdrText.TextStyle = fyne.TextStyle{Bold: true}
		hdrText.TextSize = scaleFont(9.5, s)
		if isSelected || isDraggingThisCol {
			hdrText.Color = theme.PrimaryColor()
		}
		hdrText.Move(fyne.NewPos(colStartX+4*s, originY+13*s))
		objs = append(objs, hdrText)

		// Badge lebar karakter
		wBadge := canvas.NewText(fmt.Sprintf("%d ch", col.ColWidth), color.NRGBA{R: 148, G: 163, B: 184, A: 255})
		wBadge.TextSize = scaleFont(7.5, s)
		wBadge.Move(fyne.NewPos(colStartX+colWidthPx-28*s, originY+3*s))
		objs = append(objs, wBadge)
	}

	// 5. Excel Row Headers (Nomor Baris di Sebelah Kiri) & Spreadsheet Data Grid
	targetRow := w.GetTargetRow()

	for rIdx := 1; rIdx <= displayRows; rIdx++ {
		rowY := originY + tableHdrHeight + float32(rIdx-1)*rowHeight
		isTargetRow := (rIdx == targetRow)
		isHoveredRow := (w.hoverRowIdx == rIdx)

		// A. Header Baris Excel (Kolom Nomor Baris 01, 02...)
		rowHdrBox := canvas.NewRectangle(color.NRGBA{R: 248, G: 250, B: 252, A: 255})
		if isTargetRow {
			rowHdrBox.FillColor = color.NRGBA{R: 254, G: 240, B: 138, A: 255} // Highlight Target Baris Kuning Lembut
			rowHdrBox.StrokeColor = color.NRGBA{R: 202, G: 138, B: 4, A: 255}
			rowHdrBox.StrokeWidth = 1.2
		} else if isHoveredRow {
			rowHdrBox.FillColor = color.NRGBA{R: 241, G: 245, B: 249, A: 255}
			rowHdrBox.StrokeColor = color.NRGBA{R: 148, G: 163, B: 184, A: 255}
			rowHdrBox.StrokeWidth = 1.0
		} else {
			rowHdrBox.StrokeColor = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
			rowHdrBox.StrokeWidth = 0.8
		}
		rowHdrBox.Resize(fyne.NewSize(rowHdrW, rowHeight))
		rowHdrBox.Move(fyne.NewPos(rowHdrX, rowY))
		objs = append(objs, rowHdrBox)

		rowNumStr := fmt.Sprintf("%02d", rIdx)
		if isTargetRow {
			rowNumStr = fmt.Sprintf("▶%d", rIdx)
		}
		rowNumText := canvas.NewText(rowNumStr, color.NRGBA{R: 100, G: 116, B: 139, A: 255})
		if isTargetRow {
			rowNumText.Color = color.NRGBA{R: 161, G: 98, B: 7, A: 255}
		}
		rowNumText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		rowNumText.TextSize = scaleFont(8.5, s)
		rowNumText.Move(fyne.NewPos(rowHdrX+4*s, rowY+4*s))
		objs = append(objs, rowNumText)

		// B. Baris Data Transaksi Sampel (Data Cells)
		var sampleData *struct {
			Baris      int
			Tanggal    string
			Kode       string
			Keterangan string
			Debet      int64
			Kredit     int64
			Saldo      int64
			Paraf      string
		}
		if rIdx-1 < len(samplePassbookRows) {
			sampleData = &samplePassbookRows[rIdx-1]
		}

		// Background Baris Data
		rowBg := canvas.NewRectangle(color.White)
		if isTargetRow {
			rowBg.FillColor = color.NRGBA{R: 254, G: 252, B: 232, A: 255} // Highlight baris target cetak
		} else if rIdx%2 == 0 {
			rowBg.FillColor = color.NRGBA{R: 248, G: 250, B: 252, A: 160} // Zebra stripe halus
		}
		rowBg.Resize(fyne.NewSize(tableTotalWidth, rowHeight))
		rowBg.Move(fyne.NewPos(tableLeftX, rowY))
		objs = append(objs, rowBg)

		// Garis Grid Horizontal (Spreadsheet horizontal grid line)
		hLine := canvas.NewLine(color.NRGBA{R: 226, G: 232, B: 240, A: 255})
		hLine.Position1 = fyne.NewPos(tableLeftX, rowY+rowHeight)
		hLine.Position2 = fyne.NewPos(tableRightX, rowY+rowHeight)
		hLine.StrokeWidth = 0.8
		objs = append(objs, hLine)

		// Render Sel Data per Kolom
		for cIdx, col := range tmpl.Columns {
			colStartX := printX + float32(col.ColStart+leftM)*charW
			colWidthPx := float32(col.ColWidth) * charW
			isColSelected := (cIdx == w.selectedColIdx)

			// Highlight sel pada kolom yang sedang aktif terpilih
			if isColSelected && !isTargetRow {
				cellSelBg := canvas.NewRectangle(color.NRGBA{R: 240, G: 247, B: 255, A: 120})
				cellSelBg.Resize(fyne.NewSize(colWidthPx, rowHeight))
				cellSelBg.Move(fyne.NewPos(colStartX, rowY))
				objs = append(objs, cellSelBg)
			}

			// Teks isi data sel
			if sampleData != nil {
				valStr := getFormattedFieldValue(rIdx, sampleData, col.FieldKey)
				cellText := canvas.NewText(valStr, color.NRGBA{R: 15, G: 23, B: 42, A: 255})

				// Terapkan Monospace vs Proportional, Bold, Italic, dan ukuran font
				isMono := isMonospaceFont(tmpl.GetFontFamily())
				fStyleLower := strings.ToLower(tmpl.GetFontStyle())
				isBold := strings.Contains(fStyleLower, "bold")
				isItalic := strings.Contains(fStyleLower, "italic")

				cellText.TextStyle = getSafeTextStyle(isMono, isBold, isItalic)

				cellFontSize := float32(tmpl.GetFontSize())
				cellText.TextSize = scaleFont(cellFontSize, s)

				textW := float32(len([]rune(valStr))) * charW
				var textX float32
				switch col.Align {
				case "right":
					textX = colStartX + colWidthPx - textW - 4*s
					if textX < colStartX {
						textX = colStartX + 2*s
					}
				case "center":
					textX = colStartX + (colWidthPx-textW)/2
					if textX < colStartX {
						textX = colStartX + 2*s
					}
				default:
					textX = colStartX + 4*s
				}

				cellText.Move(fyne.NewPos(textX, rowY+4*s))
				objs = append(objs, cellText)
			}
		}

		// Badge penanda target cetak pada baris
		if isTargetRow {
			tagBg := canvas.NewRectangle(color.NRGBA{R: 254, G: 240, B: 138, A: 255})
			tagBg.StrokeColor = color.NRGBA{R: 202, G: 138, B: 4, A: 255}
			tagBg.StrokeWidth = 1.0
			tagBg.CornerRadius = 3 * s
			tagBg.Resize(fyne.NewSize(125*s, 16*s))
			tagBg.Move(fyne.NewPos(tableRightX+8*s, rowY+3*s))
			objs = append(objs, tagBg)

			tagText := canvas.NewText(fmt.Sprintf("▶ BARIS %02d TARGET", rIdx), color.NRGBA{R: 161, G: 98, B: 7, A: 255})
			tagText.TextStyle = fyne.TextStyle{Bold: true}
			tagText.TextSize = scaleFont(8.0, s)
			tagText.Move(fyne.NewPos(tableRightX+12*s, rowY+4*s))
			objs = append(objs, tagText)
		}
	}

	// 6. EXCEL SPREADSHEET VERTICAL GRIDLINES & RESIZE DIVIDERS
	// Garis batas paling kiri (Border 0)
	isBorder0Active := (w.hoverBorderIdx == 0 || w.dragMode == 3)
	leftBorderLine := canvas.NewLine(color.NRGBA{R: 203, G: 213, B: 225, A: 255})
	leftBorderLine.Position1 = fyne.NewPos(tableLeftX, originY)
	leftBorderLine.Position2 = fyne.NewPos(tableLeftX, originY+tableTotalHeight)
	leftBorderLine.StrokeWidth = 1.0

	if isBorder0Active {
		leftBorderLine.StrokeColor = theme.PrimaryColor()
		leftBorderLine.StrokeWidth = 2.5

		tip0 := canvas.NewText(fmt.Sprintf("⇄ Margin Kiri: %d kolom", tmpl.LeftMargin), theme.PrimaryColor())
		tip0.TextStyle = fyne.TextStyle{Bold: true}
		tip0.TextSize = scaleFont(9.0, s)
		tip0.Move(fyne.NewPos(tableLeftX-40*s, originY-16*s))
		objs = append(objs, tip0)
	}
	objs = append(objs, leftBorderLine)

	// Garis pembatas vertikal antar kolom (Border 1..N)
	for i, col := range tmpl.Columns {
		borderIdx := i + 1
		borderX := printX + float32(leftM+col.ColStart+col.ColWidth)*charW
		isBorderActive := (w.hoverBorderIdx == borderIdx || (w.dragMode == 2 && w.dragBorderIdx == borderIdx))

		vLine := canvas.NewLine(color.NRGBA{R: 226, G: 232, B: 240, A: 255})
		vLine.Position1 = fyne.NewPos(borderX, originY)
		vLine.Position2 = fyne.NewPos(borderX, originY+tableTotalHeight)
		vLine.StrokeWidth = 1.0

		// Handle grip visual di header kolom (Excel-style resize marker notch)
		divGrip := canvas.NewRectangle(color.NRGBA{R: 148, G: 163, B: 184, A: 200})
		divGrip.Resize(fyne.NewSize(2.5*s, 16*s))
		divGrip.Move(fyne.NewPos(borderX-1.25*s, originY+6*s))

		if isBorderActive {
			vLine.StrokeColor = theme.PrimaryColor()
			vLine.StrokeWidth = 2.5

			divGrip.FillColor = theme.PrimaryColor()
			divGrip.Resize(fyne.NewSize(4*s, 20*s))
			divGrip.Move(fyne.NewPos(borderX-2*s, originY+4*s))

			// Floating Tooltip Real-time saat digeser
			cLetter := getExcelColLetter(i)
			colMm := float32(col.ColWidth) * 2.54
			infoText := fmt.Sprintf("⇄ [%s] %s: Lebar %d ch (%0.1f mm)", cLetter, col.DisplayName, col.ColWidth, colMm)
			tipBg := canvas.NewRectangle(theme.PrimaryColor())
			tipBg.CornerRadius = 3 * s
			tipBg.Resize(fyne.NewSize(float32(len(infoText))*6.2*s+12*s, 18*s))
			tipBg.Move(fyne.NewPos(borderX-35*s, originY-22*s))
			objs = append(objs, tipBg)

			tipLabel := canvas.NewText(infoText, color.White)
			tipLabel.TextStyle = fyne.TextStyle{Bold: true}
			tipLabel.TextSize = scaleFont(8.5, s)
			tipLabel.Move(fyne.NewPos(borderX-30*s, originY-20*s))
			objs = append(objs, tipLabel)
		}

		objs = append(objs, vLine, divGrip)
	}

	// Garis penutup bawah tabel
	bottomLine := canvas.NewLine(color.NRGBA{R: 203, G: 213, B: 225, A: 255})
	bottomLine.Position1 = fyne.NewPos(rowHdrX, originY+tableTotalHeight)
	bottomLine.Position2 = fyne.NewPos(tableRightX, originY+tableTotalHeight)
	bottomLine.StrokeWidth = 1.2
	objs = append(objs, bottomLine)

	// Badge Informasi Spesifikasi Buku Tabungan di Sudut Bawah Kertas
	maxColsBadge := tmpl.MaxColumns
	if maxColsBadge <= 0 {
		maxColsBadge = 100
	}
	badgeText := fmt.Sprintf("📄 Format Buku: %d Baris  •  Max %d Kolom",
		tmpl.GetTotalRows(), maxColsBadge)
	badgeBg := canvas.NewRectangle(color.NRGBA{R: 241, G: 245, B: 249, A: 235})
	badgeBg.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
	badgeBg.StrokeWidth = 1.0
	badgeBg.CornerRadius = 3 * s
	badgeW := float32(len(badgeText))*5.6*s + 16*s
	badgeBg.Resize(fyne.NewSize(badgeW, 18*s))
	badgeBg.Move(fyne.NewPos(paperX+paperW-badgeW-14*s, paperY+paperH-24*s))
	objs = append(objs, badgeBg)

	badgeLabel := canvas.NewText(badgeText, color.NRGBA{R: 71, G: 85, B: 105, A: 255})
	badgeLabel.TextStyle = fyne.TextStyle{Bold: true}
	badgeLabel.TextSize = scaleFont(8.0, s)
	badgeLabel.Move(fyne.NewPos(paperX+paperW-badgeW-8*s, paperY+paperH-22*s))
	objs = append(objs, badgeLabel)

	r.objects = objs
}

func getFormattedFieldValue(rowIdx int, sRow *struct {
	Baris      int
	Tanggal    string
	Kode       string
	Keterangan string
	Debet      int64
	Kredit     int64
	Saldo      int64
	Paraf      string
}, key string) string {
	if sRow == nil {
		return ""
	}
	switch strings.ToLower(key) {
	case "no", "nomor", "baris":
		return fmt.Sprintf("%02d", rowIdx)
	case "tanggal", "tgl":
		return sRow.Tanggal
	case "kode_transaksi", "sandi", "kode":
		return sRow.Kode
	case "keterangan", "uraian", "jenis_transaksi", "jenis":
		return sRow.Keterangan
	case "debet":
		return formatIDRCurrency(sRow.Debet)
	case "kredit":
		return formatIDRCurrency(sRow.Kredit)
	case "saldo", "saldo_sesudah":
		return formatIDRCurrency(sRow.Saldo)
	case "petugas", "validasi", "teller", "paraf":
		return sRow.Paraf
	default:
		return "-"
	}
}
