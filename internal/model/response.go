package model

// PingResponse merepresentasikan hasil pengecekan health check GET /ping
type PingResponse struct {
	Status           string `json:"status"`
	PrinterConnected bool   `json:"printer_connected"`
	PrinterName      string `json:"printer_name"`
	Version          string `json:"version"`
}

// PrintResponse merepresentasikan status cetak berhasil
type PrintResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ErrorResponse merepresentasikan format standar respon gagal
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
