package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"passbook-bridge/internal/config"
	"passbook-bridge/internal/printer"
)

func TestHandleEject(t *testing.T) {
	cfg := config.DefaultConfig()
	mockPrinter := printer.NewMockPrinter("Mock Test")
	handler := NewHandler(mockPrinter, cfg)

	req := httptest.NewRequest(http.MethodPost, "/eject", nil)
	rr := httptest.NewRecorder()

	router := SetupRouter(handler)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var res struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false")
	}
}
