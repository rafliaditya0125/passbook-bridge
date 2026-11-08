package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"passbook-bridge/internal/config"
)

func TestMakeColumnsSticky(t *testing.T) {
	tmpl := &config.LayoutTemplate{
		ID:         "test",
		Name:       "Test Template",
		MaxColumns: 80,
		Columns: []config.ColumnDef{
			{ID: "c1", DisplayName: "Col 1", ColStart: 0, ColWidth: 10},
			{ID: "c2", DisplayName: "Col 2", ColStart: 99, ColWidth: 15}, // Gapped
			{ID: "c3", DisplayName: "Col 3", ColStart: 5, ColWidth: 20},  // Overlapped
		},
	}

	MakeColumnsSticky(tmpl)

	if tmpl.Columns[0].ColStart != 0 {
		t.Errorf("Expected Col 1 ColStart 0, got %d", tmpl.Columns[0].ColStart)
	}
	if tmpl.Columns[1].ColStart != 10 {
		t.Errorf("Expected Col 2 ColStart 10, got %d", tmpl.Columns[1].ColStart)
	}
	if tmpl.Columns[2].ColStart != 25 {
		t.Errorf("Expected Col 3 ColStart 25 (10+15), got %d", tmpl.Columns[2].ColStart)
	}

	// Now simulate resizing Col 1 to width 14
	tmpl.Columns[0].ColWidth = 14
	MakeColumnsSticky(tmpl)

	if tmpl.Columns[1].ColStart != 14 {
		t.Errorf("Sticky failed: Col 2 should have shifted to 14, got %d", tmpl.Columns[1].ColStart)
	}
	if tmpl.Columns[2].ColStart != 29 {
		t.Errorf("Sticky failed: Col 3 should have shifted to 29 (14+15), got %d", tmpl.Columns[2].ColStart)
	}
}

func TestPassbookCanvasWidget_ZoomAndPan(t *testing.T) {
	_ = test.NewApp()

	tmpl := &config.LayoutTemplate{
		ID:         "test_zoom",
		Name:       "Test Zoom",
		MaxColumns: 80,
		Columns: []config.ColumnDef{
			{ID: "c1", DisplayName: "Col 1", ColStart: 0, ColWidth: 10},
		},
	}

	w := NewPassbookCanvasWidget(tmpl, nil, nil)
	if w.GetScale() != 1.0 {
		t.Errorf("Expected initial scale 1.0, got %f", w.GetScale())
	}

	// Zoom In
	w.ZoomIn()
	if w.GetScale() < 1.14 || w.GetScale() > 1.16 {
		t.Errorf("Expected scale ~1.15, got %f", w.GetScale())
	}

	// Zoom Out
	w.ZoomOut()
	if w.GetScale() < 0.99 || w.GetScale() > 1.01 {
		t.Errorf("Expected scale ~1.0, got %f", w.GetScale())
	}

	// Reset Zoom
	w.SetScale(1.8)
	w.panX = 50
	w.panY = 30
	w.ResetZoom()
	if w.GetScale() != 1.0 || w.panX != 0 || w.panY != 0 {
		t.Errorf("ResetZoom failed, got scale=%f panX=%f panY=%f", w.GetScale(), w.panX, w.panY)
	}
}

func TestPassbookCanvasWidget_DragInteractions(t *testing.T) {
	_ = test.NewApp()

	tmpl := &config.LayoutTemplate{
		ID:         "test_drag",
		Name:       "Test Drag",
		MaxColumns: 80,
		LeftMargin: 2,
		TopMargin:  3,
		Columns: []config.ColumnDef{
			{ID: "c1", DisplayName: "Col 1", ColStart: 0, ColWidth: 10},
			{ID: "c2", DisplayName: "Col 2", ColStart: 10, ColWidth: 15},
		},
	}

	var changed bool
	w := NewPassbookCanvasWidget(tmpl, nil, func() {
		changed = true
	})

	// Test case 2: Sticky Excel resize
	w.dragMode = 2
	w.dragBorderIdx = 1 // Border 1 = right border of Col 1
	w.dragStartX = 100
	w.initialColWidths = []int{10, 15}

	// Simulate dragging 5 columns to the right (assuming charW = 10px, so +50px)
	// deltaCols = round(50 / 8.7) ~ 5 or 6
	_, _, _, _, _, _, charW, _ := w.getGeometry()
	movePx := charW * 4 // +4 columns

	ev := &fyne.DragEvent{
		PointEvent: fyne.PointEvent{
			Position: fyne.NewPos(100+movePx, 200),
		},
	}
	w.Dragged(ev)

	if !changed {
		t.Errorf("Expected OnChanged to be called during drag resize")
	}
	if tmpl.Columns[0].ColWidth != 14 {
		t.Errorf("Expected Col 1 width 14, got %d", tmpl.Columns[0].ColWidth)
	}
	if tmpl.Columns[1].ColStart != 14 {
		t.Errorf("Expected Col 2 ColStart to be sticky at 14, got %d", tmpl.Columns[1].ColStart)
	}

	w.DragEnd()
	if w.dragMode != 0 {
		t.Errorf("Expected dragMode 0 after DragEnd, got %d", w.dragMode)
	}
}

func TestPassbookCanvasWidget_ExcelFeatures(t *testing.T) {
	_ = test.NewApp()

	// 1. Test Excel column letters
	if getExcelColLetter(0) != "A" {
		t.Errorf("Expected 0 -> A, got %s", getExcelColLetter(0))
	}
	if getExcelColLetter(1) != "B" {
		t.Errorf("Expected 1 -> B, got %s", getExcelColLetter(1))
	}
	if getExcelColLetter(6) != "G" {
		t.Errorf("Expected 6 -> G, got %s", getExcelColLetter(6))
	}

	// 2. Test Indonesian currency formatting
	if formatIDRCurrency(0) != "0,00" {
		t.Errorf("Expected 0,00, got %s", formatIDRCurrency(0))
	}
	if formatIDRCurrency(5000000) != "5.000.000,00" {
		t.Errorf("Expected 5.000.000,00, got %s", formatIDRCurrency(5000000))
	}
	if formatIDRCurrency(100000) != "100.000,00" {
		t.Errorf("Expected 100.000,00, got %s", formatIDRCurrency(100000))
	}

	// 3. Test Widget Target Row & Quick Adjust
	tmpl := &config.LayoutTemplate{
		ID:         "test_excel",
		Name:       "Test Excel",
		MaxColumns: 80,
		Columns: []config.ColumnDef{
			{ID: "c1", DisplayName: "No", ColStart: 0, ColWidth: 4},
			{ID: "c2", DisplayName: "Tanggal", ColStart: 4, ColWidth: 11},
		},
	}

	var rowNotified int
	w := NewPassbookCanvasWidget(tmpl, nil, nil)
	w.OnTargetRowChanged = func(row int) {
		rowNotified = row
	}

	w.SetTargetRow(14)
	if w.GetTargetRow() != 14 || rowNotified != 14 {
		t.Errorf("Expected targetRow 14, got %d (notified: %d)", w.GetTargetRow(), rowNotified)
	}

	// 4. Test Quick Adjust Column Lebar & Tab X
	w.SetSelectedColumn(1) // Select Tanggal
	w.AdjustSelectedWidth(2)
	if tmpl.Columns[1].ColWidth != 13 {
		t.Errorf("Expected ColWidth 13, got %d", tmpl.Columns[1].ColWidth)
	}

	w.AdjustSelectedStart(1)
	if tmpl.Columns[1].ColStart != 5 {
		t.Errorf("Expected ColStart 5, got %d", tmpl.Columns[1].ColStart)
	}

	// 5. Test Sticky Mode
	if !w.IsStickyMode() {
		t.Errorf("Expected stickyMode true by default")
	}
	w.SetStickyMode(false)
	if w.IsStickyMode() {
		t.Errorf("Expected stickyMode false after toggle")
	}
}

func TestPassbookCanvasWidget_BookSizeAndDimensions(t *testing.T) {
	_ = test.NewApp()

	tmpl := &config.LayoutTemplate{
		ID:           "test_size",
		Name:         "Test Size",
		MaxColumns:   80,
		TotalRows:    34,
		PageWidthMM:  148.0,
		PageHeightMM: 105.0,
		Columns: []config.ColumnDef{
			{ID: "c1", DisplayName: "No", ColStart: 0, ColWidth: 4},
		},
	}

	w := NewPassbookCanvasWidget(tmpl, nil, nil)
	_, _, paperW1, paperH1, _, _, _, _ := w.getGeometry()

	// Sekarang ubah ke preset yang lebih besar (Buku Besar A5: 210 x 148 mm)
	tmpl.PageWidthMM = 210.0
	tmpl.PageHeightMM = 148.0
	tmpl.TotalRows = 40
	w.SetTemplate(tmpl)

	_, _, paperW2, paperH2, _, _, _, _ := w.getGeometry()

	if paperW2 <= paperW1 {
		t.Errorf("Ukuran kertas tidak bertambah: paperW1=%f, paperW2=%f", paperW1, paperW2)
	}
	if paperH2 <= paperH1 {
		t.Errorf("Tinggi kertas tidak bertambah: paperH1=%f, paperH2=%f", paperH1, paperH2)
	}
	if w.GetDisplayRows() != 34 { // capped at 34 display rows
		t.Errorf("Expected display rows 34, got %d", w.GetDisplayRows())
	}
}
