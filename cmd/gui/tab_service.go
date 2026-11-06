package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/service"
)

// executeWithPasswordPrompt menjalankan aksi service dengan meminta password administrator langsung di GUI Fyne jika dibutuhkan
func executeWithPasswordPrompt(
	win fyne.Window,
	actionName string,
	action func(password string) (string, error),
	onSuccess func(output string),
) {
	// Pada Windows: otorisasi administrator ditangani langsung oleh sistem operasi via dialog UAC bawaan (Yes/No),
	// sehingga pengguna TIDAK PERNAH diminta mengetikkan password di dalam GUI aplikasi (cocok untuk akun administrator tanpa password).
	if runtime.GOOS == "windows" {
		out, err := action("")
		if err != nil {
			showError(err, win)
		} else {
			onSuccess(out)
		}
		return
	}

	// Jika Linux, cek dulu apakah sudo tanpa password (sudo -n) sudah aktif
	if runtime.GOOS == "linux" {
		testCmd := exec.Command("sudo", "-n", "true")
		if err := testCmd.Run(); err == nil {
			out, err := action("")
			if err != nil {
				showError(err, win)
			} else {
				onSuccess(out)
			}
			return
		}
	}

	// Tampilkan modal password prompt di dalam GUI Fyne (Dead Center) - KHUSUS LINUX jika butuh sudo
	passEntry := widget.NewPasswordEntry()
	passEntry.SetPlaceHolder("Masukkan password root / sudo (kosongkan jika tanpa password)")

	infoLabel := widget.NewLabel(
		fmt.Sprintf("Tindakan '%s' membutuhkan izin root / sudo pada sistem Linux.\nSilakan masukkan password akun Anda untuk melanjutkan:", actionName),
	)

	var d ModalDialog

	handleExecute := func() {
		pwd := passEntry.Text
		if d != nil {
			d.Hide()
		}

		out, err := action(pwd)
		if err != nil {
			showError(fmt.Errorf("tindakan gagal: %w\n\nDetail output:\n%s", err, out), win)
		} else {
			onSuccess(out)
		}
	}

	passEntry.OnSubmitted = func(string) {
		handleExecute()
	}

	dialogBody := container.NewVBox(
		infoLabel,
		passEntry,
	)

	d = showCustomConfirm(
		"Otorisasi Root / Sudo",
		"Lanjutkan",
		"Batal",
		dialogBody,
		func(confirmed bool) {
			if confirmed {
				handleExecute()
			}
		},
		win,
	)
	d.Show()
	win.Canvas().Focus(passEntry)
}

func buildServiceTab(win fyne.Window) fyne.CanvasObject {
	statusTitle := widget.NewLabelWithStyle("Status Background Service:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusValue := widget.NewLabelWithStyle("Memeriksa status...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusDetail := widget.NewLabel("")

	var currentStatus service.ServiceStatus
	var btnToggle *widget.Button
	var btnUninstall *widget.Button

	refreshStatus := func() {
		st, msg := service.CheckStatus()
		currentStatus = st
		statusDetail.SetText(msg)

		if btnToggle == nil {
			return
		}

		switch st {
		case service.StatusRunning:
			statusValue.SetText("● STATUS SERVICE: AKTIF (ON)")
			btnToggle.SetText("Nonaktifkan Service (Turn OFF)")
			btnToggle.SetIcon(theme.MediaStopIcon())
			btnToggle.Importance = widget.DangerImportance
			btnToggle.Enable()
			if btnUninstall != nil {
				btnUninstall.Enable()
			}
		case service.StatusStopped:
			statusValue.SetText("○ STATUS SERVICE: NONAKTIF (OFF)")
			btnToggle.SetText("Aktifkan Service (Turn ON)")
			btnToggle.SetIcon(theme.MediaPlayIcon())
			btnToggle.Importance = widget.HighImportance
			btnToggle.Enable()
			if btnUninstall != nil {
				btnUninstall.Enable()
			}
		case service.StatusNotInstalled:
			statusValue.SetText("⚠ STATUS SERVICE: BELUM TERPASANG (NOT INSTALLED)")
			btnToggle.SetText("Pasang & Aktifkan Service (Turn ON)")
			btnToggle.SetIcon(theme.ContentAddIcon())
			btnToggle.Importance = widget.HighImportance
			btnToggle.Enable()
			if btnUninstall != nil {
				btnUninstall.Disable()
			}
		default:
			statusValue.SetText(string(st))
			btnToggle.SetText("Periksa / Refresh")
			btnToggle.SetIcon(theme.ViewRefreshIcon())
			btnToggle.Importance = widget.MediumImportance
		}
		btnToggle.Refresh()
		statusValue.Refresh()
	}

	btnRefresh := widget.NewButtonWithIcon("Refresh Status", theme.ViewRefreshIcon(), func() {
		refreshStatus()
	})

	// Tombol Smart Toggle ON/OFF
	btnToggle = widget.NewButtonWithIcon("Aktifkan Service (Turn ON)", theme.MediaPlayIcon(), func() {
		switch currentStatus {
		case service.StatusRunning:
			// Sedang aktif -> Matikan (Turn OFF)
			executeWithPasswordPrompt(win, "Hentikan Service", func(pwd string) (string, error) {
				return service.StopService(pwd)
			}, func(out string) {
				showInfo("Service Dinonaktifkan", "Background service berhasil dihentikan (OFF).", win)
				refreshStatus()
			})

		case service.StatusStopped:
			// Sedang mati -> Nyalakan (Turn ON)
			executeWithPasswordPrompt(win, "Aktifkan Service", func(pwd string) (string, error) {
				return service.StartService(pwd)
			}, func(out string) {
				showInfo("Service Diaktifkan", "Background service berhasil dinyalakan (ON)!", win)
				refreshStatus()
			})

		case service.StatusNotInstalled:
			// Belum terpasang -> Pasang & Otomatis Nyalakan (Install & Turn ON)
			exePath, err := os.Executable()
			if err != nil {
				exePath = filepath.Join(".", "passbook-bridge")
			}
			executeWithPasswordPrompt(win, "Pasang & Aktifkan Service", func(pwd string) (string, error) {
				return service.InstallService(exePath, pwd)
			}, func(out string) {
				showInfo("Service Berhasil Dipasang",
					fmt.Sprintf("Service berhasil dipasang dan otomatis berjalan di background!\n\nDetail:\n%s", out), win)
				refreshStatus()
			})
		}
	})
	btnToggle.Importance = widget.HighImportance

	btnUninstall = widget.NewButtonWithIcon("Hapus Service (Uninstall)", theme.DeleteIcon(), func() {
		showConfirm("Konfirmasi Hapus Service", "Apakah Anda yakin ingin mencopot background service Passbook Bridge?", func(ok bool) {
			if !ok {
				return
			}
			executeWithPasswordPrompt(win, "Hapus Service", func(pwd string) (string, error) {
				return service.UninstallService(pwd)
			}, func(out string) {
				showInfo("Uninstall Service", fmt.Sprintf("Service berhasil dicopot dari sistem:\n%s", out), win)
				refreshStatus()
			})
		}, win)
	})

	// Inisialisasi status awal
	refreshStatus()

	osInfo := fmt.Sprintf("Sistem Operasi Terdeteksi: %s/%s", runtime.GOOS, runtime.GOARCH)
	infoText := widget.NewLabel(
		"Panduan Manajemen Background Service (ON / OFF):\n\n" +
			"1. Tombol ON / OFF secara cerdas menyesuaikan status:\n" +
			"   - Jika service mati (OFF), tombol akan menyalakan service (Turn ON).\n" +
			"   - Jika service aktif (ON), tombol akan menghentikan service (Turn OFF).\n" +
			"   - Jika belum terpasang, tombol otomatis membuat file service dan langsung menyalakannya.\n" +
			"2. Pada Linux: Mengatur unit systemd (/etc/systemd/system/passbook-bridge.service).\n" +
			"3. Pada Windows: Mengatur Windows Service Controller (PassbookBridge via sc.exe).\n" +
			"4. Pengelolaan izin root / administrator:\n" +
			"   - Pada Windows: Konfirmasi administrator dilakukan otomatis melalui dialog UAC bawaan (Yes/No), tanpa meminta ketik password.\n" +
			"   - Pada Linux: Dialog otorisasi root/sudo akan muncul di dalam aplikasi jika sistem membutuhkannya.",
	)

	// Tata letak tombol utama: Tombol Toggle ON/OFF berukuran besar, Refresh, dan tombol Uninstall
	buttonRow := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(300, 42), btnToggle),
		btnRefresh,
		widget.NewSeparator(),
		btnUninstall,
	)

	card := widget.NewCard("Manajemen Background Service (Linux Systemd / Windows Service)", osInfo,
		container.NewVBox(
			statusTitle,
			statusValue,
			statusDetail,
			widget.NewSeparator(),
			buttonRow,
			widget.NewSeparator(),
			infoText,
		),
	)

	return container.NewScroll(card)
}
