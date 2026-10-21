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
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	MaxColumns int         `json:"max_columns"`
	TopMargin  int         `json:"top_margin,omitempty"`  // offset baris/margin atas (baris ke-n)
	LeftMargin int         `json:"left_margin,omitempty"` // offset kolom/margin kiri
	Columns    []ColumnDef `json:"columns"`
}

// LayoutConfig menyimpan seluruh template dan template aktif yang dipilih.
type LayoutConfig struct {
	ActiveTemplate string           `json:"active_template"`
	Templates      []LayoutTemplate `json:"templates"`
}

// DefaultLayoutConfig menghasilkan konfigurasi layout standar passbook (67-80 kolom).
func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		ActiveTemplate: "default",
		Templates: []LayoutTemplate{
			{
				ID:         "default",
				Name:       "Buku Tabungan Standar",
				MaxColumns: 80,
				Columns: []ColumnDef{
					{
						ID:          "col_tanggal",
						FieldKey:    "tanggal",
						DisplayName: "Tanggal",
						ColStart:    0,
						ColWidth:    9,
						Align:       "left",
					},
					{
						ID:          "col_sandi",
						FieldKey:    "kode_transaksi",
						DisplayName: "Sandi",
						ColStart:    9,
						ColWidth:    5,
						Align:       "left",
					},
					{
						ID:          "col_debet",
						FieldKey:    "debet",
						DisplayName: "Debet",
						ColStart:    14,
						ColWidth:    13,
						Align:       "right",
					},
					{
						ID:          "col_kredit",
						FieldKey:    "kredit",
						DisplayName: "Kredit",
						ColStart:    27,
						ColWidth:    13,
						Align:       "right",
					},
					{
						ID:          "col_saldo",
						FieldKey:    "saldo_sesudah",
						DisplayName: "Saldo",
						ColStart:    40,
						ColWidth:    15,
						Align:       "right",
					},
					{
						ID:          "col_petugas",
						FieldKey:    "petugas",
						DisplayName: "Validasi / Teller",
						ColStart:    55,
						ColWidth:    12,
						Align:       "left",
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
