package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ColumnDef merepresentasikan definisi satu kolom pada buku tabungan.
type ColumnDef struct {
	ID          string `json:"id"`
	FieldKey    string `json:"field_key"`
	DisplayName string `json:"display_name"`
	ColStart    int    `json:"col_start"`
	ColWidth    int    `json:"col_width"`
	Align       string `json:"align"` // "left", "right", "center"
}

// LayoutTemplate merepresentasikan satu preset layout buku tabungan bank tertentu.
type LayoutTemplate struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	MaxColumns   int         `json:"max_columns"`
	TotalRows    int         `json:"total_rows,omitempty"`     // Jumlah baris dalam 1 halaman buku (misal 34)
	PageWidthMM  float64     `json:"page_width_mm,omitempty"`  // Lebar fisik halaman buku tabungan dalam mm (misal 148.0)
	PageHeightMM float64     `json:"page_height_mm,omitempty"` // Tinggi fisik halaman buku tabungan dalam mm (misal 105.0)
	BookSizeName string      `json:"book_size_name,omitempty"` // Nama preset ukuran (misal "14,8 x 10,5 cm")
	TopMargin    int         `json:"top_margin,omitempty"`     // offset baris/margin atas (baris ke-n)
	LeftMargin   int         `json:"left_margin,omitempty"`    // offset kolom/margin kiri
	Columns      []ColumnDef `json:"columns"`
}

// GetPageDimensions mengembalikan dimensi fisik buku tabungan (lebar & tinggi dalam mm).
func (t *LayoutTemplate) GetPageDimensions() (widthMM, heightMM float64) {
	if t != nil && t.PageWidthMM > 20 && t.PageHeightMM > 20 {
		return t.PageWidthMM, t.PageHeightMM
	}
	// Standar ukuran fisik buku tabungan perbankan: 148 mm x 105 mm (14.8 x 10.5 cm)
	return 148.0, 105.0
}

// GetTotalRows mengembalikan total baris buku tabungan (default 34 baris).
func (t *LayoutTemplate) GetTotalRows() int {
	if t != nil && t.TotalRows > 0 {
		return t.TotalRows
	}
	return 34
}

// LayoutConfig menyimpan seluruh template dan template aktif yang dipilih.
type LayoutConfig struct {
	ActiveTemplate string           `json:"active_template"`
	Templates      []LayoutTemplate `json:"templates"`
}

// DefaultLayoutConfig menghasilkan konfigurasi layout standar passbook (sesuai format Buku Tabungan 34 Baris).
func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		ActiveTemplate: "tabungan-standar-34",
		Templates: []LayoutTemplate{
			{
				ID:           "tabungan-standar-34",
				Name:         "Buku Tabungan Standar (34 Baris)",
				MaxColumns:   80,
				TotalRows:    34,
				PageWidthMM:  148.0,
				PageHeightMM: 105.0,
				BookSizeName: "14,8 x 10,5 cm (Standar BPR / Koperasi)",
				TopMargin:    0,
				LeftMargin:   0,
				Columns: []ColumnDef{
					{
						ID:          "col_no",
						FieldKey:    "no",
						DisplayName: "No",
						ColStart:    0,
						ColWidth:    4,
						Align:       "left",
					},
					{
						ID:          "col_tanggal",
						FieldKey:    "tanggal",
						DisplayName: "Tanggal",
						ColStart:    4,
						ColWidth:    11,
						Align:       "left",
					},
					{
						ID:          "col_sandi",
						FieldKey:    "kode_transaksi",
						DisplayName: "Kode/Sandi",
						ColStart:    16,
						ColWidth:    6,
						Align:       "center",
					},
					{
						ID:          "col_debet",
						FieldKey:    "debet",
						DisplayName: "Debet",
						ColStart:    23,
						ColWidth:    12,
						Align:       "right",
					},
					{
						ID:          "col_kredit",
						FieldKey:    "kredit",
						DisplayName: "Kredit",
						ColStart:    36,
						ColWidth:    12,
						Align:       "right",
					},
					{
						ID:          "col_saldo",
						FieldKey:    "saldo_sesudah",
						DisplayName: "Saldo",
						ColStart:    49,
						ColWidth:    14,
						Align:       "right",
					},
					{
						ID:          "col_paraf",
						FieldKey:    "petugas",
						DisplayName: "Ket/Paraf",
						ColStart:    64,
						ColWidth:    12,
						Align:       "center",
					},
				},
			},
			{
				ID:         "tabungan-custom-x",
				Name:       "Pengaturan Tabungan (Current X: 0, 11, 70, 115...)",
				MaxColumns: 420,
				TopMargin:  52,
				LeftMargin: 0,
				Columns: []ColumnDef{
					{
						ID:          "col_no",
						FieldKey:    "no",
						DisplayName: "No",
						ColStart:    0,
						ColWidth:    10,
						Align:       "left",
					},
					{
						ID:          "col_tanggal",
						FieldKey:    "tanggal",
						DisplayName: "Tanggal",
						ColStart:    11,
						ColWidth:    55,
						Align:       "left",
					},
					{
						ID:          "col_sandi",
						FieldKey:    "kode_transaksi",
						DisplayName: "Kode/Sandi",
						ColStart:    70,
						ColWidth:    40,
						Align:       "center",
					},
					{
						ID:          "col_debet",
						FieldKey:    "debet",
						DisplayName: "Debet",
						ColStart:    115,
						ColWidth:    85,
						Align:       "right",
					},
					{
						ID:          "col_kredit",
						FieldKey:    "kredit",
						DisplayName: "Kredit",
						ColStart:    205,
						ColWidth:    75,
						Align:       "right",
					},
					{
						ID:          "col_saldo",
						FieldKey:    "saldo_sesudah",
						DisplayName: "Saldo",
						ColStart:    285,
						ColWidth:    80,
						Align:       "right",
					},
					{
						ID:          "col_paraf",
						FieldKey:    "petugas",
						DisplayName: "Ket/Paraf",
						ColStart:    370,
						ColWidth:    45,
						Align:       "center",
					},
				},
			},
		},
	}
}

// GetActiveTemplate mengembalikan template yang sedang aktif, atau template pertama jika tidak ditemukan.
func (lc *LayoutConfig) GetActiveTemplate() *LayoutTemplate {
	for i := range lc.Templates {
		if lc.Templates[i].ID == lc.ActiveTemplate {
			return &lc.Templates[i]
		}
	}
	if len(lc.Templates) > 0 {
		return &lc.Templates[0]
	}
	return nil
}

// LoadLayoutConfig membaca layouts.json, jika tidak ada mengembalikan default.
func LoadLayoutConfig(path string) (*LayoutConfig, error) {
	cfg := DefaultLayoutConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("gagal membaca file layout %s: %w", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("format layout json tidak valid: %w", err)
	}

	if cfg.ActiveTemplate == "" && len(cfg.Templates) > 0 {
		cfg.ActiveTemplate = cfg.Templates[0].ID
	}

	return cfg, nil
}

// SaveLayoutConfig menyimpan konfigurasi layout ke file JSON.
func SaveLayoutConfig(path string, cfg *LayoutConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("gagal serialisasi layout ke JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
