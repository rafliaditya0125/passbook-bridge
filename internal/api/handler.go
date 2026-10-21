package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/escp"
	"passbook-bridge/internal/model"
	"passbook-bridge/internal/printer"
)

const AppVersion = "2.4.0"

// Handler menangani endpoint HTTP API Passbook Bridge
type Handler struct {
	printer       printer.Printer
	cfg           *config.Config
	layouts       *config.LayoutConfig
	configPath    string
	layoutsPath   string
	startTime     time.Time
	totalPrints   int64
	lastPrintTime string
	mu            sync.Mutex
	logMu         sync.Mutex
	logEntries    []string
}

// NewHandler membuat instance Handler
func NewHandler(p printer.Printer, cfg *config.Config) *Handler {
	layouts, err := config.LoadLayoutConfig("layouts.json")
	if err != nil {
		layouts = config.DefaultLayoutConfig()
	}

	h := &Handler{
		printer:       p,
		cfg:           cfg,
		layouts:       layouts,
		configPath:    "config.json",
		layoutsPath:   "layouts.json",
		startTime:     time.Now(),
		totalPrints:   0,
		lastPrintTime: "-",
		logEntries:    make([]string, 0),
	}
	h.AddLog("Passbook Bridge Daemon initialized.")
	return h
}

// SetLayoutConfig mengatur konfigurasi layout eksplisit
func (h *Handler) SetLayoutConfig(l *config.LayoutConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.layouts = l
}

// SetPaths mengatur path file konfigurasi
func (h *Handler) SetPaths(configPath, layoutsPath string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if configPath != "" {
		h.configPath = configPath
	}
	if layoutsPath != "" {
		h.layoutsPath = layoutsPath
	}
}

// AddLog mencatat log ke in-memory ring buffer
func (h *Handler) AddLog(msg string) {
	h.logMu.Lock()
	defer h.logMu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, msg)
	h.logEntries = append(h.logEntries, entry)
	if len(h.logEntries) > 200 {
		h.logEntries = h.logEntries[len(h.logEntries)-200:]
	}
}

// writeJSON adalah helper untuk mengirim respon JSON
func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Gagal menulis respon JSON", "error", err)
	}
}

// HandlePing menangani endpoint GET /ping untuk health check dan status printer
func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Success: false,
			Message: "Method tidak diizinkan, gunakan GET",
		})
		return
	}

	connected, err := h.printer.Ping()
	if err != nil {
		slog.Warn("Pengecekan printer ping menghasilkan peringatan", "error", err)
	}

	resp := model.PingResponse{
		Status:           "online",
		PrinterConnected: connected,
		PrinterName:      h.printer.Name(),
		Version:          AppVersion,
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandlePrint menangani endpoint POST /print untuk mencetak baris transaksi ke buku tabungan
func (h *Handler) HandlePrint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Success: false,
			Message: "Method tidak diizinkan, gunakan POST",
		})
		return
	}

	// Batasi ukuran request body maksimal 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var req model.PrintRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: "Gagal membaca body request: " + err.Error(),
		})
		return
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: "Format JSON tidak valid: " + err.Error(),
		})
		return
	}

	// Validasi input
	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Buat perintah byte ESC/P dengan template aktif
	opts := escp.GeneratorOptions{
		EnableEject: h.cfg.EnableEject,
		Template:    h.layouts.GetActiveTemplate(),
	}
	rawBytes := escp.GeneratePrintCommand(&req, opts)

	// Kirim data ke hardware printer
	if err := h.printer.Print(rawBytes); err != nil {
		slog.Error("Gagal mencetak transaksi ke printer", "error", err)

		errMsg := err.Error()
		statusCode := http.StatusInternalServerError

		lowerErr := strings.ToLower(errMsg)
		if strings.Contains(lowerErr, "tidak terdeteksi") ||
			strings.Contains(lowerErr, "offline") ||
			strings.Contains(lowerErr, "tidak ditemukan") ||
			strings.Contains(lowerErr, "tidak terhubung") {
			statusCode = http.StatusServiceUnavailable
		}

		writeJSON(w, statusCode, model.ErrorResponse{
			Success: false,
			Message: errMsg,
		})
		return
	}

	h.mu.Lock()
	h.totalPrints++
	h.lastPrintTime = time.Now().Format("15:04:05")
	h.mu.Unlock()
	h.AddLog(fmt.Sprintf("Print OK: %s %s (Rp %v)", req.KodeTransaksi, req.JenisTransaksi, req.Jumlah))

	writeJSON(w, http.StatusOK, model.PrintResponse{
		Success: true,
		Message: "Baris transaksi berhasil dicetak ke buku tabungan",
	})
}

// HandleEject menangani endpoint POST /eject untuk mengeluarkan buku tabungan dari printer
func (h *Handler) HandleEject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Success: false,
			Message: "Method tidak diizinkan, gunakan POST",
		})
		return
	}

	err := h.printer.Print(escp.CmdFormFeedEject)
	if err != nil {
		slog.Error("Gagal mengeluarkan buku tabungan", "error", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Message: "Gagal mengeluarkan buku tabungan: " + err.Error(),
		})
		return
	}

	h.AddLog("Perintah Eject berhasil dikirim ke printer.")
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Buku tabungan berhasil dikeluarkan",
	})
}
