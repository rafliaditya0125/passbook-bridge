package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/escp"
	"passbook-bridge/internal/model"
	"passbook-bridge/internal/printer"
)

// clickableFontBox adalah widget card interaktif untuk menampilkan status font dan membuka dialog font via klik / double click
type clickableFontBox struct {
	widget.BaseWidget
	titleLabel   *canvas.Text
	descLabel    *canvas.Text
	OnOpenDialog func()
}

func newClickableFontBox(onOpenDialog func()) *clickableFontBox {
	b := &clickableFontBox{
		OnOpenDialog: onOpenDialog,
		titleLabel:   canvas.NewText("Arial Narrow, 10 pt", color.NRGBA{R: 15, G: 23, B: 42, A: 255}),
		descLabel:    canvas.NewText("Regular • 17 CPI", color.NRGBA{R: 100, G: 116, B: 139, A: 255}),
	}
	b.titleLabel.TextSize = 13.0
	b.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	b.descLabel.TextSize = 10.5
	b.ExtendBaseWidget(b)
	return b
}

func (b *clickableFontBox) SetFontInfo(family string, size int, style string, pitch int) {
	if family == "" {
		family = "Arial Narrow"
	}
	if size <= 0 {
		size = 10
	}
	if style == "" {
		style = "Regular"
	}
	if pitch <= 0 {
		pitch = 17
	}
	b.titleLabel.Text = fmt.Sprintf("🔤 %s, %d pt", family, size)
	b.descLabel.Text = fmt.Sprintf("Gaya: %s  •  Pitch: %d CPI Dot-Matrix", style, pitch)

	isMono := isMonospaceFont(family)
	isBold := strings.Contains(strings.ToLower(style), "bold")
	isItalic := strings.Contains(strings.ToLower(style), "italic")
	b.titleLabel.TextStyle = getSafeTextStyle(isMono, isBold, isItalic)
	b.Refresh()
}

func (b *clickableFontBox) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.NRGBA{R: 248, G: 250, B: 252, A: 255})
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = color.NRGBA{R: 203, G: 213, B: 225, A: 255}
	border.StrokeWidth = 1.0

	content := container.NewVBox(b.titleLabel, b.descLabel)
	padded := container.NewPadded(content)
	stack := container.NewStack(bg, border, padded)

	return widget.NewSimpleRenderer(stack)
}

func (b *clickableFontBox) Tapped(pe *fyne.PointEvent) {
	if b.OnOpenDialog != nil {
		b.OnOpenDialog()
	}
}

func (b *clickableFontBox) DoubleTapped(pe *fyne.PointEvent) {
	if b.OnOpenDialog != nil {
		b.OnOpenDialog()
	}
}

// doubleClickEntry adalah widget Entry kustom yang mendukung event DoubleTapped (klik 2x)
type doubleClickEntry struct {
	widget.Entry
	OnOpenDialog func()
}

func newDoubleClickEntry(onOpenDialog func()) *doubleClickEntry {
	e := &doubleClickEntry{OnOpenDialog: onOpenDialog}
	e.ExtendBaseWidget(e)
	return e
}

func (e *doubleClickEntry) DoubleTapped(pe *fyne.PointEvent) {
	if e.OnOpenDialog != nil {
		e.OnOpenDialog()
	}
}

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

	sampleReq := &model.PrintRequest{
		BarisKe:        14,
		Tanggal:        time.Now().Format("02-01-2006"),
		KodeTransaksi:  "K14",
		JenisTransaksi: "setor",
		Jumlah:         7000,
		SaldoSesudah:   7000,
		Petugas:        "adm",
		Keterangan:     "Setoran Target Baris 14",
	}

	var canvasWidget *PassbookCanvasWidget
	var rebuildColumnList func()
	var updateColumnInputs func()
	var templateSelect *widget.Select
	var refreshTemplateList func()
	var entryTargetRow *widget.Entry
	var entryTotalRows *widget.Entry
	var entryMaxCols *widget.Entry
	var entryTopMargin *widget.Entry
	var entryFontSize *doubleClickEntry
	var fontBoxWidget *clickableFontBox
	var updateFontUI func()

	updateSettingsUI := func() {
		if activeTmpl == nil {
			return
		}
		isUpdatingUI = true
		defer func() { isUpdatingUI = false }()

		if entryMaxCols != nil {
			curCols := activeTmpl.MaxColumns
			if curCols <= 0 {
				curCols = 100
			}
			entryMaxCols.SetText(fmt.Sprintf("%d", curCols))
		}
		if entryTotalRows != nil {
			entryTotalRows.SetText(fmt.Sprintf("%d", activeTmpl.GetTotalRows()))
		}
		if entryTopMargin != nil {
			entryTopMargin.SetText(fmt.Sprintf("%d", activeTmpl.TopMargin))
		}
		if updateFontUI != nil {
			updateFontUI()
		}
	}

	// --- 1. Canvas Widget Initialization ---
	canvasWidget = NewPassbookCanvasWidget(activeTmpl, func(colIdx int) {
		selectedColIdx = colIdx
		if rebuildColumnList != nil {
			rebuildColumnList()
		}
	}, func() {
		if updateColumnInputs != nil {
			updateColumnInputs()
		}
		_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
		state.NotifyStateChanged()
	})

	canvasWidget.OnTargetRowChanged = func(row int) {
		if entryTargetRow != nil && entryTargetRow.Text != fmt.Sprintf("%d", row) {
			isUpdatingUI = true
			entryTargetRow.SetText(fmt.Sprintf("%d", row))
			isUpdatingUI = false
		}
		sampleReq.SetBaris(row)
		sampleReq.KodeTransaksi = fmt.Sprintf("K%02d", row)
		sampleReq.Keterangan = fmt.Sprintf("Setoran Target Baris %d", row)
	}

	// --- 2. Kiri Panel - Card 1: Pengaturan Cetak Buku Tabungan ---
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
			if rebuildColumnList != nil {
				rebuildColumnList()
			}
			updateSettingsUI()
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

	btnAddTemplate := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
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
				MaxColumns: 100,
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
			if rebuildColumnList != nil {
				rebuildColumnList()
			}
		}, win)
		d.Show()
		win.Canvas().Focus(nameEntry)
	})

	templateRow := container.NewBorder(nil, nil, nil, btnAddTemplate, templateSelect)

	entryTotalRows = widget.NewEntry()
	entryTotalRows.SetText(fmt.Sprintf("%d", activeTmpl.GetTotalRows()))
	entryTotalRows.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 10 && v <= 100 {
			activeTmpl.TotalRows = v
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	}

	btnRowsMinus := widget.NewButton("-", func() {
		if activeTmpl.GetTotalRows() > 10 {
			activeTmpl.TotalRows = activeTmpl.GetTotalRows() - 1
			entryTotalRows.SetText(fmt.Sprintf("%d", activeTmpl.TotalRows))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	btnRowsPlus := widget.NewButton("+", func() {
		if activeTmpl.GetTotalRows() < 100 {
			activeTmpl.TotalRows = activeTmpl.GetTotalRows() + 1
			entryTotalRows.SetText(fmt.Sprintf("%d", activeTmpl.TotalRows))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	rowTotalRows := container.NewBorder(nil, nil, nil, container.NewHBox(btnRowsMinus, btnRowsPlus), entryTotalRows)

	entryMaxCols = widget.NewEntry()
	curColsInit := activeTmpl.MaxColumns
	if curColsInit <= 0 {
		curColsInit = 100
	}
	entryMaxCols.SetText(fmt.Sprintf("%d", curColsInit))
	entryMaxCols.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 20 && v <= 200 {
			activeTmpl.MaxColumns = v
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	}

	btnColsMinus := widget.NewButton("-", func() {
		if activeTmpl.MaxColumns > 20 {
			activeTmpl.MaxColumns--
			entryMaxCols.SetText(fmt.Sprintf("%d", activeTmpl.MaxColumns))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	btnColsPlus := widget.NewButton("+", func() {
		if activeTmpl.MaxColumns < 200 {
			activeTmpl.MaxColumns++
			entryMaxCols.SetText(fmt.Sprintf("%d", activeTmpl.MaxColumns))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	rowMaxCols := container.NewBorder(nil, nil, nil, container.NewHBox(btnColsMinus, btnColsPlus), entryMaxCols)

	entrySpasiSetelah := widget.NewEntry()
	entrySpasiSetelah.SetText("0")

	entryTopMargin = widget.NewEntry()
	entryTopMargin.SetText(fmt.Sprintf("%d", activeTmpl.TopMargin))
	entryTopMargin.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 0 && v <= 100 {
			activeTmpl.TopMargin = v
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	}

	btnTopMinus := widget.NewButton("-", func() {
		if activeTmpl.TopMargin > 0 {
			activeTmpl.TopMargin--
			entryTopMargin.SetText(fmt.Sprintf("%d", activeTmpl.TopMargin))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	btnTopPlus := widget.NewButton("+", func() {
		if activeTmpl.TopMargin < 100 {
			activeTmpl.TopMargin++
			entryTopMargin.SetText(fmt.Sprintf("%d", activeTmpl.TopMargin))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	rowTopMargin := container.NewBorder(nil, nil, nil, container.NewHBox(btnTopMinus, btnTopPlus), entryTopMargin)

	// Target Baris Awal Cetak (seperti pada form Cetak Buku Tabungan video Ichasoft)
	entryTargetRow = widget.NewEntry()
	entryTargetRow.SetText("14")
	entryTargetRow.OnChanged = func(s string) {
		if isUpdatingUI || canvasWidget == nil {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 1 && v <= 34 {
			canvasWidget.SetTargetRow(v)
			sampleReq.SetBaris(v)
			sampleReq.KodeTransaksi = fmt.Sprintf("K%02d", v)
			sampleReq.Keterangan = fmt.Sprintf("Setoran Target Baris %d", v)
		}
	}

	btnRowMinus := widget.NewButton("-", func() {
		cur := canvasWidget.GetTargetRow()
		if cur > 1 {
			canvasWidget.SetTargetRow(cur - 1)
		}
	})

	btnRowPlus := widget.NewButton("+", func() {
		cur := canvasWidget.GetTargetRow()
		if cur < 34 {
			canvasWidget.SetTargetRow(cur + 1)
		}
	})

	rowTargetRow := container.NewBorder(nil, nil, nil, container.NewHBox(btnRowMinus, btnRowPlus), entryTargetRow)

	entrySpasiAntar := widget.NewEntry()
	entrySpasiAntar.SetText("15")

	showFontDialog := func() {
		if activeTmpl == nil {
			return
		}

		fontFamilies := GetSystemFontFamilies()

		cleanFamilyName := func(val string) string {
			val = strings.TrimSpace(val)
			for _, f := range fontFamilies {
				if strings.EqualFold(val, f) {
					return f
				}
			}
			return val
		}

		curFam := cleanFamilyName(activeTmpl.GetFontFamily())
		if curFam == "" && len(fontFamilies) > 0 {
			curFam = fontFamilies[0]
		}

		selectFamily := widget.NewSelect(fontFamilies, nil)
		selectFamily.SetSelected(curFam)

		entrySearchFont := widget.NewEntry()
		entrySearchFont.SetPlaceHolder("Ketik untuk memfilter font...")
		entrySearchFont.OnChanged = func(q string) {
			q = strings.ToLower(strings.TrimSpace(q))
			if q == "" {
				selectFamily.Options = fontFamilies
			} else {
				var matches []string
				for _, f := range fontFamilies {
					if strings.Contains(strings.ToLower(f), q) {
						matches = append(matches, f)
					}
				}
				if len(matches) == 0 {
					matches = []string{curFam}
				}
				selectFamily.Options = matches
			}
			selectFamily.Refresh()
		}

		searchAndSelectFamily := container.NewVBox(
			entrySearchFont,
			selectFamily,
		)

		fontSizes := []string{"8", "9", "10", "11", "12", "14", "16", "18", "20"}
		selectSize := widget.NewSelect(fontSizes, nil)
		selectSize.SetSelected(fmt.Sprintf("%d", activeTmpl.GetFontSize()))

		fontStyles := []string{"Regular", "Bold", "Italic", "Bold Italic"}
		selectStyle := widget.NewSelect(fontStyles, nil)
		curStyle := activeTmpl.GetFontStyle()
		if curStyle != "" {
			selectStyle.SetSelected(curStyle)
		} else {
			selectStyle.SetSelected("Regular")
		}

		fontPitches := []string{
			"17 CPI (Condensed - Standar Buku Tabungan)",
			"15 CPI (Rapat)",
			"12 CPI (Elite - Sedang)",
			"10 CPI (Pica - Lebar)",
			"20 CPI (Sangat Rapat)",
		}
		selectPitch := widget.NewSelect(fontPitches, nil)
		switch activeTmpl.GetFontPitchCPI() {
		case 10:
			selectPitch.SetSelected("10 CPI (Pica - Lebar)")
		case 12:
			selectPitch.SetSelected("12 CPI (Elite - Sedang)")
		case 15:
			selectPitch.SetSelected("15 CPI (Rapat)")
		case 20:
			selectPitch.SetSelected("20 CPI (Sangat Rapat)")
		default:
			selectPitch.SetSelected("17 CPI (Condensed - Standar Buku Tabungan)")
		}

		// Kotak Preview Sampel Teks
		previewText := canvas.NewText("05-10-2021  K01  Setoran Tunai   5.000.000,00   adm", color.NRGBA{R: 15, G: 23, B: 42, A: 255})
		lblPreviewInfo := canvas.NewText("", color.NRGBA{R: 71, G: 85, B: 105, A: 255})
		lblPreviewInfo.TextSize = 10.5

		updateSamplePreview := func() {
			fam := cleanFamilyName(selectFamily.Selected)
			isMono := isMonospaceFont(fam)
			st := strings.ToLower(selectStyle.Selected)
			isBold := strings.Contains(st, "bold")
			isItalic := strings.Contains(st, "italic")

			previewText.TextStyle = getSafeTextStyle(isMono, isBold, isItalic)

			var cpiVal int
			var pitchDesc string
			switch {
			case strings.HasPrefix(selectPitch.Selected, "10"):
				cpiVal = 10
				pitchDesc = "10 CPI (Pica - Lebar)"
			case strings.HasPrefix(selectPitch.Selected, "12"):
				cpiVal = 12
				pitchDesc = "12 CPI (Elite - Sedang)"
			case strings.HasPrefix(selectPitch.Selected, "15"):
				cpiVal = 15
				pitchDesc = "15 CPI (Rapat)"
			case strings.HasPrefix(selectPitch.Selected, "20"):
				cpiVal = 20
				pitchDesc = "20 CPI (Sangat Rapat)"
			default:
				cpiVal = 17
				pitchDesc = "17 CPI (Condensed - Standar)"
			}

			previewBaseSize := float32(10.0)
			if s, err := strconv.Atoi(selectSize.Selected); err == nil && s >= 6 {
				previewBaseSize = float32(s)
			}

			// Skala teks pratinjau sesuai CPI (10 CPI lebih lebar/besar, 17-20 CPI lebih ramping)
			cpiScale := float32(17.14) / float32(cpiVal)
			previewText.TextSize = (previewBaseSize + 2.0) * (0.65 + 0.35*cpiScale)
			previewText.Refresh()

			famType := "Sans-Serif Proporsional"
			if isMono {
				famType = "Fixed-Width Monospace"
			}
			lblPreviewInfo.Text = fmt.Sprintf("Karakteristik: %s  •  Gaya: %s  •  Ukuran: %s pt  •  Pitch: %s",
				famType, selectStyle.Selected, selectSize.Selected, pitchDesc)
			lblPreviewInfo.Refresh()
		}

		selectFamily.OnChanged = func(string) { updateSamplePreview() }
		selectSize.OnChanged = func(string) { updateSamplePreview() }
		selectStyle.OnChanged = func(string) { updateSamplePreview() }
		selectPitch.OnChanged = func(string) { updateSamplePreview() }
		updateSamplePreview()

		previewBox := container.NewPadded(container.NewVBox(
			container.NewCenter(previewText),
			container.NewCenter(lblPreviewInfo),
		))
		previewCard := widget.NewCard("Pratinjau Font", "", previewBox)

		form := widget.NewForm(
			widget.NewFormItem("Jenis Font (Sistem)", searchAndSelectFamily),
			widget.NewFormItem("Ukuran Font (pt)", selectSize),
			widget.NewFormItem("Gaya Huruf (Style)", selectStyle),
			widget.NewFormItem("Pitch Dot-Matrix (CPI)", selectPitch),
		)

		d := showCustomConfirm(
			"Pilih Font Buku Tabungan (Font Sistem)",
			"Terapkan", "Batal",
			container.NewVBox(
				form,
				widget.NewSeparator(),
				previewCard,
			),
			func(confirmed bool) {
				if !confirmed {
					return
				}
				activeTmpl.FontFamily = cleanFamilyName(selectFamily.Selected)
				if s, err := strconv.Atoi(selectSize.Selected); err == nil {
					activeTmpl.FontSize = s
				}
				activeTmpl.FontStyle = selectStyle.Selected

				switch {
				case strings.HasPrefix(selectPitch.Selected, "10"):
					activeTmpl.FontPitchCPI = 10
				case strings.HasPrefix(selectPitch.Selected, "12"):
					activeTmpl.FontPitchCPI = 12
				case strings.HasPrefix(selectPitch.Selected, "15"):
					activeTmpl.FontPitchCPI = 15
				case strings.HasPrefix(selectPitch.Selected, "20"):
					activeTmpl.FontPitchCPI = 20
				default:
					activeTmpl.FontPitchCPI = 17
				}

				if updateFontUI != nil {
					updateFontUI()
				}

				if canvasWidget != nil {
					canvasWidget.Refresh()
				}
				_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
				state.NotifyStateChanged()
				state.Log(fmt.Sprintf("Font buku tabungan diubah: %s %dpt %s (%d CPI)",
					activeTmpl.GetFontFamily(), activeTmpl.GetFontSize(), activeTmpl.GetFontStyle(), activeTmpl.GetFontPitchCPI()))
			},
			win,
		)
		d.Show()
	}

	fontBoxWidget = newClickableFontBox(showFontDialog)

	entryFontSize = newDoubleClickEntry(showFontDialog)
	entryFontSize.SetText(fmt.Sprintf("%d", activeTmpl.GetFontSize()))
	entryFontSize.OnChanged = func(s string) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v >= 6 && v <= 24 {
			activeTmpl.FontSize = v
			if fontBoxWidget != nil {
				fontBoxWidget.SetFontInfo(
					activeTmpl.GetFontFamily(),
					activeTmpl.GetFontSize(),
					activeTmpl.GetFontStyle(),
					activeTmpl.GetFontPitchCPI(),
				)
			}
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	}

	btnFontMinus := widget.NewButton("-", func() {
		if activeTmpl.GetFontSize() > 6 {
			activeTmpl.FontSize = activeTmpl.GetFontSize() - 1
			if updateFontUI != nil {
				updateFontUI()
			}
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	btnFontPlus := widget.NewButton("+", func() {
		if activeTmpl.GetFontSize() < 24 {
			activeTmpl.FontSize = activeTmpl.GetFontSize() + 1
			if updateFontUI != nil {
				updateFontUI()
			}
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
			_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
			state.NotifyStateChanged()
		}
	})

	updateFontUI = func() {
		if activeTmpl == nil {
			return
		}
		if fontBoxWidget != nil {
			fontBoxWidget.SetFontInfo(
				activeTmpl.GetFontFamily(),
				activeTmpl.GetFontSize(),
				activeTmpl.GetFontStyle(),
				activeTmpl.GetFontPitchCPI(),
			)
		}
		if entryFontSize != nil {
			isUpdatingUI = true
			entryFontSize.SetText(fmt.Sprintf("%d", activeTmpl.GetFontSize()))
			isUpdatingUI = false
		}
	}
	updateFontUI()

	btnPickFont := widget.NewButtonWithIcon("Pilih Font...", theme.ColorPaletteIcon(), showFontDialog)

	fontSizeControls := container.NewBorder(nil, nil, widget.NewLabel("Ukuran (pt):"), container.NewHBox(btnFontMinus, btnFontPlus), entryFontSize)

	fontBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, btnPickFont, fontBoxWidget),
		fontSizeControls,
		widget.NewLabelWithStyle("💡 Klik 2x kotak font atau klik 'Pilih Font...' untuk ganti jenis font, gaya, & pitch CPI", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
	)

	printSettingsForm := widget.NewForm(
		widget.NewFormItem("Template Buku", templateRow),
		widget.NewFormItem("Maksimal Kolom", rowMaxCols),
		widget.NewFormItem("Jumlah Baris", rowTotalRows),
		widget.NewFormItem("Target Baris Awal Cetak", rowTargetRow),
		widget.NewFormItem("Margin Atas", rowTopMargin),
		widget.NewFormItem("Spasi Setelah Baris", entrySpasiSetelah),
		widget.NewFormItem("Spasi antar Baris", entrySpasiAntar),
		widget.NewFormItem("Font", fontBox),
	)

	printSettingsCard := widget.NewCard("Pengaturan Cetak Buku Tabungan", "", printSettingsForm)
	updateSettingsUI()

	// --- 3. Kiri Panel - Card 2: Posisi Kolom / Current X ( Tab ) ---
	type colBinding struct {
		idx        int
		posEntry   *widget.Entry
		widthEntry *widget.Entry
	}
	var bindings []colBinding
	columnRowsBox := container.NewVBox()

	showEditColumnDialog := func(colIdx int) {
		if colIdx < 0 || colIdx >= len(activeTmpl.Columns) {
			return
		}
		col := &activeTmpl.Columns[colIdx]

		nameEntry := widget.NewEntry()
		nameEntry.SetText(col.DisplayName)

		keySelect := widget.NewSelect([]string{
			"no", "tanggal", "kode_transaksi", "keterangan", "debet", "kredit", "saldo_sesudah", "petugas",
		}, nil)
		keySelect.SetSelected(col.FieldKey)

		alignSelect := widget.NewSelect([]string{"left", "center", "right"}, nil)
		alignSelect.SetSelected(col.Align)

		widthEntry := widget.NewEntry()
		widthEntry.SetText(fmt.Sprintf("%d", col.ColWidth))

		posEntry := widget.NewEntry()
		posEntry.SetText(fmt.Sprintf("%d", col.ColStart))

		form := widget.NewForm(
			widget.NewFormItem("Nama Tampilan (Header)", nameEntry),
			widget.NewFormItem("Field Data", keySelect),
			widget.NewFormItem("Rata Teks", alignSelect),
			widget.NewFormItem("Posisi X (Tab Stop)", posEntry),
			widget.NewFormItem("Lebar Kolom (Karakter)", widthEntry),
		)

		d := showCustomConfirm(
			fmt.Sprintf("Pengaturan Kolom '%s'", col.DisplayName),
			"Simpan", "Batal",
			form,
			func(confirmed bool) {
				if !confirmed {
					return
				}
				if strings.TrimSpace(nameEntry.Text) != "" {
					col.DisplayName = strings.TrimSpace(nameEntry.Text)
				}
				col.FieldKey = keySelect.Selected
				col.Align = alignSelect.Selected
				maxLimit := 100
				if activeTmpl != nil && activeTmpl.MaxColumns > 0 {
					maxLimit = activeTmpl.MaxColumns
				}
				if p, err := strconv.Atoi(strings.TrimSpace(posEntry.Text)); err == nil && p >= 0 && p <= maxLimit {
					col.ColStart = p
				}
				if w, err := strconv.Atoi(strings.TrimSpace(widthEntry.Text)); err == nil && w >= 2 && w <= maxLimit {
					col.ColWidth = w
				}
				if canvasWidget != nil {
					canvasWidget.Refresh()
				}
				if rebuildColumnList != nil {
					rebuildColumnList()
				}
			},
			win,
		)
		d.Show()
	}

	rebuildColumnList = func() {
		columnRowsBox.Objects = nil
		bindings = nil

		if activeTmpl == nil {
			return
		}

		for i := range activeTmpl.Columns {
			colIdx := i
			col := &activeTmpl.Columns[colIdx]
			colLetter := getExcelColLetter(colIdx)

			nameStyle := fyne.TextStyle{Bold: true}
			if colIdx == selectedColIdx {
				nameStyle.Italic = true
			}
			label := widget.NewLabelWithStyle(fmt.Sprintf("[%s] %s", colLetter, col.DisplayName), fyne.TextAlignLeading, nameStyle)
			if colIdx == selectedColIdx {
				label.Importance = widget.HighImportance
			}

			// Posisi X (Tab Stop)
			posEntry := widget.NewEntry()
			posEntry.SetText(fmt.Sprintf("%d", col.ColStart))
			posEntry.OnChanged = func(val string) {
				if isUpdatingUI || colIdx >= len(activeTmpl.Columns) {
					return
				}
				maxLimit := 100
				if activeTmpl != nil && activeTmpl.MaxColumns > 0 {
					maxLimit = activeTmpl.MaxColumns
				}
				if v, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && v >= 0 && v <= maxLimit {
					activeTmpl.Columns[colIdx].ColStart = v
					if canvasWidget != nil {
						canvasWidget.Refresh()
					}
				}
			}

			// Lebar Kolom (Width)
			widthEntry := widget.NewEntry()
			widthEntry.SetText(fmt.Sprintf("%d", col.ColWidth))
			widthEntry.OnChanged = func(val string) {
				if isUpdatingUI || colIdx >= len(activeTmpl.Columns) {
					return
				}
				maxLimit := 100
				if activeTmpl != nil && activeTmpl.MaxColumns > 0 {
					maxLimit = activeTmpl.MaxColumns
				}
				if v, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && v >= 2 && v <= maxLimit {
					activeTmpl.Columns[colIdx].ColWidth = v
					if canvasWidget != nil {
						canvasWidget.Refresh()
					}
				}
			}

			bindings = append(bindings, colBinding{idx: colIdx, posEntry: posEntry, widthEntry: widthEntry})

			btnPosMinus := widget.NewButton("-", func() {
				if colIdx < len(activeTmpl.Columns) && activeTmpl.Columns[colIdx].ColStart > 0 {
					activeTmpl.Columns[colIdx].ColStart--
					posEntry.SetText(fmt.Sprintf("%d", activeTmpl.Columns[colIdx].ColStart))
					if canvasWidget != nil {
						canvasWidget.Refresh()
					}
				}
			})

			btnPosPlus := widget.NewButton("+", func() {
				maxLimit := 100
				if activeTmpl != nil && activeTmpl.MaxColumns > 0 {
					maxLimit = activeTmpl.MaxColumns
				}
				if colIdx < len(activeTmpl.Columns) && activeTmpl.Columns[colIdx].ColStart < maxLimit {
					activeTmpl.Columns[colIdx].ColStart++
					posEntry.SetText(fmt.Sprintf("%d", activeTmpl.Columns[colIdx].ColStart))
					if canvasWidget != nil {
						canvasWidget.Refresh()
					}
				}
			})

			btnWidthMinus := widget.NewButton("-", func() {
				if colIdx < len(activeTmpl.Columns) && activeTmpl.Columns[colIdx].ColWidth > 2 {
					activeTmpl.Columns[colIdx].ColWidth--
					widthEntry.SetText(fmt.Sprintf("%d", activeTmpl.Columns[colIdx].ColWidth))
					if canvasWidget != nil {
						canvasWidget.Refresh()
					}
					updateColumnInputs()
				}
			})

			btnWidthPlus := widget.NewButton("+", func() {
				maxLimit := 100
				if activeTmpl != nil && activeTmpl.MaxColumns > 0 {
					maxLimit = activeTmpl.MaxColumns
				}
				if colIdx < len(activeTmpl.Columns) && activeTmpl.Columns[colIdx].ColWidth < maxLimit {
					activeTmpl.Columns[colIdx].ColWidth++
					widthEntry.SetText(fmt.Sprintf("%d", activeTmpl.Columns[colIdx].ColWidth))
					if canvasWidget != nil {
						canvasWidget.Refresh()
					}
					updateColumnInputs()
				}
			})

			btnEdit := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
				selectedColIdx = colIdx
				if canvasWidget != nil {
					canvasWidget.SetSelectedColumn(colIdx)
				}
				showEditColumnDialog(colIdx)
			})

			posXControls := container.NewHBox(
				widget.NewLabel("Tab X:"),
				container.NewGridWrap(fyne.NewSize(42, 34), posEntry),
				btnPosMinus,
				btnPosPlus,
			)

			widthControls := container.NewHBox(
				widget.NewLabel("Lebar:"),
				container.NewGridWrap(fyne.NewSize(38, 34), widthEntry),
				btnWidthMinus,
				btnWidthPlus,
				btnEdit,
			)

			colRow := container.NewBorder(
				nil, nil,
				container.NewGridWrap(fyne.NewSize(110, 36), label),
				widthControls,
				posXControls,
			)

			columnRowsBox.Add(colRow)
		}

		columnRowsBox.Refresh()
	}

	updateColumnInputs = func() {
		isUpdatingUI = true
		defer func() { isUpdatingUI = false }()

		if activeTmpl == nil {
			return
		}

		for _, b := range bindings {
			if b.idx < len(activeTmpl.Columns) {
				expectedPos := fmt.Sprintf("%d", activeTmpl.Columns[b.idx].ColStart)
				if b.posEntry.Text != expectedPos {
					b.posEntry.SetText(expectedPos)
				}
				expectedW := fmt.Sprintf("%d", activeTmpl.Columns[b.idx].ColWidth)
				if b.widthEntry.Text != expectedW {
					b.widthEntry.SetText(expectedW)
				}
			}
		}

		expectedTop := fmt.Sprintf("%d", activeTmpl.TopMargin)
		if entryTopMargin.Text != expectedTop {
			entryTopMargin.SetText(expectedTop)
		}
	}

	btnAddCol := widget.NewButtonWithIcon("Tambah Kolom", theme.ContentAddIcon(), func() {
		if activeTmpl == nil {
			return
		}
		maxLimit := 100
		if activeTmpl.MaxColumns > 0 {
			maxLimit = activeTmpl.MaxColumns
		}
		lastEnd := 0
		if len(activeTmpl.Columns) > 0 {
			last := activeTmpl.Columns[len(activeTmpl.Columns)-1]
			lastEnd = last.ColStart + last.ColWidth
		}
		if lastEnd > maxLimit-10 {
			lastEnd = maxLimit - 15
			if lastEnd < 0 {
				lastEnd = 0
			}
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
		selectedColIdx = len(activeTmpl.Columns) - 1
		if canvasWidget != nil {
			canvasWidget.SetSelectedColumn(selectedColIdx)
		}
		rebuildColumnList()
	})

	btnDeleteCol := widget.NewButtonWithIcon("Hapus Kolom", theme.DeleteIcon(), func() {
		if activeTmpl == nil || len(activeTmpl.Columns) <= 1 {
			showInfo("Perhatian", "Template harus memiliki minimal 1 kolom.", win)
			return
		}
		if selectedColIdx < 0 || selectedColIdx >= len(activeTmpl.Columns) {
			selectedColIdx = len(activeTmpl.Columns) - 1
		}
		activeTmpl.Columns = append(activeTmpl.Columns[:selectedColIdx], activeTmpl.Columns[selectedColIdx+1:]...)
		if selectedColIdx >= len(activeTmpl.Columns) {
			selectedColIdx = len(activeTmpl.Columns) - 1
		}
		if canvasWidget != nil {
			canvasWidget.SetSelectedColumn(selectedColIdx)
		}
		rebuildColumnList()
	})

	btnAutoAlign := widget.NewButtonWithIcon("Rapatkan Kolom (Excel)", theme.ViewRefreshIcon(), func() {
		if activeTmpl == nil {
			return
		}
		MakeColumnsSticky(activeTmpl)
		if canvasWidget != nil {
			canvasWidget.SetStickyMode(true)
			canvasWidget.Refresh()
		}
		updateColumnInputs()
		showInfo("Kolom Rapat", "Seluruh posisi kolom berhasil dirapatkan menempel seperti sel spreadsheet Excel.", win)
	})

	colButtons := container.NewGridWithColumns(3, btnAddCol, btnDeleteCol, btnAutoAlign)

	colPositionCard := widget.NewCard("Posisi Kolom/ Current X ( Tab )", "",
		container.NewVBox(
			widget.NewLabelWithStyle("Ketik nilai Tab X / Lebar, atau langsung geser garis pada tabel Excel di preview:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			columnRowsBox,
			widget.NewSeparator(),
			colButtons,
		),
	)

	// Build initial list
	rebuildColumnList()

	// --- 4. Kanan Panel - Card 3: Preview Spreadsheet Excel Interaktif ---
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
	btnResetPan := widget.NewButtonWithIcon("Pusatkan", theme.ViewFullScreenIcon(), func() {
		if canvasWidget != nil {
			canvasWidget.ResetPan()
		}
	})

	if canvasWidget != nil {
		canvasWidget.OnScaleChanged = func(s float32) {
			zoomLabel.SetText(fmt.Sprintf("%d%%", int(s*100)))
		}
	}

	// Quick Column Action buttons di atas preview
	btnQuickStartMinus := widget.NewButton("◀ Tab X", func() {
		if canvasWidget != nil {
			canvasWidget.AdjustSelectedStart(-1)
		}
	})
	btnQuickStartPlus := widget.NewButton("Tab X ▶", func() {
		if canvasWidget != nil {
			canvasWidget.AdjustSelectedStart(1)
		}
	})
	btnQuickWidthMinus := widget.NewButton("− Lebar", func() {
		if canvasWidget != nil {
			canvasWidget.AdjustSelectedWidth(-1)
		}
	})
	btnQuickWidthPlus := widget.NewButton("+ Lebar", func() {
		if canvasWidget != nil {
			canvasWidget.AdjustSelectedWidth(1)
		}
	})

	var btnStickyToggle *widget.Button
	btnStickyToggle = widget.NewButton("🔒 Mode Kolom Rapat (Excel)", func() {
		if canvasWidget != nil {
			newSticky := !canvasWidget.IsStickyMode()
			canvasWidget.SetStickyMode(newSticky)
			if newSticky {
				btnStickyToggle.SetText("🔒 Mode Kolom Rapat (Excel)")
			} else {
				btnStickyToggle.SetText("🔓 Mode Tab Bebas")
			}
		}
	})

	previewTopBar := container.NewHBox(
		widget.NewLabelWithStyle("Alat Kolom:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		btnQuickStartMinus,
		btnQuickStartPlus,
		btnQuickWidthMinus,
		btnQuickWidthPlus,
		btnStickyToggle,
		layout.NewSpacer(),
		widget.NewLabelWithStyle("Zoom:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		btnZoomMinus,
		container.NewGridWrap(fyne.NewSize(52, 34), zoomLabel),
		btnZoomPlus,
		btnZoomReset,
		btnResetPan,
	)

	previewCard := widget.NewCard("Preview Spreadsheet Buku Tabungan (Interaktif)", "",
		container.NewBorder(
			previewTopBar,
			nil,
			nil,
			nil,
			canvasWidget,
		),
	)

	// --- 5. Bawah Panel - Action Bar: Simpan, Test Cetak, Pilih Nama Printer ---
	btnSaveLayout := widget.NewButtonWithIcon("Simpan", theme.DocumentSaveIcon(), func() {
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

	btnTestPrint := widget.NewButtonWithIcon("Test Cetak", theme.DocumentPrintIcon(), func() {
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

		targetRow := canvasWidget.GetTargetRow()
		sampleReq.SetBaris(targetRow)
		sampleReq.KodeTransaksi = fmt.Sprintf("K%02d", targetRow)
		sampleReq.Keterangan = fmt.Sprintf("Setoran Target Baris %d", targetRow)

		formattedLine := escp.FormatLineWithTemplate(sampleReq, activeTmpl)
		state.Log("=== PENGUJIAN CETAK BUKU TABUNGAN ===")
		state.Log(fmt.Sprintf("Template Aktif: %s (Baris Target: %d)", activeTmpl.Name, targetRow))
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
				fmt.Sprintf("Perintah cetak buku tabungan berhasil dikirim ke printer '%s'!\n\nBaris Target Cetak: %d\nTeks Hasil Cetak:\n%s", state.Config.PrinterName, targetRow, formattedLine), win)
		}
	})

	// Pindai printer perangkat sistem
	var printerSelect *widget.Select
	devs := printer.ScanAllDevices()
	var devNames []string
	currPrinter := state.Config.PrinterName
	if currPrinter == "" {
		currPrinter = "EPSON PLQ-20 ESC/P2"
	}
	devNames = append(devNames, currPrinter)
	for _, d := range devs {
		if d.Name != "" && d.Name != currPrinter {
			devNames = append(devNames, d.Name)
		}
	}
	if len(devNames) == 1 && currPrinter == "EPSON PLQ-20 ESC/P2" {
		devNames = append(devNames, "Generic / Text Only", "Simulated Mock Passbook Printer")
	}

	printerSelect = widget.NewSelect(devNames, func(s string) {
		state.Config.PrinterName = s
		_ = config.SaveConfig(state.ConfigPath, state.Config)
	})
	printerSelect.Selected = currPrinter

	bottomBar := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(130, 38), btnSaveLayout),
		container.NewGridWrap(fyne.NewSize(140, 38), btnTestPrint),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Pilih Nama Printer Yang digunakan:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWrap(fyne.NewSize(260, 38), printerSelect),
		layout.NewSpacer(),
	)

	// --- 6. Tata Letak Dua Kolom (Kiri: Pengaturan & Posisi Kolom, Kanan: Preview Spreadsheet) ---
	leftVBox := container.NewVBox(
		printSettingsCard,
		widget.NewSeparator(),
		colPositionCard,
	)
	leftScroll := container.NewVScroll(leftVBox)

	split := container.NewHSplit(leftScroll, previewCard)
	split.SetOffset(0.35) // ~35% panel formulir, ~65% preview visual spreadsheet

	return container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), bottomBar),
		nil,
		nil,
		split,
	)
}
