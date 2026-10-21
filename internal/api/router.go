package api

import (
	"fmt"
	"net/http"
)

// SetupRouter mengonfigurasi routing HTTP beserta seluruh middleware yang dibutuhkan
func SetupRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// REST API Endpoints sesuai SRS
	mux.HandleFunc("/ping", h.HandlePing)
	mux.HandleFunc("/print", h.HandlePrint)
	mux.HandleFunc("/eject", h.HandleEject)

	// Root API Info Endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"app":"Passbook Bridge","version":"%s","status":"active"}`, AppVersion)
	})

	// Terapkan middleware CORS dan Logging secara bertingkat
	return CORSMiddleware(LoggingMiddleware(mux))
}
