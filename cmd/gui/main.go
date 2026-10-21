package main

import (
	"flag"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"passbook-bridge/internal/config"
)

func main() {
	configPath := flag.String("config", "config.json", "Path file konfigurasi server")
	layoutsPath := flag.String("layouts", "layouts.json", "Path file layout buku tabungan")
	flag.Parse()

	state.ConfigPath = *configPath
	state.LayoutsPath = *layoutsPath

	// 1. Muat Config Server
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error memuat config: %v\n", err)
		cfg = config.DefaultConfig()
	}
	state.Config = cfg

	// 2. Muat Layout Buku Tabungan
	layoutCfg, err := config.LoadLayoutConfig(*layoutsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error memuat layout: %v\n", err)
		layoutCfg = config.DefaultLayoutConfig()
	}
	state.Layouts = layoutCfg

	// 3. Inisialisasi Fyne GUI Application
	myApp := app.NewWithID("com.passbook.bridge")
	myWindow := myApp.NewWindow("Passbook Bridge - Controller & Layout Designer")
	state.Window = myWindow

	// 4. Otomatis nyalakan server bridge jika belum aktif
	_ = state.StartServer()

	// 5. Bangun 4 Tab Utama Native Desktop Fyne
	tabDesigner := container.NewTabItemWithIcon("Layout Designer", theme.DocumentCreateIcon(), buildDesignerTab(myWindow))
	tabDashboard := container.NewTabItemWithIcon("Dashboard & Monitor", theme.HomeIcon(), buildDashboardTab(myWindow))
	tabSettings := container.NewTabItemWithIcon("Pengaturan Hardware", theme.SettingsIcon(), buildSettingsTab(myWindow))
	tabService := container.NewTabItemWithIcon("Service Manager", theme.ComputerIcon(), buildServiceTab(myWindow))

	tabs := container.NewAppTabs(tabDesigner, tabDashboard, tabSettings, tabService)
	tabs.SetTabLocation(container.TabLocationTop)

	myWindow.SetContent(tabs)
	myWindow.Resize(fyne.NewSize(1240, 780))
	myWindow.CenterOnScreen()

	// 6. System Tray Menu
	if desk, ok := myApp.(desktop.App); ok {
		menu := fyne.NewMenu("Passbook Bridge",
			fyne.NewMenuItem("Buka Controller GUI", func() {
				myWindow.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Mulai Server", func() {
				_ = state.StartServer()
			}),
			fyne.NewMenuItem("Hentikan Server", func() {
				_ = state.StopServer()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Keluar", func() {
				_ = state.StopServer()
				myApp.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
	}

	state.Log("Aplikasi Passbook Bridge Desktop Controller & Layout Designer siap digunakan.")
	myWindow.ShowAndRun()
}
