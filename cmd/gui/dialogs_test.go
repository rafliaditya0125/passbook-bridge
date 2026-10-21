package main

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func getModalCardBox(win fyne.Window) (fyne.CanvasObject, fyne.Size) {
	top := win.Canvas().Overlays().Top()
	if top == nil {
		return nil, fyne.Size{}
	}
	stack, ok := top.(*fyne.Container)
	if !ok || len(stack.Objects) < 2 {
		return nil, fyne.Size{}
	}
	centerCont, ok := stack.Objects[1].(*fyne.Container)
	if !ok || len(centerCont.Objects) == 0 {
		return nil, fyne.Size{}
	}
	box := centerCont.Objects[0]
	return box, win.Canvas().Size()
}

func TestModalCentering_Info(t *testing.T) {
	app := test.NewApp()
	w := app.NewWindow("Test Info")
	w.Resize(fyne.NewSize(1600, 855))

	showInfo("Test Title", "Test Message Info", w)

	box, winSize := getModalCardBox(w)
	if box == nil {
		t.Fatalf("Expected modal card to be present in overlay")
	}

	centerX := box.Position().X + box.Size().Width/2
	centerY := box.Position().Y + box.Size().Height/2

	expectedX := winSize.Width / 2
	expectedY := winSize.Height / 2

	if centerX != expectedX {
		t.Errorf("Expected info modal center X=%.2f, got %.2f", expectedX, centerX)
	}
	if centerY != expectedY {
		t.Errorf("Expected info modal center Y=%.2f, got %.2f", expectedY, centerY)
	}
}

func TestModalCentering_Error(t *testing.T) {
	app := test.NewApp()
	w := app.NewWindow("Test Error")
	w.Resize(fyne.NewSize(1200, 800))

	showError(errors.New("gagal menghubungkan ke printer"), w)

	box, winSize := getModalCardBox(w)
	if box == nil {
		t.Fatalf("Expected modal card to be present in overlay")
	}

	centerX := box.Position().X + box.Size().Width/2
	centerY := box.Position().Y + box.Size().Height/2

	expectedX := winSize.Width / 2
	expectedY := winSize.Height / 2

	if centerX != expectedX {
		t.Errorf("Expected error modal center X=%.2f, got %.2f", expectedX, centerX)
	}
	if centerY != expectedY {
		t.Errorf("Expected error modal center Y=%.2f, got %.2f", expectedY, centerY)
	}
}

func TestModalCentering_Confirm(t *testing.T) {
	app := test.NewApp()
	w := app.NewWindow("Test Confirm")
	w.Resize(fyne.NewSize(1024, 768))

	var confirmed bool
	showConfirm("Hapus Item", "Apakah Anda yakin ingin menghapus?", func(res bool) {
		confirmed = res
	}, w)

	box, winSize := getModalCardBox(w)
	if box == nil {
		t.Fatalf("Expected modal card to be present in overlay")
	}

	centerX := box.Position().X + box.Size().Width/2
	centerY := box.Position().Y + box.Size().Height/2

	expectedX := winSize.Width / 2
	expectedY := winSize.Height / 2

	if centerX != expectedX {
		t.Errorf("Expected confirm modal center X=%.2f, got %.2f", expectedX, centerX)
	}
	if centerY != expectedY {
		t.Errorf("Expected confirm modal center Y=%.2f, got %.2f", expectedY, centerY)
	}
	_ = confirmed
}

func TestModalCentering_CustomConfirm(t *testing.T) {
	app := test.NewApp()
	w := app.NewWindow("Test Custom")
	w.Resize(fyne.NewSize(1450, 775))

	content := container.NewGridWrap(fyne.NewSize(500, 100), widget.NewLabel("Custom Form"))
	d := showCustomConfirm("Sudo Prompt", "Lanjutkan", "Batal", content, func(bool) {}, w)
	d.Show()

	box, winSize := getModalCardBox(w)
	if box == nil {
		t.Fatalf("Expected custom modal card to be present in overlay")
	}

	centerX := box.Position().X + box.Size().Width/2
	centerY := box.Position().Y + box.Size().Height/2

	expectedX := winSize.Width / 2
	expectedY := winSize.Height / 2

	if centerX != expectedX {
		t.Errorf("Expected custom modal center X=%.2f, got %.2f", expectedX, centerX)
	}
	if centerY != expectedY {
		t.Errorf("Expected custom modal center Y=%.2f, got %.2f", expectedY, centerY)
	}

	// Test Hide()
	d.Hide()
	if w.Canvas().Overlays().Top() != nil {
		t.Errorf("Expected overlay to be removed after Hide()")
	}
}
