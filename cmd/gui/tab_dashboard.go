package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/escp"
	"passbook-bridge/internal/model"
	"passbook-bridge/internal/printer"
)

// readOnlyEntry adalah MultiLineEntry yang hanya bisa dibaca (satu arah: sistem ke user).
// Mencegah user mengetik, menghapus, atau memodifikasi teks log secara tidak sengaja,
// namun tetap mendukung seleksi teks dengan mouse dan copy (Ctrl+C).
type readOnlyEntry struct {
	widget.Entry
}

func newReadOnlyEntry() *readOnlyEntry {
	e := &readOnlyEntry{}
	e.MultiLine = true
	e.TextStyle = fyne.TextStyle{Monospace: true}
	e.Wrapping = fyne.TextTruncate
	e.ExtendBaseWidget(e)
	return e
}

func (e *readOnlyEntry) TypedRune(r rune) {
	// Blokir pengetikan karakter oleh user
}

func (e *readOnlyEntry) TypedKey(key *fyne.KeyEvent) {
	// Izinkan navigasi, tetapi blokir tombol yang memodifikasi teks
	switch key.Name {
	case fyne.KeyBackspace, fyne.KeyDelete, fyne.KeyReturn, fyne.KeyEnter, fyne.KeyTab:
		return
	default:
		e.Entry.TypedKey(key)
	}
}

func (e *readOnlyEntry) TypedShortcut(shortcut fyne.Shortcut) {
	// Blokir paste (Ctrl+V) dan cut (Ctrl+X), izinkan copy (Ctrl+C) dan select all (Ctrl+A)
	if _, ok := shortcut.(*fyne.ShortcutPaste); ok {
		return
	}
	if _, ok := shortcut.(*fyne.ShortcutCut); ok {
		return
	}
	e.Entry.TypedShortcut(shortcut)
}

func buildDashboardTab(win fyne.Window) fyne.CanvasObject {
	statusBadge := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	infoPort := widget.NewLabel("")
	infoDriver := widget.NewLabel("")
	infoTemplate := widget.NewLabel("")
	infoEject := widget.NewLabel("")

	var btnStart *widget.Button
	var btnStop *widget.Button

	refreshDashboard := func() {
		if state.IsRunning {
			statusBadge.SetText(fmt.Sprintf("● Status Server: AKTIF (Listening :%d)", state.Config.Port))
			if btnStart != nil {
				btnStart.Disable()
			}
			if btnStop != nil {
				btnStop.Enable()
			}
		} else {
			statusBadge.SetText("○ Status Server: Nonaktif (Stopped)")
			if btnStart != nil {
				btnStart.Enable()
			}
			if btnStop != nil {
				btnStop.Disable()
			}
		}

		activeTmplName := "Default"
		if state.Layouts != nil && state.Layouts.GetActiveTemplate() != nil {
			activeTmplName = state.Layouts.GetActiveTemplate().Name
		}

		infoPort.SetText(fmt.Sprintf("Port: %d (HTTP)", state.Config.Port))
		infoDriver.SetText(fmt.Sprintf("Driver: %s (%s)", state.Config.PrinterType, state.Config.PrinterName))
		infoTemplate.SetText(fmt.Sprintf("Template: %s", activeTmplName))
		infoEject.SetText(fmt.Sprintf("Auto Eject: %v", state.Config.EnableEject))
	}

	state.OnStateChanged = refreshDashboard

	logView := newReadOnlyEntry()
	logView.SetMinRowsVisible(14)
	logView.SetText(strings.Join(state.LogEntries, "\n"))

	state.OnLogUpdated = func(entry string) {
		logView.SetText(strings.Join(state.LogEntries, "\n"))
		logView.CursorRow = len(state.LogEntries)
	}

	btnStart = widget.NewButtonWithIcon("Mulai Server Bridge", theme.MediaPlayIcon(), func() {
		btnStart.Disable()
		err := state.StartServer()
		if err != nil {
			showError(err, win)
			refreshDashboard()
			return
		}
		refreshDashboard()
	})

	btnStop = widget.NewButtonWithIcon("Hentikan Server", theme.MediaStopIcon(), func() {
		btnStop.Disable()
		err := state.StopServer()
		if err != nil {
			showError(err, win)
		}
		refreshDashboard()
	})

	refreshDashboard()

	btnTestPrint := widget.NewButtonWithIcon("Test Cetak Baris 14", theme.DocumentPrintIcon(), func() {
		sampleReq := &model.PrintRequest{
			Tanggal:        time.Now().Format("02/01/06"),
			KodeTransaksi:  "SET",
			JenisTransaksi: "setor",
			Jumlah:         150000,
			SaldoSesudah:   15600000,
			Petugas:        "TELLER01",
			Keterangan:     "Setoran Uji Coba Baris 14",
		}

		tmpl := state.Layouts.GetActiveTemplate()
		formattedLine := escp.FormatLineWithTemplate(sampleReq, tmpl)

		state.Log("=== PENGUJIAN CETAK BUKU TABUNGAN ===")
		state.Log(fmt.Sprintf("Template: %s", tmpl.Name))
		state.Log(fmt.Sprintf("Hasil Cetak (%d char): |%s|", len(formattedLine), formattedLine))

		var targetPrinter printer.Printer = state.Printer
		var closeTemp bool
		if targetPrinter == nil {
			var err error
			targetPrinter, err = printer.NewPrinter(state.Config)
			if err != nil {
				showError(fmt.Errorf("gagal menghubungkan printer: %w", err), win)
				return
			}
			closeTemp = true
		}

		cmdBytes := escp.GeneratePrintCommand(sampleReq, escp.GeneratorOptions{
			EnableEject: state.Config.EnableEject,
			Template:    tmpl,
		})
		err := targetPrinter.Print(cmdBytes)
		if closeTemp {
			_ = targetPrinter.Close()
		}

		if err != nil {
			state.Log(fmt.Sprintf("Test print gagal: %v", err))
			showError(fmt.Errorf("gagal mengirim ke printer: %w", err), win)
		} else {
			state.Log("Test print sukses dikirim ke printer.")
			showInfo("Test Print Sukses",
				fmt.Sprintf("Perintah cetak berhasil dikirim ke printer (%s).\n\nTeks yang dicetak:\n%s", targetPrinter.Name(), formattedLine), win)
		}
	})

	btnClearLog := widget.NewButtonWithIcon("Bersihkan Log", theme.DeleteIcon(), func() {
		state.LogEntries = make([]string, 0)
		logView.SetText("")
	})

	btnCopyLog := widget.NewButtonWithIcon("Salin Log", theme.ContentCopyIcon(), func() {
		win.Clipboard().SetContent(logView.Text)
		showInfo("Disalin", "Seluruh log berhasil disalin ke clipboard!", win)
	})

	controls := container.NewHBox(btnStart, btnStop, btnTestPrint, widget.NewSeparator(), btnClearLog, btnCopyLog)

	statsGrid := container.NewGridWithColumns(4,
		infoPort,
		infoDriver,
		infoTemplate,
		infoEject,
	)

	topCard := widget.NewCard("Status Layanan Passbook Bridge", "",
		container.NewVBox(
			statusBadge,
			widget.NewSeparator(),
			statsGrid,
			widget.NewSeparator(),
			controls,
		),
	)

	logCard := widget.NewCard("Live Monitor & Activity Log", "Aktivitas koneksi serial, transmisi ESC/P, dan antrean cetak core banking", logView)

	return container.NewBorder(topCard, nil, nil, nil, logCard)
}
