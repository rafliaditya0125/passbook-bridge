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
