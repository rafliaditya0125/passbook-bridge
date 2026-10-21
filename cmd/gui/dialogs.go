package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ModalDialog merepresentasikan modal dialog terpusat (dead-center) yang aktif
type ModalDialog interface {
	Show()
	Hide()
}

// modalBackdrop adalah kanvas latar belakang semi-transparan yang memblokir
// semua klik mouse ke konten di bawahnya (modal blocker)
type modalBackdrop struct {
	widget.BaseWidget
	rect  *canvas.Rectangle
	onTap func()
}

func newModalBackdrop(onTap func()) *modalBackdrop {
	b := &modalBackdrop{
		rect:  canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 140}),
		onTap: onTap,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *modalBackdrop) Tapped(*fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *modalBackdrop) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.rect)
}

// customModal mengelola overlay modal terpusat
type customModal struct {
	overlay fyne.CanvasObject
	win     fyne.Window
}

func (m *customModal) Show() {
	if m.win != nil && m.win.Canvas() != nil && m.overlay != nil {
		m.win.Canvas().Overlays().Add(m.overlay)
	}
}

func (m *customModal) Hide() {
	if m.win != nil && m.win.Canvas() != nil && m.overlay != nil {
		m.win.Canvas().Overlays().Remove(m.overlay)
		m.overlay = nil
	}
}

// createModalCard membangun dialog box dengan proporsi padding dan visual yang rapi,
// menggunakan layout container.NewCenter yang menjamin secara matematis titik tengah popup (X_center, Y_center)
// selalu 100% tepat di titik tengah layar/jendela (W/2, H/2).
func createModalCard(title string, icon fyne.Resource, content fyne.CanvasObject, buttons []fyne.CanvasObject, minWidth float32, win fyne.Window) ModalDialog {
	m := &customModal{win: win}

	if minWidth <= 0 {
		minWidth = 500
	}

	// 1. Header dengan Icon & Judul yang terpisah rapi
	var titleRow fyne.CanvasObject
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if icon != nil {
		titleRow = container.NewHBox(
			widget.NewIcon(icon),
			titleLabel,
		)
	} else {
		titleRow = titleLabel
	}

	headerContainer := container.NewVBox(
		container.NewPadded(titleRow),
		widget.NewSeparator(),
	)

	// 2. Footer Tombol (ukuran dinamis sesuai teks tombol, tidak terpotong)
	btnBox := container.NewHBox()
	btnBox.Add(layout.NewSpacer())
	for i, b := range buttons {
		if i > 0 {
			btnBox.Add(layout.NewSpacer())
		}
		btnBox.Add(b)
	}
	btnBox.Add(layout.NewSpacer())

	footerContainer := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(container.NewCenter(btnBox)),
	)

	// 3. Body Content dengan ruang padding yang proporsional
	bodyContainer := container.NewPadded(content)

	// 4. Panel Layout terstruktur (Border Layout: Header Top, Footer Bottom, Body Center)
	panelContent := container.NewBorder(
		headerContainer,
		footerContainer,
		nil,
		nil,
		bodyContainer,
	)

	// 5. Background Themed dengan Corner Radius dan Border halus
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	bg := canvas.NewRectangle(th.Color(theme.ColorNameOverlayBackground, v))
	bg.CornerRadius = 12
	bg.StrokeColor = th.Color(theme.ColorNameSeparator, v)
	bg.StrokeWidth = 1

	paddedPanel := container.NewStack(
		bg,
		container.NewPadded(panelContent),
	)

	// Hitung ukuran proporsional
	contentMin := paddedPanel.MinSize()
	finalWidth := fyne.Max(minWidth, contentMin.Width+24)
	finalHeight := contentMin.Height + 16

	cardBox := container.NewGridWrap(fyne.NewSize(finalWidth, finalHeight), paddedPanel)
	centered := container.NewCenter(cardBox)

	m.overlay = container.NewStack(newModalBackdrop(nil), centered)
	return m
}

// CenterDialog dipertahankan untuk backwards compatibility.
// Layout container.NewCenter otomatis memposisikan dialog di titik tengah presisi.
func CenterDialog(win fyne.Window) {
	// No-op
}

// showInfo menampilkan dialog informasi yang otomatis terpusat tepat di tengah layar
func showInfo(title, message string, win fyne.Window) {
	var m ModalDialog
	btnOk := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		m.Hide()
	})
	btnOk.Importance = widget.HighImportance

	msgLabel := widget.NewLabel(message)
	msgLabel.Wrapping = fyne.TextWrapWord

	var body fyne.CanvasObject = msgLabel
	if len(message) > 400 {
		scroll := container.NewVScroll(msgLabel)
		scroll.SetMinSize(fyne.NewSize(480, 220))
		body = scroll
	}

	m = createModalCard(title, theme.InfoIcon(), body, []fyne.CanvasObject{btnOk}, 500, win)
	m.Show()
}

// showError menampilkan dialog error yang otomatis terpusat tepat di tengah layar
func showError(err error, win fyne.Window) {
	if err == nil {
		return
	}
	var m ModalDialog
	btnDismiss := widget.NewButtonWithIcon("Tutup", theme.CancelIcon(), func() {
		m.Hide()
	})

	msgLabel := widget.NewLabel(err.Error())
	msgLabel.Wrapping = fyne.TextWrapWord

	var body fyne.CanvasObject = msgLabel
	if len(err.Error()) > 300 {
		scroll := container.NewVScroll(msgLabel)
		scroll.SetMinSize(fyne.NewSize(480, 200))
		body = scroll
	}

	m = createModalCard("Terjadi Kesalahan", theme.ErrorIcon(), body, []fyne.CanvasObject{btnDismiss}, 500, win)
	m.Show()
}

// showConfirm menampilkan dialog konfirmasi (Lanjutkan / Batal) terpusat tepat di tengah layar
func showConfirm(title, message string, callback func(bool), win fyne.Window) {
	var m ModalDialog
	btnCancel := widget.NewButtonWithIcon("Batal", theme.CancelIcon(), func() {
		m.Hide()
		if callback != nil {
			callback(false)
		}
	})
	btnConfirm := widget.NewButtonWithIcon("Lanjutkan", theme.ConfirmIcon(), func() {
		m.Hide()
		if callback != nil {
			callback(true)
		}
	})
	btnConfirm.Importance = widget.HighImportance

	msgLabel := widget.NewLabel(message)
	msgLabel.Wrapping = fyne.TextWrapWord

	m = createModalCard(title, theme.QuestionIcon(), msgLabel, []fyne.CanvasObject{btnCancel, btnConfirm}, 500, win)
	m.Show()
}

// showCustomConfirm menampilkan modal kustom dengan tombol aksi konfirmasi dan batal
func showCustomConfirm(title, confirmText, dismissText string, content fyne.CanvasObject, callback func(bool), win fyne.Window) ModalDialog {
	var m ModalDialog
	btnCancel := widget.NewButtonWithIcon(dismissText, theme.CancelIcon(), func() {
		m.Hide()
		if callback != nil {
			callback(false)
		}
	})
	btnConfirm := widget.NewButtonWithIcon(confirmText, theme.ConfirmIcon(), func() {
		m.Hide()
		if callback != nil {
			callback(true)
		}
	})
	btnConfirm.Importance = widget.HighImportance

	m = createModalCard(title, nil, content, []fyne.CanvasObject{btnCancel, btnConfirm}, 540, win)
	return m
}

// showCustom menampilkan modal kustom dengan satu tombol tutup
func showCustom(title, dismissText string, content fyne.CanvasObject, win fyne.Window) ModalDialog {
	var m ModalDialog
	btnDismiss := widget.NewButtonWithIcon(dismissText, theme.CancelIcon(), func() {
		m.Hide()
	})

	m = createModalCard(title, nil, content, []fyne.CanvasObject{btnDismiss}, 540, win)
	return m
}
