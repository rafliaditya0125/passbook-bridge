package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/escp"
	"passbook-bridge/internal/model"
	"passbook-bridge/internal/printer"
)

func buildDesignerTab(win fyne.Window) fyne.CanvasObject {
	activeTmpl := state.Layouts.GetActiveTemplate()
	if activeTmpl == nil {
		if len(state.Layouts.Templates) > 0 {
			activeTmpl = &state.Layouts.Templates[0]
			state.Layouts.ActiveTemplate = activeTmpl.ID
		} else {
			def := config.DefaultLayoutConfig()
			state.Layouts = def
			activeTmpl = &def.Templates[0]
		}
	}

	selectedColIdx := 0
	isUpdatingUI := false

	// --- 1. State & Sample Data ---
	charUsageLabel := widget.NewLabelWithStyle("0 / 80 Kolom", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})

	sampleReq := &model.PrintRequest{
		Tanggal:        time.Now().Format("02/01/06"),
		KodeTransaksi:  "SET",
		JenisTransaksi: "setor",
		Jumlah:         150000,
		SaldoSesudah:   15600000,
		Petugas:        "TELLER01",
		Keterangan:     "Setoran Uji Coba Baris 14",
	}

	// --- 2. Inspector Controls ---
	topMarginLabel := widget.NewLabel("0 baris")
	topMarginSlider := widget.NewSlider(0, 18)
	topMarginSlider.Step = 1

	leftMarginLabel := widget.NewLabel("0 kolom")
	leftMarginSlider := widget.NewSlider(0, 30)
	leftMarginSlider.Step = 1

	colNameEntry := widget.NewEntry()
	colKeySelect := widget.NewSelect([]string{
		"tanggal", "kode_transaksi", "keterangan", "debet", "kredit", "saldo_sesudah", "petugas",
	}, nil)

	colAlignSelect := widget.NewSelect([]string{"left", "center", "right"}, nil)

	colStartLabel := widget.NewLabel("0")
	colStartSlider := widget.NewSlider(0, 78)
	colStartSlider.Step = 1

	colWidthLabel := widget.NewLabel("10")
	colWidthSlider := widget.NewSlider(2, 45)
	colWidthSlider.Step = 1

	var btnMoveLeft *widget.Button
	var btnMoveRight *widget.Button
	var btnDeleteCol *widget.Button
	var btnAddCol *widget.Button

	var canvasWidget *PassbookCanvasWidget

	// --- 3. Core Synchronization Functions ---
	updateLivePreview := func() {
		if activeTmpl == nil {
			return
		}
		maxEnd := 0
		leftM := activeTmpl.LeftMargin
		for _, col := range activeTmpl.Columns {
			end := col.ColStart + leftM + col.ColWidth
			if end > maxEnd {
				maxEnd = end
			}
		}
		charUsageLabel.SetText(fmt.Sprintf("%d / %d Kolom", maxEnd, activeTmpl.MaxColumns))
	}

	updateInspector := func() {
		isUpdatingUI = true
		defer func() { isUpdatingUI = false }()

		if activeTmpl == nil {
			return
		}

		topMarginSlider.SetValue(float64(activeTmpl.TopMargin))
		topMarginLabel.SetText(fmt.Sprintf("%d baris", activeTmpl.TopMargin))

		leftMarginSlider.SetValue(float64(activeTmpl.LeftMargin))
		leftMarginLabel.SetText(fmt.Sprintf("%d kolom", activeTmpl.LeftMargin))

		if selectedColIdx < 0 || selectedColIdx >= len(activeTmpl.Columns) {
			colNameEntry.SetText("")
			return
		}

		col := &activeTmpl.Columns[selectedColIdx]
		colNameEntry.SetText(col.DisplayName)
		colKeySelect.SetSelected(col.FieldKey)
		colAlignSelect.SetSelected(col.Align)

		colStartSlider.SetValue(float64(col.ColStart))
		colStartLabel.SetText(fmt.Sprintf("%d", col.ColStart))

		colWidthSlider.SetValue(float64(col.ColWidth))
		colWidthLabel.SetText(fmt.Sprintf("%d char", col.ColWidth))

		if btnMoveLeft != nil {
			if selectedColIdx == 0 {
				btnMoveLeft.Disable()
			} else {
				btnMoveLeft.Enable()
			}
		}

		if btnMoveRight != nil {
			if selectedColIdx == len(activeTmpl.Columns)-1 {
				btnMoveRight.Disable()
			} else {
				btnMoveRight.Enable()
			}
		}
	}

	// Connect inspector event listeners
	topMarginSlider.OnChanged = func(val float64) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		activeTmpl.TopMargin = int(val)
		topMarginLabel.SetText(fmt.Sprintf("%d baris", activeTmpl.TopMargin))
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	leftMarginSlider.OnChanged = func(val float64) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		activeTmpl.LeftMargin = int(val)
		leftMarginLabel.SetText(fmt.Sprintf("%d kolom", activeTmpl.LeftMargin))
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	colNameEntry.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil || selectedColIdx >= len(activeTmpl.Columns) {
			return
		}
		activeTmpl.Columns[selectedColIdx].DisplayName = s
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	colKeySelect.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil || selectedColIdx >= len(activeTmpl.Columns) {
			return
		}
		activeTmpl.Columns[selectedColIdx].FieldKey = s
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	colAlignSelect.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil || selectedColIdx >= len(activeTmpl.Columns) {
			return
		}
		activeTmpl.Columns[selectedColIdx].Align = s
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	colStartSlider.OnChanged = func(val float64) {
		if isUpdatingUI || activeTmpl == nil || selectedColIdx >= len(activeTmpl.Columns) {
			return
		}
		activeTmpl.Columns[selectedColIdx].ColStart = int(val)
		if selectedColIdx == 0 {
			MakeColumnsSticky(activeTmpl)
		}
		colStartLabel.SetText(fmt.Sprintf("%d", activeTmpl.Columns[selectedColIdx].ColStart))
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	colWidthSlider.OnChanged = func(val float64) {
		if isUpdatingUI || activeTmpl == nil || selectedColIdx >= len(activeTmpl.Columns) {
			return
		}
		activeTmpl.Columns[selectedColIdx].ColWidth = int(val)
		MakeColumnsSticky(activeTmpl)
		colWidthLabel.SetText(fmt.Sprintf("%d char", activeTmpl.Columns[selectedColIdx].ColWidth))
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateLivePreview()
	}

	btnMoveLeft = widget.NewButtonWithIcon("Geser Kiri", theme.NavigateBackIcon(), func() {
		if selectedColIdx > 0 && selectedColIdx < len(activeTmpl.Columns) {
			activeTmpl.Columns[selectedColIdx], activeTmpl.Columns[selectedColIdx-1] = activeTmpl.Columns[selectedColIdx-1], activeTmpl.Columns[selectedColIdx]
			MakeColumnsSticky(activeTmpl)
			selectedColIdx--
			canvasWidget.SetSelectedColumn(selectedColIdx)
			updateInspector()
			updateLivePreview()
		}
	})

	btnMoveRight = widget.NewButtonWithIcon("Geser Kanan", theme.NavigateNextIcon(), func() {
		if selectedColIdx < len(activeTmpl.Columns)-1 {
			activeTmpl.Columns[selectedColIdx], activeTmpl.Columns[selectedColIdx+1] = activeTmpl.Columns[selectedColIdx+1], activeTmpl.Columns[selectedColIdx]
			MakeColumnsSticky(activeTmpl)
			selectedColIdx++
			canvasWidget.SetSelectedColumn(selectedColIdx)
			updateInspector()
			updateLivePreview()
		}
	})

	btnDeleteCol = widget.NewButtonWithIcon("Hapus Kolom", theme.DeleteIcon(), func() {
		if len(activeTmpl.Columns) <= 1 {
			showInfo("Perhatian", "Template harus memiliki minimal 1 kolom.", win)
			return
		}
		activeTmpl.Columns = append(activeTmpl.Columns[:selectedColIdx], activeTmpl.Columns[selectedColIdx+1:]...)
		if selectedColIdx >= len(activeTmpl.Columns) {
			selectedColIdx = len(activeTmpl.Columns) - 1
		}
		MakeColumnsSticky(activeTmpl)
		canvasWidget.SetSelectedColumn(selectedColIdx)
		updateInspector()
		updateLivePreview()
	})

	btnAddCol = widget.NewButtonWithIcon("Tambah Kolom Baru", theme.ContentAddIcon(), func() {
		lastEnd := 0
		if len(activeTmpl.Columns) > 0 {
			last := activeTmpl.Columns[len(activeTmpl.Columns)-1]
			lastEnd = last.ColStart + last.ColWidth
		}
		if lastEnd > 72 {
			lastEnd = 65
		}

		newCol := config.ColumnDef{
			ID:          fmt.Sprintf("col_%d", time.Now().UnixNano()%100000),
			FieldKey:    "keterangan",
			DisplayName: "Kolom Baru",
			ColStart:    lastEnd,
			ColWidth:    10,
			Align:       "left",
		}
		activeTmpl.Columns = append(activeTmpl.Columns, newCol)
		MakeColumnsSticky(activeTmpl)
		selectedColIdx = len(activeTmpl.Columns) - 1
		canvasWidget.SetSelectedColumn(selectedColIdx)
		updateInspector()
		updateLivePreview()
	})

	// --- 4. Canvas Widget Initialization ---
	canvasWidget = NewPassbookCanvasWidget(activeTmpl, func(colIdx int) {
		selectedColIdx = colIdx
		updateInspector()
	}, func() {
		updateInspector()
		updateLivePreview()
	})

	// --- 5. TOP TOOLBAR: Template Switcher & Quick Tools ---
	var templateSelect *widget.Select
	var refreshTemplateList func()

	templateNames := make([]string, 0, len(state.Layouts.Templates))
	templateMap := make(map[string]*config.LayoutTemplate)
	for i := range state.Layouts.Templates {
		tmpl := &state.Layouts.Templates[i]
		templateNames = append(templateNames, tmpl.Name)
		templateMap[tmpl.Name] = tmpl
	}

	templateSelect = widget.NewSelect(templateNames, func(selectedName string) {
		if tmpl, ok := templateMap[selectedName]; ok {
			state.Layouts.ActiveTemplate = tmpl.ID
			activeTmpl = tmpl
			selectedColIdx = 0
			if canvasWidget != nil {
				canvasWidget.SetTemplate(activeTmpl)
			}
			updateInspector()
			updateLivePreview()
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})
	if activeTmpl != nil {
		templateSelect.Selected = activeTmpl.Name
	}

	refreshTemplateList = func() {
		templateNames = make([]string, 0, len(state.Layouts.Templates))
		templateMap = make(map[string]*config.LayoutTemplate)
		for i := range state.Layouts.Templates {
			tmpl := &state.Layouts.Templates[i]
			templateNames = append(templateNames, tmpl.Name)
			templateMap[tmpl.Name] = tmpl
		}
		templateSelect.Options = templateNames
		if activeTmpl != nil {
			templateSelect.Selected = activeTmpl.Name
		}
		templateSelect.Refresh()
	}

	btnAddTemplate := widget.NewButtonWithIcon("Buat Template Baru", theme.ContentAddIcon(), func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Contoh: Bank BJB Tandamata")

		d := showCustomConfirm("Buat Template Buku Baru", "Buat", "Batal", container.NewVBox(
			widget.NewLabel("Masukkan nama template buku tabungan baru:"),
			nameEntry,
		), func(confirmed bool) {
			if !confirmed || strings.TrimSpace(nameEntry.Text) == "" {
				return
			}
			newID := fmt.Sprintf("tmpl-%d", time.Now().Unix())
			newTmpl := config.LayoutTemplate{
				ID:         newID,
				Name:       strings.TrimSpace(nameEntry.Text),
				MaxColumns: 80,
				TopMargin:  0,
				LeftMargin: 0,
				Columns:    config.DefaultLayoutConfig().Templates[0].Columns,
			}
			state.Layouts.Templates = append(state.Layouts.Templates, newTmpl)
			state.Layouts.ActiveTemplate = newID
			activeTmpl = &state.Layouts.Templates[len(state.Layouts.Templates)-1]
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
			refreshTemplateList()
			if canvasWidget != nil {
				canvasWidget.SetTemplate(activeTmpl)
			}
			updateInspector()
			updateLivePreview()
		}, win)
		d.Show()
		win.Canvas().Focus(nameEntry)
	})

	btnAutoAlign := widget.NewButtonWithIcon("Rapatkan Kolom (Auto-Align)", theme.ViewRefreshIcon(), func() {
		curr := 0
		for i := range activeTmpl.Columns {
			activeTmpl.Columns[i].ColStart = curr
			curr += activeTmpl.Columns[i].ColWidth
		}
		if canvasWidget != nil {
			canvasWidget.Refresh()
		}
		updateInspector()
		updateLivePreview()
	})

	btnSaveLayout := widget.NewButtonWithIcon("Simpan Layout", theme.DocumentSaveIcon(), func() {
		err := config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
		if err != nil {
			showError(fmt.Errorf("gagal menyimpan layout: %w", err), win)
			return
		}
		state.Log(fmt.Sprintf("Konfigurasi layout [%s] berhasil disimpan.", activeTmpl.Name))
		state.NotifyStateChanged()
		showInfo("Tersimpan", fmt.Sprintf("Pengaturan layout '%s' berhasil disimpan ke %s!", activeTmpl.Name, state.LayoutsPath), win)
	})
	btnSaveLayout.Importance = widget.HighImportance

	btnTestPrint := widget.NewButtonWithIcon("Test Cetak Baris 14", theme.DocumentPrintIcon(), func() {
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

		formattedLine := escp.FormatLineWithTemplate(sampleReq, activeTmpl)
		state.Log("=== PENGUJIAN CETAK BUKU TABUNGAN (BARIS 14) ===")
		state.Log(fmt.Sprintf("Template Aktif: %s", activeTmpl.Name))
		state.Log(fmt.Sprintf("Hasil Cetak (%d char): |%s|", len(formattedLine), formattedLine))

		cmdBytes := escp.GeneratePrintCommand(sampleReq, escp.GeneratorOptions{
			EnableEject: state.Config.EnableEject,
			Template:    activeTmpl,
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
			showInfo("Berhasil Dicetak",
				fmt.Sprintf("Perintah cetak baris ke-14 berhasil dikirim ke printer!\n\nTeks Baris:\n%s", formattedLine), win)
		}
	})

	// --- 5. Zoom & Pan Controls ---
	zoomLabel := widget.NewLabelWithStyle("100%", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	btnZoomMinus := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
		if canvasWidget != nil {
			canvasWidget.ZoomOut()
		}
	})
	btnZoomPlus := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if canvasWidget != nil {
			canvasWidget.ZoomIn()
		}
	})
	btnZoomReset := widget.NewButton("100%", func() {
		if canvasWidget != nil {
			canvasWidget.ResetZoom()
		}
	})
	btnResetPan := widget.NewButtonWithIcon("Tengah", theme.ViewFullScreenIcon(), func() {
		if canvasWidget != nil {
			canvasWidget.ResetPan()
		}
	})

	if canvasWidget != nil {
		canvasWidget.OnScaleChanged = func(s float32) {
			zoomLabel.SetText(fmt.Sprintf("%d%%", int(s*100)))
		}
	}

	zoomBox := container.NewHBox(
		widget.NewLabelWithStyle("Zoom:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		btnZoomMinus,
		container.NewGridWrap(fyne.NewSize(52, 36), zoomLabel),
		btnZoomPlus,
		btnZoomReset,
		btnResetPan,
	)

	var split *container.Split
	var sidebarScroll *container.Scroll

	btnToggleSidebar := widget.NewButtonWithIcon("Sidebar", theme.MenuIcon(), func() {
		if sidebarScroll != nil && split != nil {
			if sidebarScroll.Visible() {
				sidebarScroll.Hide()
			} else {
				sidebarScroll.Show()
			}
			split.Refresh()
		}
	})

	topToolbar := container.NewHBox(
		widget.NewLabelWithStyle("Template:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWrap(fyne.NewSize(200, 36), templateSelect),
		btnAddTemplate,
		widget.NewSeparator(),
		zoomBox,
		widget.NewSeparator(),
		btnAutoAlign,
		btnSaveLayout,
		btnTestPrint,
		widget.NewSeparator(),
		btnToggleSidebar,
	)

	tipsBanner := widget.NewLabelWithStyle(
		"💡 Tips: Geser garis pembatas sidebar (↔) untuk atur ukuran (seperti VS Code) • Drag border kolom (↔) untuk ubah lebar sticky (Excel) • Scroll mouse untuk zoom",
		fyne.TextAlignCenter, fyne.TextStyle{Italic: true},
	)

	// --- 6. Assemble Sidebar & Canvas ---
	btnTopMinus := widget.NewButton("-1 Baris", func() {
		if activeTmpl.TopMargin > 0 {
			activeTmpl.TopMargin--
			topMarginSlider.SetValue(float64(activeTmpl.TopMargin))
		}
	})
	btnTopPlus := widget.NewButton("+1 Baris", func() {
		if activeTmpl.TopMargin < 18 {
			activeTmpl.TopMargin++
			topMarginSlider.SetValue(float64(activeTmpl.TopMargin))
		}
	})
	btnLeftMinus := widget.NewButton("-1 Kolom", func() {
		if activeTmpl.LeftMargin > 0 {
			activeTmpl.LeftMargin--
			leftMarginSlider.SetValue(float64(activeTmpl.LeftMargin))
		}
	})
	btnLeftPlus := widget.NewButton("+1 Kolom", func() {
		if activeTmpl.LeftMargin < 35 {
			activeTmpl.LeftMargin++
			leftMarginSlider.SetValue(float64(activeTmpl.LeftMargin))
		}
	})

	btnWMinus := widget.NewButton("-1", func() {
		if selectedColIdx >= 0 && selectedColIdx < len(activeTmpl.Columns) {
			if activeTmpl.Columns[selectedColIdx].ColWidth > 2 {
				colWidthSlider.SetValue(float64(activeTmpl.Columns[selectedColIdx].ColWidth - 1))
			}
		}
	})
	btnWPlus := widget.NewButton("+1", func() {
		if selectedColIdx >= 0 && selectedColIdx < len(activeTmpl.Columns) {
			colWidthSlider.SetValue(float64(activeTmpl.Columns[selectedColIdx].ColWidth + 1))
		}
	})
	btnWMinus5 := widget.NewButton("-5", func() {
		if selectedColIdx >= 0 && selectedColIdx < len(activeTmpl.Columns) {
			newW := activeTmpl.Columns[selectedColIdx].ColWidth - 5
			if newW < 2 {
				newW = 2
			}
			colWidthSlider.SetValue(float64(newW))
		}
	})
	btnWPlus5 := widget.NewButton("+5", func() {
		if selectedColIdx >= 0 && selectedColIdx < len(activeTmpl.Columns) {
			colWidthSlider.SetValue(float64(activeTmpl.Columns[selectedColIdx].ColWidth + 5))
		}
	})

	marginTip := widget.NewLabel("💡 Drag bar biru [:::] pada canvas atau gunakan kontrol di bawah:")
	marginTip.Wrapping = fyne.TextWrapWord

	rowTop := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Margin Atas (Baris):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		topMarginLabel,
	)
	rowLeft := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Margin Kiri (Kolom):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		leftMarginLabel,
	)

	tableMarginCard := widget.NewCard("Posisi & Margin Tabel", "",
		container.NewVBox(
			marginTip,
			rowTop,
			topMarginSlider,
			container.NewGridWithColumns(2, btnTopMinus, btnTopPlus),
			widget.NewSeparator(),
			rowLeft,
			leftMarginSlider,
			container.NewGridWithColumns(2, btnLeftMinus, btnLeftPlus),
		),
	)

	colForm := widget.NewForm(
		widget.NewFormItem("Nama Kolom", colNameEntry),
		widget.NewFormItem("Data Field", colKeySelect),
		widget.NewFormItem("Rata Teks", colAlignSelect),
	)

	rowStart := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Posisi Kolom (Start):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		colStartLabel,
	)
	rowWidth := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Lebar Kolom (Width):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		colWidthLabel,
	)

	rowUsage := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Kapasitas Terpakai:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		charUsageLabel,
	)

	columnInspectorCard := widget.NewCard("Inspector Kolom Terpilih", "",
		container.NewVBox(
			colForm,
			widget.NewSeparator(),
			rowStart,
			colStartSlider,
			widget.NewSeparator(),
			rowWidth,
			colWidthSlider,
			container.NewGridWithColumns(4, btnWMinus5, btnWMinus, btnWPlus, btnWPlus5),
			widget.NewSeparator(),
			rowUsage,
			widget.NewSeparator(),
			container.NewGridWithColumns(2, btnMoveLeft, btnMoveRight),
			container.NewGridWithColumns(2, btnAddCol, btnDeleteCol),
		),
	)

	sidebarVBox := container.NewVBox(
		tableMarginCard,
		widget.NewSeparator(),
		columnInspectorCard,
	)
	sidebarScroll = container.NewVScroll(sidebarVBox)

	// Center area: Full-height interactive Passbook Canvas
	centerArea := container.NewStack(canvasWidget)

	// Resizable split container (VS Code style resizable sidebar)
	split = container.NewHSplit(centerArea, sidebarScroll)
	split.SetOffset(0.71) // Default ~71% canvas, ~29% sidebar

	// Perform initial updates
	updateInspector()
	updateLivePreview()

	topArea := container.NewVBox(
		topToolbar,
		tipsBanner,
		widget.NewSeparator(),
	)

	return container.NewBorder(
		topArea,
		nil,
		nil,
		nil,
		split,
	)
}
