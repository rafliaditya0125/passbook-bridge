package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/printer"
)

func buildSettingsTab(win fyne.Window) fyne.CanvasObject {
	portEntry := widget.NewEntry()
	portEntry.SetText(strconv.Itoa(state.Config.Port))

	printerTypeSelect := widget.NewSelect([]string{"mock", "spooler", "serial"}, func(s string) {})
	printerTypeSelect.SetSelected(state.Config.PrinterType)

	printerNameEntry := widget.NewEntry()
	printerNameEntry.SetText(state.Config.PrinterName)

	serialPortSelect := widget.NewSelect([]string{"COM1", "COM2", "COM3", "COM4", "/dev/ttyUSB0", "/dev/ttyACM0", "/dev/ttyS0"}, func(s string) {})
	if state.Config.SerialPort != "" {
		serialPortSelect.SetSelected(state.Config.SerialPort)
	}

	baudRateSelect := widget.NewSelect([]string{"9600", "19200", "38400", "115200"}, func(s string) {})
	baudRateSelect.SetSelected(strconv.Itoa(state.Config.BaudRate))

	logLevelSelect := widget.NewSelect([]string{"debug", "info", "warn", "error"}, func(s string) {})
	logLevelSelect.SetSelected(state.Config.LogLevel)

	ejectCheck := widget.NewCheck("Otomatis Keluarkan Buku Tabungan (Form Feed / Eject)", func(b bool) {})
	ejectCheck.SetChecked(state.Config.EnableEject)

	// Fungsi untuk menerapkan perangkat yang dipilih ke form konfigurasi
	applyDeviceToForm := func(dev printer.DeviceInfo) {
		printerTypeSelect.SetSelected(dev.Type)
		if dev.Type == "spooler" {
			printerNameEntry.SetText(dev.Name)
		} else if dev.Type == "serial" {
			serialPortSelect.SetSelected(dev.Port)
			printerNameEntry.SetText(dev.Name)
		} else {
			printerNameEntry.SetText(dev.Name)
		}
		state.Log(fmt.Sprintf("Perangkat printer dipilih: %s (%s, Port: %s)", dev.Name, dev.Type, dev.Port))
	}

	// Dialog interaktif pemilihan perangkat jika terdeteksi lebih dari 1 printer
	showDeviceSelectionDialog := func(devices []printer.DeviceInfo) {
		var d ModalDialog

		listContainer := container.NewVBox()

		for _, dev := range devices {
			dInfo := dev

			title := dInfo.Name
			if dInfo.IsPassbook {
				title = "⭐ " + dInfo.Name + " (Rekomendasi Passbook)"
			}

			lblTitle := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			lblDesc := widget.NewLabel(fmt.Sprintf("Tipe Driver: %s | Port: %s\n%s", strings.ToUpper(dInfo.Type), dInfo.Port, dInfo.Description))

			btnPick := widget.NewButtonWithIcon("Pilih Printer Ini", theme.ConfirmIcon(), func() {
				if d != nil {
					d.Hide()
				}
				applyDeviceToForm(dInfo)
				showInfo("Perangkat Dipilih",
					fmt.Sprintf("Perangkat '%s' berhasil dipilih dan diterapkan ke form pengaturan!\n\nSilakan klik 'Simpan Konfigurasi' untuk menyimpan.", dInfo.Name), win)
			})
			if dInfo.IsPassbook {
				btnPick.Importance = widget.HighImportance
			}

			cardRow := widget.NewCard("", "",
				container.NewBorder(
					nil, nil, nil, btnPick,
					container.NewVBox(lblTitle, lblDesc),
				),
			)

			listContainer.Add(cardRow)
		}

		scrollContent := container.NewVScroll(listContainer)
		scrollContent.SetMinSize(fyne.NewSize(620, 320))

		headerNote := widget.NewLabelWithStyle(
			fmt.Sprintf("Ditemukan %d perangkat printer/port pada sistem. Silakan tentukan printer yang ingin Anda gunakan:", len(devices)),
			fyne.TextAlignLeading, fyne.TextStyle{Italic: true},
		)

		dialogContent := container.NewBorder(
			headerNote,
			nil, nil, nil,
			scrollContent,
		)

		d = showCustom("Pilih Perangkat Passbook Printer", "Batal", container.NewGridWrap(fyne.NewSize(640, 380), dialogContent), win)
		d.Show()
	}

	// Tombol Pindai Perangkat (Scan Devices)
	btnScanDevices := widget.NewButtonWithIcon("Pindai Perangkat (Scan Device)", theme.SearchIcon(), func() {
		devices := printer.ScanAllDevices()

		// Perbarui juga opsi dropdown serial port jika ada port serial baru terdeteksi
		var serialPorts []string
		for _, dev := range devices {
			if dev.Type == "serial" && dev.Port != "" {
				serialPorts = append(serialPorts, dev.Port)
			}
		}
		if len(serialPorts) > 0 {
			serialPortSelect.Options = serialPorts
			serialPortSelect.Refresh()
		}

		if len(devices) == 0 {
			dMock := showCustomConfirm("Hasil Pindai Perangkat", "Gunakan Mode Mock", "Tutup",
				container.NewVBox(
					widget.NewLabel("Tidak ada printer fisik (Spooler) atau port serial (COM) yang terdeteksi saat ini."),
					widget.NewLabel("Pastikan:"),
					widget.NewLabel("1. Kabel data USB / Serial printer passbook terpasang dengan baik."),
					widget.NewLabel("2. Printer dalam kondisi menyala (Power ON)."),
					widget.NewLabel("3. Driver printer sudah terpasang di Windows Devices & Printers."),
					widget.NewSeparator(),
					widget.NewLabel("Ingin beralih ke Mode Mock (Simulasi Virtual) untuk pengujian tanpa printer fisik?"),
				), func(useMock bool) {
					if useMock {
						printerTypeSelect.SetSelected("mock")
						printerNameEntry.SetText("Simulator Virtual Passbook")
					}
				}, win)
			dMock.Show()
			dMock.Show()
			CenterDialog(win)
			return
		}

		if len(devices) == 1 {
			dev := devices[0]
			applyDeviceToForm(dev)
			showInfo("1 Perangkat Terdeteksi",
				fmt.Sprintf("Terdeteksi 1 perangkat printer:\n\nNama: %s\nTipe: %s (Port: %s)\n\nPerangkat telah otomatis dipilih pada form konfigurasi!", dev.Name, dev.Type, dev.Port), win)
			return
		}

		// Jika terdeteksi >= 2 perangkat, tampilkan dialog interaktif agar user bisa memilih
		showDeviceSelectionDialog(devices)
	})
	btnScanDevices.Importance = widget.HighImportance

	scanCard := widget.NewCard("Deteksi Otomatis Perangkat Printer (Auto-Scan)", "",
		container.NewVBox(
			widget.NewLabelWithStyle("Sistem akan melacak printer passbook fisik (Epson PLQ-20/30, Olivetti PR2+, Compuprint SP40) serta port serial COM aktif yang terhubung ke komputer.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			container.NewHBox(btnScanDevices),
		),
	)

	form := widget.NewForm(
		widget.NewFormItem("Port HTTP Server", portEntry),
		widget.NewFormItem("Tipe Driver Printer", printerTypeSelect),
		widget.NewFormItem("Nama Printer (Windows Spooler)", printerNameEntry),
		widget.NewFormItem("Port Serial (COM/TTY)", serialPortSelect),
		widget.NewFormItem("Serial Baud Rate", baudRateSelect),
		widget.NewFormItem("Log Verbosity", logLevelSelect),
		widget.NewFormItem("Fitur Eject", ejectCheck),
	)

	btnTestConn := widget.NewButtonWithIcon("Uji Koneksi Printer", theme.MediaPlayIcon(), func() {
		tempCfg := *state.Config
		tempCfg.PrinterType = printerTypeSelect.Selected
		tempCfg.PrinterName = printerNameEntry.Text
		tempCfg.SerialPort = serialPortSelect.Selected

		p, err := printer.NewPrinter(&tempCfg)
		if err != nil {
			showError(fmt.Errorf("koneksi printer gagal: %w", err), win)
			return
		}
		_ = p.Close()
		showInfo("Koneksi Berhasil", fmt.Sprintf("Perangkat printer '%s' (%s) berhasil terhubung dan siap!", p.Name(), tempCfg.PrinterType), win)
	})

	btnSave := widget.NewButtonWithIcon("Simpan Konfigurasi Printer & Server", theme.DocumentSaveIcon(), func() {
		port, err := strconv.Atoi(portEntry.Text)
		if err != nil || port <= 0 {
			showError(fmt.Errorf("port harus berupa angka valid"), win)
			return
		}

		baud, _ := strconv.Atoi(baudRateSelect.Selected)
		if baud <= 0 {
			baud = 9600
		}

		state.Config.Port = port
		state.Config.PrinterType = printerTypeSelect.Selected
		state.Config.PrinterName = printerNameEntry.Text
		state.Config.SerialPort = serialPortSelect.Selected
		state.Config.BaudRate = baud
		state.Config.LogLevel = logLevelSelect.Selected
		state.Config.EnableEject = ejectCheck.Checked

		err = config.SaveConfig(state.ConfigPath, state.Config)
		if err != nil {
			showError(fmt.Errorf("gagal menyimpan konfigurasi: %w", err), win)
			return
		}

		state.Log("Konfigurasi server & printer berhasil diperbarui di " + state.ConfigPath)
		state.NotifyStateChanged()
		showInfo("Berhasil", "Konfigurasi server & printer berhasil disimpan!", win)
	})
	btnSave.Importance = widget.HighImportance

	buttonsRow := container.NewHBox(btnSave, btnTestConn)

	cardConfig := widget.NewCard("Pengaturan Koneksi & Hardware", "Konfigurasi port komunikasi serial dot-matrix dan parameter server",
		container.NewVBox(
			form,
			widget.NewSeparator(),
			buttonsRow,
		),
	)

	return container.NewScroll(
		container.NewVBox(
			scanCard,
			widget.NewSeparator(),
			cardConfig,
		),
	)
}
