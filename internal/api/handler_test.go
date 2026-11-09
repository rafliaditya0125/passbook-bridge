package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/model"
	"passbook-bridge/internal/printer"
)

func setupTestServer() (*Handler, http.Handler, *printer.MockPrinter) {
	cfg := config.DefaultConfig()
	mockP := printer.NewMockPrinter("EPSON PLQ-20")
	h := NewHandler(mockP, cfg)
	router := SetupRouter(h)
	return h, router, mockP
}

func TestHandlePing(t *testing.T) {
	_, router, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status code harus 200, dapat: %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Header CORS Access-Control-Allow-Origin tidak ada atau tidak valid")
	}

	var resp model.PingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Gagal unmarshal respon ping: %v", err)
	}

	if resp.Status != "online" || !resp.PrinterConnected || resp.PrinterName != "EPSON PLQ-20" {
		t.Errorf("Isi respon ping tidak sesuai: %+v", resp)
	}
}

func TestOptionsPreflight(t *testing.T) {
	_, router, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodOptions, "/print", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Options request harus mengembalikan 204 No Content, dapat: %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Header CORS tidak ditemukan pada preflight")
	}
}

func TestHandlePrint_Success(t *testing.T) {
	_, router, _ := setupTestServer()

	payload := `{
		"tanggal": "2026-10-20 13:45:00",
		"kode_transaksi": "BKM001/10/26",
		"jenis_transaksi": "setor",
		"jumlah": 100000,
		"saldo_sesudah": 250000,
		"keterangan": "BKM001/10/26",
		"petugas": "SRI Y"
	}`

	req := httptest.NewRequest(http.MethodPost, "/print", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status code harus 200, dapat: %d, body: %s", rec.Code, rec.Body.String())
	}

	var resp model.PrintResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Gagal decode JSON respon: %v", err)
	}

	if !resp.Success {
		t.Errorf("Success harus bernilai true, dapat false: %s", resp.Message)
	}
}

func TestHandlePrint_InvalidPayload(t *testing.T) {
	_, router, _ := setupTestServer()

	// Missing kode_transaksi dan jenis_transaksi
	payload := `{"tanggal": "2026-10-20", "jumlah": 50000}`

	req := httptest.NewRequest(http.MethodPost, "/print", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Harusnya mengembalikan 400 Bad Request, dapat: %d", rec.Code)
	}

	var resp model.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Gagal decode error response: %v", err)
	}

	if resp.Success != false {
		t.Errorf("Success harus bernilai false")
	}
}

func TestHandlePrint_PrinterOffline(t *testing.T) {
	_, router, mockP := setupTestServer()
	mockP.SetConnected(false)

	payload := `{
		"tanggal": "2026-10-20 13:45:00",
		"kode_transaksi": "BKM001/10/26",
		"jenis_transaksi": "setor",
		"jumlah": 100000,
		"saldo_sesudah": 250000
	}`

	req := httptest.NewRequest(http.MethodPost, "/print", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Harusnya mengembalikan 503 Service Unavailable, dapat: %d", rec.Code)
	}
}

func TestHandlePrint_ServerPayloadWithBarisAliases(t *testing.T) {
	_, router, mockP := setupTestServer()

	payload := `{
      "tanggal": "08/11/2026 14:00:00",
      "kode_transaksi": "BKM-0001",
      "jenis_transaksi": "setor",
      "jumlah": 50000,
      "saldo_sesudah": 250000,
      "keterangan": "BKM-0001",
      "petugas": "Teller 1",
      "baris": 5,
      "line": 5,
      "row": 5
    }`

	req := httptest.NewRequest(http.MethodPost, "/print", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Harusnya mengembalikan 200 OK, dapat: %d, body: %s", rec.Code, rec.Body.String())
	}

	lastPrint := mockP.GetLastPrinted()
	if len(lastPrint) == 0 {
		t.Fatalf("MockPrinter tidak menerima bytes cetak")
	}

	// Baris 5 artinya offset 4 baris baru (CR LF x 4)
	crlfCount := bytes.Count(lastPrint, []byte{0x0D, 0x0A})
	if crlfCount < 4 {
		t.Errorf("Expected at least 4 CRLFs for line skipping to line 5, got %d", crlfCount)
	}

	// Cek teks baris tercetak mengandung tanggal 08/11/26 dan BKM-0001
	printStr := string(lastPrint)
	if !strings.Contains(printStr, "08/11/26") {
		t.Errorf("Hasil cetak harus memformat tanggal 08/11/2026 menjadi 08/11/26, got: %q", printStr)
	}
	if !strings.Contains(printStr, "BKM-00") {
		t.Errorf("Hasil cetak harus memuat kode transaksi BKM-00, got: %q", printStr)
	}
}
