package main

import (
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/escp"
	"passbook-bridge/internal/model"
)

// Sample transactions for realistic bank passbook preview
var samplePassbookRows = []struct {
	Baris int
	Req   *model.PrintRequest
}{
	{
		Baris: 1,
		Req: &model.PrintRequest{
			Tanggal:        "02/09/24",
			KodeTransaksi:  "101",
			JenisTransaksi: "setor",
			Jumlah:         5000000,
			SaldoSesudah:   5000000,
			Petugas:        "TLR01",
			Keterangan:     "SETORAN AWAL TUNAI",
		},
	},
	{
		Baris: 2,
		Req: &model.PrintRequest{
			Tanggal:        "05/09/24",
			KodeTransaksi:  "104",
			JenisTransaksi: "tarik",
			Jumlah:         550000,
			SaldoSesudah:   4450000,
			Petugas:        "ATM09",
			Keterangan:     "PENARIKAN TUNAI ATM",
		},
	},
	{
		Baris: 3,
		Req: &model.PrintRequest{
			Tanggal:        "12/09/24",
			KodeTransaksi:  "102",
			JenisTransaksi: "setor",
			Jumlah:         11000000,
			SaldoSesudah:   15450000,
			Petugas:        "TLR02",
			Keterangan:     "SETORAN TUNAI TELLER",
		},
	},
	{
		Baris: 14,
		Req: &model.PrintRequest{
			Tanggal:        "20/10/26",
			KodeTransaksi:  "SET",
			JenisTransaksi: "setor",
			Jumlah:         150000,
			SaldoSesudah:   15600000,
			Petugas:        "TELLER01",
			Keterangan:     "Setoran Uji Coba Baris 14",
		},
	},
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

func scaleFont(base float32, s float32) float32 {
	sz := base * s
	if sz < 7.5 {
		return 7.5
	}
	return sz
}

// PassbookCanvasWidget adalah custom interactive widget Fyne untuk mendesain layout buku tabungan berbasis spreadsheet.
// Mendukung:
// 1. Zoom (Perbesar/Perkecil) skala kertas passbook
// 2. Pan (Geser-geser) posisi kertas di atas canvas
// 3. Drag border antar kolom untuk resize lebar kolom (sticky seperti Excel)
// 4. Drag header kolom untuk swap/reorder urutan kolom
// 5. Drag border paling kiri atau Grip Bar untuk geser posisi seluruh tabel (X/Y)
type PassbookCanvasWidget struct {
	widget.BaseWidget

	tmpl           *config.LayoutTemplate
	selectedColIdx int

	// Zoom and Pan
	scale          float32
	panX           float32
	panY           float32
	initialPanX    float32
	initialPanY    float32
	OnScaleChanged func(scale float32)

	// Spreadsheet Drag Interaction State
	dragMode          int // 0: None, 1: Move whole table (X/Y), 2: Sticky Excel Resize (border), 3: Move Left Margin, 4: Column Reorder, 5: Pan Viewport
	dragBorderIdx     int // 0: Left edge, 1..N: Border antara kolom i-1 dan kolom i
	dragTargetIdx     int // Kolom yang sedang di-reorder
	dragStartX        float32
	dragStartY        float32
	initialColWidths  []int
	initialColStarts  []int
	initialLeftMargin int
	initialTopMargin  int

	// Hover state
	hoverBorderIdx int // -1 if none, 0 for left edge, 1..N for column right borders
	hoverGrip      bool
	currentCursor  desktop.Cursor

	// Callbacks
	OnSelected func(colIdx int)
	OnChanged  func()
}

func NewPassbookCanvasWidget(tmpl *config.LayoutTemplate, onSelected func(int), onChanged func()) *PassbookCanvasWidget {
	MakeColumnsSticky(tmpl)
	w := &PassbookCanvasWidget{
		tmpl:           tmpl,
		scale:          1.0,
		selectedColIdx: 0,
		hoverBorderIdx: -1,
		currentCursor:  desktop.DefaultCursor,
		OnSelected:     onSelected,
		OnChanged:      onChanged,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *PassbookCanvasWidget) SetTemplate(tmpl *config.LayoutTemplate) {
	w.tmpl = tmpl
	MakeColumnsSticky(w.tmpl)
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
		w.hoverGrip = false
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

	paperX = 24 + w.panX
	paperY = 20 + w.panY
	paperW = 740 * s
	paperH = 490 * s

	printX = paperX + 22*s
	printW = paperW - 44*s // printable width

	maxCols := 80
	if w.tmpl != nil && w.tmpl.MaxColumns > 0 {
		maxCols = w.tmpl.MaxColumns
	}
	charW = printW / float32(maxCols)

	topM := 0
	if w.tmpl != nil && w.tmpl.TopMargin > 0 {
		topM = w.tmpl.TopMargin
	}
	originY = paperY + (110+float32(topM)*13)*s
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

	_, _, _, _, printX, printW, charW, originY := w.getGeometry()
	leftM := w.tmpl.LeftMargin

	gripY := originY - 26*s
	gripH := 22 * s

	// Check Grip Bar
	if pos.Y >= gripY && pos.Y <= gripY+gripH && pos.X >= printX && pos.X <= printX+printW {
		w.hoverGrip = true
		w.hoverBorderIdx = -1
		w.currentCursor = desktop.CrosshairCursor
		return
	}
	w.hoverGrip = false

	// Check Spreadsheet Vertical Borders (Border 0: Left edge, Border 1..N: Column boundaries)
	tableHdrHeight := 26 * s
	totalRowsHeight := float32(len(samplePassbookRows)) * 24 * s
	tableTop := originY
	tableBottom := originY + tableHdrHeight + totalRowsHeight
	detectedBorder := -1

	hitRadius := float64(12 * s)
	if hitRadius < 8 {
		hitRadius = 8
	}

	if pos.Y >= tableTop-6 && pos.Y <= tableBottom+6 {
		// Border 0: Sisi paling kiri kolom pertama
		leftEdgeX := printX + float32(leftM+w.tmpl.Columns[0].ColStart)*charW
		if math.Abs(float64(pos.X-leftEdgeX)) <= hitRadius {
			detectedBorder = 0
		} else {
			// Border 1..N: Garis pembatas kanan masing-masing kolom
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
		w.currentCursor = desktop.HResizeCursor
	} else if pos.Y >= originY && pos.Y <= originY+tableHdrHeight {
		// Over column header cell body: pointer cursor
		for _, col := range w.tmpl.Columns {
			colStartX := printX + float32(col.ColStart+leftM)*charW
			colEndX := colStartX + float32(col.ColWidth)*charW
			if pos.X >= colStartX && pos.X <= colEndX {
				w.currentCursor = desktop.PointerCursor
				return
			}
		}
		w.currentCursor = desktop.DefaultCursor
	} else {
		w.currentCursor = desktop.DefaultCursor
	}
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

	tableHdrHeight := 26 * s
	totalRowsHeight := float32(len(samplePassbookRows)) * 24 * s
	tableTop := originY
	tableBottom := originY + tableHdrHeight + totalRowsHeight

	if e.Position.Y >= tableTop && e.Position.Y <= tableBottom {
		for i, col := range w.tmpl.Columns {
			colStartX := printX + float32(col.ColStart+leftM)*charW
			colEndX := colStartX + float32(col.ColWidth)*charW

			if e.Position.X >= colStartX && e.Position.X <= colEndX {
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

// Implementasi fyne.Draggable
func (w *PassbookCanvasWidget) Dragged(e *fyne.DragEvent) {
	if w.tmpl == nil || len(w.tmpl.Columns) == 0 {
		return
	}

	s := w.scale
	if s <= 0 {
		s = 1.0
	}

	_, _, _, _, printX, printW, charW, originY := w.getGeometry()
	leftM := w.tmpl.LeftMargin

	tableHdrHeight := 26 * s
	totalRowsHeight := float32(len(samplePassbookRows)) * 24 * s
	tableTotalHeight := tableHdrHeight + totalRowsHeight

	hitRadius := float64(14 * s)
	if hitRadius < 10 {
		hitRadius = 10
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

		// A. Check Grip Bar (Geser seluruh tabel X & Y)
		gripY := originY - 26*s
		gripH := 22 * s
		if (e.Position.Y >= gripY-4 && e.Position.Y <= gripY+gripH+4 && e.Position.X >= printX-10 && e.Position.X <= printX+printW+10) || w.hoverGrip {
			w.dragMode = 1
			return
		}

		// B. Check Borders & Headers
		if e.Position.Y >= originY-6 && e.Position.Y <= originY+tableTotalHeight+6 {
			// Border 0: Sisi kiri tabel
			leftEdgeX := printX + float32(leftM+w.tmpl.Columns[0].ColStart)*charW
			if math.Abs(float64(e.Position.X-leftEdgeX)) <= hitRadius {
				w.dragMode = 3 // Move Left Margin
				w.dragBorderIdx = 0
				return
			}

			// Border 1..N: Pembatas kolom (Sticky Excel Resize)
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

			// C. Klik di body Header Kolom: Drag untuk REORDER / GESER KOLOM
			if e.Position.Y >= originY && e.Position.Y <= originY+tableHdrHeight {
				for i, col := range w.tmpl.Columns {
					colStartX := printX + float32(col.ColStart+leftM)*charW
					colEndX := colStartX + float32(col.ColWidth)*charW
					if e.Position.X >= colStartX && e.Position.X <= colEndX {
						w.dragMode = 4 // Column Reorder Drag
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
	case 1: // Geser Posisi Seluruh Tabel (X & Y)
		deltaCols := int(math.Round(float64(e.Position.X-w.dragStartX) / float64(charW)))
		newLeftM := w.initialLeftMargin + deltaCols
		if newLeftM < 0 {
			newLeftM = 0
		} else if newLeftM > 35 {
			newLeftM = 35
		}

		deltaRows := int(math.Round(float64(e.Position.Y-w.dragStartY) / float64(13.0*s)))
		newTopM := w.initialTopMargin + deltaRows
		if newTopM < 0 {
			newTopM = 0
		} else if newTopM > 18 {
			newTopM = 18
		}

		if newLeftM != w.tmpl.LeftMargin || newTopM != w.tmpl.TopMargin {
			w.tmpl.LeftMargin = newLeftM
			w.tmpl.TopMargin = newTopM
			w.Refresh()
			if w.OnChanged != nil {
				w.OnChanged()
			}
		}

	case 2: // STICKY EXCEL BORDER RESIZE: Geser Border Kolom
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
				MakeColumnsSticky(w.tmpl)
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

	case 4: // EXCEL-STYLE COLUMN REORDER (Swap Kolom saat Header Di-drag)
		curIdx := w.dragTargetIdx
		if curIdx >= 0 && curIdx < len(w.tmpl.Columns) {
			if curIdx < len(w.tmpl.Columns)-1 {
				nextCol := w.tmpl.Columns[curIdx+1]
				nextCenterX := printX + float32(leftM+nextCol.ColStart+(nextCol.ColWidth/2))*charW
				if e.Position.X > nextCenterX {
					w.tmpl.Columns[curIdx], w.tmpl.Columns[curIdx+1] = w.tmpl.Columns[curIdx+1], w.tmpl.Columns[curIdx]
					w.dragTargetIdx = curIdx + 1
					w.selectedColIdx = w.dragTargetIdx
					MakeColumnsSticky(w.tmpl)
					w.Refresh()
					if w.OnChanged != nil {
						w.OnChanged()
					}
					return
				}
			}

			if curIdx > 0 {
				prevCol := w.tmpl.Columns[curIdx-1]
				prevCenterX := printX + float32(leftM+prevCol.ColStart+(prevCol.ColWidth/2))*charW
				if e.Position.X < prevCenterX {
					w.tmpl.Columns[curIdx], w.tmpl.Columns[curIdx-1] = w.tmpl.Columns[curIdx-1], w.tmpl.Columns[curIdx]
					w.dragTargetIdx = curIdx - 1
					w.selectedColIdx = w.dragTargetIdx
					MakeColumnsSticky(w.tmpl)
					w.Refresh()
					if w.OnChanged != nil {
						w.OnChanged()
					}
					return
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
	w.hoverGrip = false
	MakeColumnsSticky(w.tmpl)
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

	// 0. Background Area Kerja (Workspace Canvas Background)
	widgetSz := w.Size()
	if widgetSz.Width < 450 {
		widgetSz.Width = 450
	}
	if widgetSz.Height < 350 {
		widgetSz.Height = 350
	}
	canvasBg := canvas.NewRectangle(color.NRGBA{R: 242, G: 244, B: 248, A: 255})
	canvasBg.Resize(widgetSz)
	canvasBg.Move(fyne.NewPos(0, 0))
	objs = append(objs, canvasBg)

	paperX, paperY, paperW, paperH, printX, printW, charW, originY := w.getGeometry()
	leftM := tmpl.LeftMargin

	// 1. Kertas Buku Tabungan (Passbook Paper Sheet & Shadow)
	paperShadow := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 30})
	paperShadow.Resize(fyne.NewSize(paperW, paperH))
	paperShadow.Move(fyne.NewPos(paperX+4*s, paperY+5*s))
	objs = append(objs, paperShadow)

	paperSheet := canvas.NewRectangle(color.NRGBA{R: 254, G: 252, B: 245, A: 255})
	paperSheet.StrokeColor = color.NRGBA{R: 210, G: 200, B: 175, A: 255}
	paperSheet.StrokeWidth = 1.5
	paperSheet.CornerRadius = 4 * s
	paperSheet.Resize(fyne.NewSize(paperW, paperH))
	paperSheet.Move(fyne.NewPos(paperX, paperY))
	objs = append(objs, paperSheet)

	// Garis Lipatan Tengah Buku (Spine Crease)
	spineX := paperX + paperW/2
	spineLine := canvas.NewLine(color.NRGBA{R: 180, G: 170, B: 145, A: 110})
	spineLine.Position1 = fyne.NewPos(spineX, paperY+6*s)
	spineLine.Position2 = fyne.NewPos(spineX, paperY+paperH-6*s)
	spineLine.StrokeWidth = 1.5
	objs = append(objs, spineLine)

	// 2. Header Buku Tabungan (Bank Identity)
	bankBadge := canvas.NewRectangle(theme.PrimaryColor())
	bankBadge.CornerRadius = 3 * s
	bankBadge.Resize(fyne.NewSize(42*s, 30*s))
	bankBadge.Move(fyne.NewPos(paperX+18*s, paperY+16*s))
	objs = append(objs, bankBadge)

	badgeText := canvas.NewText(extractBankCode(tmpl.Name), color.White)
	badgeText.TextStyle = fyne.TextStyle{Bold: true}
	badgeText.TextSize = scaleFont(12, s)
	badgeText.Alignment = fyne.TextAlignCenter
	badgeText.Move(fyne.NewPos(paperX+22*s, paperY+22*s))
	objs = append(objs, badgeText)

	bankTitle := canvas.NewText(tmpl.Name, theme.ForegroundColor())
	bankTitle.TextStyle = fyne.TextStyle{Bold: true}
	bankTitle.TextSize = scaleFont(13, s)
	bankTitle.Move(fyne.NewPos(paperX+66*s, paperY+16*s))
	objs = append(objs, bankTitle)

	bankSub := canvas.NewText("BUKU TABUNGAN NASABAH • GRID SPREADSHEET BORDER INTERAKTIF", color.NRGBA{R: 120, G: 110, B: 95, A: 255})
	bankSub.TextSize = scaleFont(9, s)
	bankSub.Move(fyne.NewPos(paperX+66*s, paperY+32*s))
	objs = append(objs, bankSub)

	accText := canvas.NewText("NO. REK: 0341-01-002891-50-8", theme.PrimaryColor())
	accText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	accText.TextSize = scaleFont(10, s)
	accText.Move(fyne.NewPos(paperX+paperW-220*s, paperY+16*s))
	objs = append(objs, accText)

	halText := canvas.NewText(fmt.Sprintf("Margin Atas: %d baris • Margin Kiri: %d kol", tmpl.TopMargin, tmpl.LeftMargin), color.NRGBA{R: 120, G: 110, B: 95, A: 255})
	halText.TextSize = scaleFont(9, s)
	halText.Move(fyne.NewPos(paperX+paperW-240*s, paperY+32*s))
	objs = append(objs, halText)

	// 3. Penggaris Karakter ESC/P (Top Character Ruler)
	rulerBg := canvas.NewRectangle(color.NRGBA{R: 240, G: 236, B: 222, A: 255})
	rulerBg.Resize(fyne.NewSize(printW, 20*s))
	rulerBg.Move(fyne.NewPos(printX, paperY+52*s))
	objs = append(objs, rulerBg)

	for c := 0; c <= tmpl.MaxColumns; c += 5 {
		cX := printX + float32(c)*charW
		tick := canvas.NewLine(color.NRGBA{R: 160, G: 150, B: 130, A: 255})
		tick.Position1 = fyne.NewPos(cX, paperY+60*s)
		tick.Position2 = fyne.NewPos(cX, paperY+70*s)
		objs = append(objs, tick)

		if c%10 == 0 {
			num := canvas.NewText(fmt.Sprintf("%d", c), color.NRGBA{R: 100, G: 90, B: 75, A: 255})
			num.TextSize = scaleFont(8.5, s)
			num.TextStyle = fyne.TextStyle{Monospace: true}
			num.Move(fyne.NewPos(cX-4*s, paperY+52*s))
			objs = append(objs, num)
		}
	}

	// 4. Grip Bar / Pegangan Geser Posisi Seluruh Tabel (X & Y)
	gripY := originY - 26*s
	gripBg := canvas.NewRectangle(color.NRGBA{R: 228, G: 236, B: 252, A: 255})
	if w.hoverGrip || w.dragMode == 1 {
		gripBg.FillColor = color.NRGBA{R: 195, G: 218, B: 255, A: 255}
	}
	gripBg.StrokeColor = theme.PrimaryColor()
	gripBg.StrokeWidth = 1
	gripBg.CornerRadius = 3 * s
	gripBg.Resize(fyne.NewSize(printW, 22*s))
	gripBg.Move(fyne.NewPos(printX, gripY))
	objs = append(objs, gripBg)

	gripText := canvas.NewText(fmt.Sprintf("::: GESER POSISI TABEL BUKU TABUNGAN (Drag X/Y) • [Top: %d baris, Left: %d kol] • Drag garis pembatas untuk ubah lebar kolom (Excel)", tmpl.TopMargin, tmpl.LeftMargin), theme.PrimaryColor())
	gripText.TextStyle = fyne.TextStyle{Bold: true}
	gripText.TextSize = scaleFont(9.5, s)
	gripText.Move(fyne.NewPos(printX+10*s, gripY+4*s))
	objs = append(objs, gripText)

	// Hitung total lebar tabel (dari awal kolom 0 sampai akhir kolom terakhir)
	tableHdrHeight := 26 * s
	totalRowsHeight := float32(len(samplePassbookRows)) * 24 * s
	tableTotalHeight := tableHdrHeight + totalRowsHeight

	tableLeftX := printX + float32(leftM+tmpl.Columns[0].ColStart)*charW
	lastCol := tmpl.Columns[len(tmpl.Columns)-1]
	tableRightX := printX + float32(leftM+lastCol.ColStart+lastCol.ColWidth)*charW
	tableTotalWidth := tableRightX - tableLeftX

	// Background seluruh area tabel
	tableFullBg := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 140})
	tableFullBg.Resize(fyne.NewSize(tableTotalWidth, tableTotalHeight))
	tableFullBg.Move(fyne.NewPos(tableLeftX, originY))
	objs = append(objs, tableFullBg)

	// 5. Header Kolom Tabel Spreadsheet (Header Cells)
	for i, col := range tmpl.Columns {
		colStartX := printX + float32(col.ColStart+leftM)*charW
		colWidthPx := float32(col.ColWidth) * charW
		isSelected := (i == w.selectedColIdx)
		isDraggingThisCol := (w.dragMode == 4 && w.dragTargetIdx == i)

		hdrBox := canvas.NewRectangle(color.NRGBA{R: 242, G: 238, B: 226, A: 255})
		if isDraggingThisCol {
			hdrBox.FillColor = color.NRGBA{R: 185, G: 215, B: 255, A: 255}
			hdrBox.StrokeColor = theme.PrimaryColor()
			hdrBox.StrokeWidth = 2
		} else if isSelected {
			hdrBox.FillColor = color.NRGBA{R: 215, G: 230, B: 255, A: 255}
			hdrBox.StrokeColor = theme.PrimaryColor()
			hdrBox.StrokeWidth = 1.5
		} else {
			hdrBox.StrokeColor = color.NRGBA{R: 200, G: 190, B: 170, A: 255}
			hdrBox.StrokeWidth = 0.8
		}
		hdrBox.Resize(fyne.NewSize(colWidthPx, tableHdrHeight))
		hdrBox.Move(fyne.NewPos(colStartX, originY))
		objs = append(objs, hdrBox)

		// Teks Nama Kolom
		dispName := col.DisplayName
		hdrText := canvas.NewText(dispName, theme.ForegroundColor())
		hdrText.TextStyle = fyne.TextStyle{Bold: true}
		hdrText.TextSize = scaleFont(9.5, s)
		if isSelected || isDraggingThisCol {
			hdrText.Color = theme.PrimaryColor()
		}
		hdrText.Move(fyne.NewPos(colStartX+4*s, originY+4*s))
		objs = append(objs, hdrText)

		// Badge lebar karakter
		wBadge := canvas.NewText(fmt.Sprintf("%d ch", col.ColWidth), color.NRGBA{R: 120, G: 110, B: 95, A: 255})
		wBadge.TextSize = scaleFont(8, s)
		wBadge.Move(fyne.NewPos(colStartX+4*s, originY+15*s))
		objs = append(objs, wBadge)
	}

	// 6. Baris-Baris Data Transaksi Sampel (Data Rows)
	rowY := originY + tableHdrHeight
	for rIdx, sRow := range samplePassbookRows {
		isTargetRow := (sRow.Baris == 14)
		rHeight := 24 * s

		// Background baris
		rowBg := canvas.NewRectangle(color.Transparent)
		if isTargetRow {
			rowBg.FillColor = color.NRGBA{R: 255, G: 248, B: 220, A: 220}
		} else if rIdx%2 == 1 {
			rowBg.FillColor = color.NRGBA{R: 247, G: 244, B: 235, A: 160}
		}
		rowBg.Resize(fyne.NewSize(tableTotalWidth, rHeight))
		rowBg.Move(fyne.NewPos(tableLeftX, rowY))
		objs = append(objs, rowBg)

		// Garis pemisah horizontal antar baris (spreadsheet row grid line)
		hLine := canvas.NewLine(color.NRGBA{R: 220, G: 212, B: 195, A: 255})
		hLine.Position1 = fyne.NewPos(tableLeftX, rowY+rHeight)
		hLine.Position2 = fyne.NewPos(tableRightX, rowY+rHeight)
		hLine.StrokeWidth = 0.8
		objs = append(objs, hLine)

		// Nomor Baris Fisik
		numText := canvas.NewText(fmt.Sprintf("%02d", sRow.Baris), color.NRGBA{R: 140, G: 130, B: 110, A: 255})
		numText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		numText.TextSize = scaleFont(9, s)
		numText.Move(fyne.NewPos(tableLeftX-18*s, rowY+5*s))
		objs = append(objs, numText)

		// Format teks data per kolom
		for _, col := range tmpl.Columns {
			colStartX := printX + float32(col.ColStart+leftM)*charW
			valStr := getFormattedFieldValue(sRow.Req, col.FieldKey)

			cellText := canvas.NewText(valStr, color.NRGBA{R: 25, G: 35, B: 45, A: 255})
			cellText.TextStyle = fyne.TextStyle{Monospace: true}
			cellText.TextSize = scaleFont(10, s)

			textW := float32(len([]rune(valStr))) * charW
			var textX float32
			switch col.Align {
			case "right":
				textX = colStartX + (float32(col.ColWidth) * charW) - textW - 4*s
				if textX < colStartX {
					textX = colStartX + 2*s
				}
			case "center":
				textX = colStartX + ((float32(col.ColWidth)*charW)-textW)/2
				if textX < colStartX {
					textX = colStartX + 2*s
				}
			default:
				textX = colStartX + 4*s
			}

			cellText.Move(fyne.NewPos(textX, rowY+5*s))
			objs = append(objs, cellText)
		}

		if isTargetRow {
			tag := canvas.NewText("[BARIS 14 TARGET CETAK]", color.NRGBA{R: 190, G: 120, B: 0, A: 255})
			tag.TextStyle = fyne.TextStyle{Bold: true}
			tag.TextSize = scaleFont(8.5, s)
			tag.Move(fyne.NewPos(tableRightX+8*s, rowY+6*s))
			objs = append(objs, tag)
		}

		rowY += rHeight
	}

	// 7. SPREADSHEET VERTICAL BORDERS (Sticky Column Dividers)
	// --- A. Border 0 (Sisi Paling Kiri) ---
	isBorder0Active := (w.hoverBorderIdx == 0 || (w.dragMode == 3))
	leftBorderLine := canvas.NewLine(color.NRGBA{R: 175, G: 165, B: 140, A: 255})
	leftBorderLine.Position1 = fyne.NewPos(tableLeftX, originY)
	leftBorderLine.Position2 = fyne.NewPos(tableLeftX, originY+tableTotalHeight)
	leftBorderLine.StrokeWidth = 1.0

	if isBorder0Active {
		leftBorderLine.StrokeColor = theme.PrimaryColor()
		leftBorderLine.StrokeWidth = 3.0

		tip0 := canvas.NewText(fmt.Sprintf("⇄ Margin Kiri Tabel: %d kol", tmpl.LeftMargin), theme.PrimaryColor())
		tip0.TextStyle = fyne.TextStyle{Bold: true}
		tip0.TextSize = scaleFont(9.5, s)
		tip0.Move(fyne.NewPos(tableLeftX-40*s, originY-16*s))
		objs = append(objs, tip0)
	}
	objs = append(objs, leftBorderLine)

	// --- B. Border 1..N (Pembatas Kolom Sticky Seperti Excel) ---
	for i, col := range tmpl.Columns {
		borderIdx := i + 1
		borderX := printX + float32(leftM+col.ColStart+col.ColWidth)*charW
		isBorderActive := (w.hoverBorderIdx == borderIdx || (w.dragMode == 2 && w.dragBorderIdx == borderIdx))

		vLine := canvas.NewLine(color.NRGBA{R: 180, G: 170, B: 150, A: 255})
		vLine.Position1 = fyne.NewPos(borderX, originY)
		vLine.Position2 = fyne.NewPos(borderX, originY+tableTotalHeight)
		vLine.StrokeWidth = 1.0

		// Handle grip visual di header kolom (Excel-style resize marker)
		divGrip := canvas.NewRectangle(color.NRGBA{R: 160, G: 150, B: 130, A: 220})
		divGrip.Resize(fyne.NewSize(3*s, 14*s))
		divGrip.Move(fyne.NewPos(borderX-1.5*s, originY+6*s))

		if isBorderActive {
			vLine.StrokeColor = theme.PrimaryColor()
			vLine.StrokeWidth = 3.0

			divGrip.FillColor = theme.PrimaryColor()
			divGrip.Resize(fyne.NewSize(5*s, 18*s))
			divGrip.Move(fyne.NewPos(borderX-2.5*s, originY+4*s))

			// Floating Tooltip Real-time
			mmWidth := float32(col.ColWidth) * 2.54
			infoText := fmt.Sprintf("⇄ %s: Lebar %d ch (%0.1f mm)", col.DisplayName, col.ColWidth, mmWidth)
			tipBg := canvas.NewRectangle(theme.PrimaryColor())
			tipBg.CornerRadius = 3 * s
			tipBg.Resize(fyne.NewSize(float32(len(infoText))*6.5*s+12*s, 18*s))
			tipBg.Move(fyne.NewPos(borderX-30*s, originY-20*s))
			objs = append(objs, tipBg)

			tipLabel := canvas.NewText(infoText, color.White)
			tipLabel.TextStyle = fyne.TextStyle{Bold: true}
			tipLabel.TextSize = scaleFont(8.5, s)
			tipLabel.Move(fyne.NewPos(borderX-25*s, originY-18*s))
			objs = append(objs, tipLabel)
		}

		objs = append(objs, vLine, divGrip)
	}

	// Garis penutup bawah tabel
	bottomLine := canvas.NewLine(color.NRGBA{R: 175, G: 165, B: 140, A: 255})
	bottomLine.Position1 = fyne.NewPos(tableLeftX, originY+tableTotalHeight)
	bottomLine.Position2 = fyne.NewPos(tableRightX, originY+tableTotalHeight)
	bottomLine.StrokeWidth = 1.2
	objs = append(objs, bottomLine)

	r.objects = objs
}

func extractBankCode(name string) string {
	n := name
	if len(n) >= 3 {
		return n[:3]
	}
	return "BNK"
}

func getFormattedFieldValue(req *model.PrintRequest, key string) string {
	switch key {
	case "tanggal":
		return escp.FormatTanggal(req.Tanggal)
	case "kode_transaksi", "sandi":
		return req.KodeTransaksi
	case "keterangan", "uraian":
		return req.Keterangan
	case "debet":
		if req.JenisTransaksi == "tarik" || req.JenisTransaksi == "biaya_admin" {
			return escp.FormatRupiah(req.Jumlah.Float64())
		}
		return "-"
	case "kredit":
		if req.JenisTransaksi == "setor" || req.JenisTransaksi == "bunga" {
			return escp.FormatRupiah(req.Jumlah.Float64())
		}
		return "-"
	case "saldo", "saldo_sesudah":
		return escp.FormatRupiah(req.SaldoSesudah.Float64())
	case "petugas", "validasi":
		return req.Petugas
	default:
		return "-"
	}
}
