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

// BookSizePreset merepresentasikan preset fisik ukuran buku tabungan (P x L).
type BookSizePreset struct {
	Name        string
	WidthMM     float64
	HeightMM    float64
	DefaultRows int
	MaxColumns  int
}

var standardBookSizes = []BookSizePreset{
	{
		Name:        "14,8 x 10,5 cm (Standar BPR / Koperasi)",
		WidthMM:     148.0,
		HeightMM:    105.0,
		DefaultRows: 34,
		MaxColumns:  80,
	},
	{
		Name:        "15,0 x 10,0 cm (Bank Mini / Sekolah)",
		WidthMM:     150.0,
		HeightMM:    100.0,
		DefaultRows: 30,
		MaxColumns:  80,
	},
	{
		Name:        "16,0 x 11,0 cm (Passbook Lebar)",
		WidthMM:     160.0,
		HeightMM:    110.0,
		DefaultRows: 34,
		MaxColumns:  85,
	},
	{
		Name:        "21,0 x 14,8 cm (Buku Besar A5)",
		WidthMM:     210.0,
		HeightMM:    148.0,
		DefaultRows: 40,
		MaxColumns:  96,
	},
}

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
	var selectBookSize *widget.Select
	var entryTotalRows *widget.Entry
	var entryTopMargin *widget.Entry
	var entryFontSize *doubleClickEntry
	var fontBoxWidget *clickableFontBox
	var updateFontUI func()

	updateBookSizeUI := func() {
		if activeTmpl == nil {
			return
		}
		isUpdatingUI = true
		defer func() { isUpdatingUI = false }()

		matched := false
		for _, preset := range standardBookSizes {
			if activeTmpl.BookSizeName == preset.Name || (activeTmpl.PageWidthMM == preset.WidthMM && activeTmpl.PageHeightMM == preset.HeightMM) {
				if selectBookSize != nil {
					selectBookSize.Selected = preset.Name
					selectBookSize.Refresh()
				}
				matched = true
				break
			}
		}
		if !matched && selectBookSize != nil {
			if activeTmpl.BookSizeName != "" {
				selectBookSize.Selected = activeTmpl.BookSizeName
			} else {
				selectBookSize.Selected = "Kustom Ukuran..."
			}
			selectBookSize.Refresh()
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
	})

	canvasWidget.OnTargetRowChanged = func(row int) {
		if entryTargetRow != nil && entryTargetRow.Text != fmt.Sprintf("%d", row) {
			isUpdatingUI = true
			entryTargetRow.SetText(fmt.Sprintf("%d", row))
			isUpdatingUI = false
		}
		sampleReq.BarisKe = row
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
			updateBookSizeUI()
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
			if rebuildColumnList != nil {
				rebuildColumnList()
			}
		}, win)
		d.Show()
		win.Canvas().Focus(nameEntry)
	})

	templateRow := container.NewBorder(nil, nil, nil, btnAddTemplate, templateSelect)

	showCustomSizeDialog := func() {
		curW, curH := activeTmpl.GetPageDimensions()
		curRows := activeTmpl.GetTotalRows()
		curCols := activeTmpl.MaxColumns
		if curCols <= 0 {
			curCols = 80
		}

		entryWidthCM := widget.NewEntry()
		entryWidthCM.SetText(fmt.Sprintf("%0.1f", curW/10.0))

		entryHeightCM := widget.NewEntry()
		entryHeightCM.SetText(fmt.Sprintf("%0.1f", curH/10.0))

		entryRowsCustom := widget.NewEntry()
		entryRowsCustom.SetText(fmt.Sprintf("%d", curRows))

		entryColsCustom := widget.NewEntry()
		entryColsCustom.SetText(fmt.Sprintf("%d", curCols))

		form := widget.NewForm(
			widget.NewFormItem("Lebar Buku (cm)", entryWidthCM),
			widget.NewFormItem("Tinggi Buku (cm)", entryHeightCM),
			widget.NewFormItem("Jumlah Baris Buku", entryRowsCustom),
			widget.NewFormItem("Maksimal Kolom (ch)", entryColsCustom),
		)

		d := showCustomConfirm(
			"Atur Ukuran Buku Kustom",
			"Terapkan", "Batal",
			form,
			func(confirmed bool) {
				if !confirmed {
					return
				}
				wCM, errW := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(entryWidthCM.Text), ",", "."), 64)
				hCM, errH := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(entryHeightCM.Text), ",", "."), 64)
				r, errR := strconv.Atoi(strings.TrimSpace(entryRowsCustom.Text))
				c, errC := strconv.Atoi(strings.TrimSpace(entryColsCustom.Text))

				if errW == nil && wCM >= 5.0 && wCM <= 50.0 {
					activeTmpl.PageWidthMM = wCM * 10.0
				}
				if errH == nil && hCM >= 5.0 && hCM <= 50.0 {
					activeTmpl.PageHeightMM = hCM * 10.0
				}
				if errR == nil && r >= 10 && r <= 100 {
					activeTmpl.TotalRows = r
				}
				if errC == nil && c >= 20 && c <= 200 {
					activeTmpl.MaxColumns = c
				}
				activeTmpl.BookSizeName = fmt.Sprintf("Kustom (%0.1f x %0.1f cm)", activeTmpl.PageWidthMM/10.0, activeTmpl.PageHeightMM/10.0)

				if entryTotalRows != nil {
					isUpdatingUI = true
					entryTotalRows.SetText(fmt.Sprintf("%d", activeTmpl.GetTotalRows()))
					isUpdatingUI = false
				}

				if canvasWidget != nil {
					canvasWidget.Refresh()
				}
				_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
				state.NotifyStateChanged()
				state.Log(fmt.Sprintf("Ukuran buku kustom diterapkan: %0.1f x %0.1f cm (%d baris)",
					activeTmpl.PageWidthMM/10.0, activeTmpl.PageHeightMM/10.0, activeTmpl.GetTotalRows()))
			},
			win,
		)
		d.Show()
	}

	// Ukuran Buku Tabungan (P x L) - Preset Standar & Opsi Kustom
	bookSizeOptions := []string{
		"14,8 x 10,5 cm (Standar BPR / Koperasi)",
		"15,0 x 10,0 cm (Bank Mini / Sekolah)",
		"16,0 x 11,0 cm (Passbook Lebar)",
		"21,0 x 14,8 cm (Buku Besar A5)",
		"Kustom Ukuran...",
	}
	selectBookSize = widget.NewSelect(bookSizeOptions, func(s string) {
		if isUpdatingUI || activeTmpl == nil {
			return
		}
		if s == "Kustom Ukuran..." {
			showCustomSizeDialog()
			return
		}
		for _, preset := range standardBookSizes {
			if preset.Name == s {
				activeTmpl.BookSizeName = preset.Name
				activeTmpl.PageWidthMM = preset.WidthMM
				activeTmpl.PageHeightMM = preset.HeightMM
				activeTmpl.TotalRows = preset.DefaultRows
				activeTmpl.MaxColumns = preset.MaxColumns

				isUpdatingUI = true
				if entryTotalRows != nil {
					entryTotalRows.SetText(fmt.Sprintf("%d", activeTmpl.TotalRows))
				}
				isUpdatingUI = false

				if canvasWidget != nil {
					canvasWidget.Refresh()
				}
				_ = config.SaveLayoutConfig(state.LayoutsPath, state.Layouts)
				state.NotifyStateChanged()
				state.Log(fmt.Sprintf("Ukuran buku diterapkan: '%s' (%.1f x %.1f mm, %d baris, %d kolom)",
					preset.Name, preset.WidthMM, preset.HeightMM, preset.DefaultRows, preset.MaxColumns))
				break
			}
		}
	})

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
		}
	}

	btnTopMinus := widget.NewButton("-", func() {
		if activeTmpl.TopMargin > 0 {
			activeTmpl.TopMargin--
			entryTopMargin.SetText(fmt.Sprintf("%d", activeTmpl.TopMargin))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
		}
	})

	btnTopPlus := widget.NewButton("+", func() {
		if activeTmpl.TopMargin < 100 {
			activeTmpl.TopMargin++
			entryTopMargin.SetText(fmt.Sprintf("%d", activeTmpl.TopMargin))
			if canvasWidget != nil {
				canvasWidget.Refresh()
			}
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
			sampleReq.BarisKe = v
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

		fontFamilies := []string{
			"Arial Narrow",
			"Courier New",
			"Consolas",
			"Lucida Console",
			"Tahoma",
			"Segoe UI",
			"Times New Roman",
			"ESC/P Draft",
		}

		cleanFamilyName := func(val string) string {
			for _, f := range fontFamilies {
				if strings.EqualFold(strings.TrimSpace(val), f) || strings.Contains(strings.ToLower(val), strings.ToLower(f)) {
					return f
				}
			}
			return strings.TrimSpace(val)
		}

		selectFamily := widget.NewSelect(fontFamilies, nil)
		curFam := cleanFamilyName(activeTmpl.GetFontFamily())
		selectFamily.SetSelected(curFam)
		if selectFamily.Selected == "" {
			selectFamily.SetSelected("Arial Narrow")
		}

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
		}
		selectPitch := widget.NewSelect(fontPitches, nil)
		switch activeTmpl.GetFontPitchCPI() {
		case 10:
			selectPitch.SetSelected("10 CPI (Pica - Lebar)")
		case 12:
			selectPitch.SetSelected("12 CPI (Elite - Sedang)")
		case 15:
			selectPitch.SetSelected("15 CPI (Rapat)")
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
			if s, err := strconv.Atoi(selectSize.Selected); err == nil && s >= 6 {
				previewText.TextSize = float32(s) + 3.0 // skala agar jelas di dialog
			}
			previewText.Refresh()

			famType := "Sans-Serif Proporsional"
			if isMono {
				famType = "Fixed-Width Monospace"
			}
			lblPreviewInfo.Text = fmt.Sprintf("Karakteristik: %s  •  Gaya: %s  •  Ukuran: %s pt",
				famType, selectStyle.Selected, selectSize.Selected)
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
			widget.NewFormItem("Jenis Font (Font Family)", selectFamily),
			widget.NewFormItem("Ukuran Font (pt)", selectSize),
			widget.NewFormItem("Gaya Huruf (Style)", selectStyle),
			widget.NewFormItem("Pitch Dot-Matrix (CPI)", selectPitch),
		)

		d := showCustomConfirm(
			"Pilih Font Buku Tabungan",
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
		widget.NewFormItem("Ukuran Buku (P x L)", selectBookSize),
		widget.NewFormItem("Jumlah Baris", entryTotalRows),
		widget.NewFormItem("Target Baris Awal Cetak", rowTargetRow),
		widget.NewFormItem("Margin Atas", rowTopMargin),
		widget.NewFormItem("Spasi Setelah Baris", entrySpasiSetelah),
		widget.NewFormItem("Spasi antar Baris", entrySpasiAntar),
		widget.NewFormItem("Font", fontBox),
	)

	printSettingsCard := widget.NewCard("Pengaturan Cetak Buku Tabungan", "", printSettingsForm)
	updateBookSizeUI()

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
				if p, err := strconv.Atoi(strings.TrimSpace(posEntry.Text)); err == nil && p >= 0 && p <= 80 {
					col.ColStart = p
				}
				if w, err := strconv.Atoi(strings.TrimSpace(widthEntry.Text)); err == nil && w >= 2 && w <= 80 {
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
				if v, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && v >= 0 && v <= 80 {
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
				if v, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && v >= 2 && v <= 80 {
					activeTmpl.Columns[colIdx].ColWidth = v
					if canvasWidget != nil {
						if canvasWidget.IsStickyMode() {
							MakeColumnsSticky(activeTmpl)
						}
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
				if colIdx < len(activeTmpl.Columns) && activeTmpl.Columns[colIdx].ColStart < 80 {
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
						if canvasWidget.IsStickyMode() {
							MakeColumnsSticky(activeTmpl)
						}
						canvasWidget.Refresh()
					}
					updateColumnInputs()
				}
			})

			btnWidthPlus := widget.NewButton("+", func() {
				if colIdx < len(activeTmpl.Columns) && activeTmpl.Columns[colIdx].ColWidth < 80 {
					activeTmpl.Columns[colIdx].ColWidth++
					widthEntry.SetText(fmt.Sprintf("%d", activeTmpl.Columns[colIdx].ColWidth))
					if canvasWidget != nil {
						if canvasWidget.IsStickyMode() {
							MakeColumnsSticky(activeTmpl)
						}
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
		sampleReq.BarisKe = targetRow
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
