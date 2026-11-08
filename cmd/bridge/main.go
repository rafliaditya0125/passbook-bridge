package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"passbook-bridge/internal/api"
	"passbook-bridge/internal/config"
	"passbook-bridge/internal/printer"
)

func main() {
	configPath := flag.String("config", "config.json", "Path file konfigurasi JSON")
	layoutsPath := flag.String("layouts", "layouts.json", "Path file layout buku tabungan")
	flag.Parse()

	// Resolusi path default jika tidak ditemukan di direktori saat ini (misal saat dijalankan sebagai Windows Service di System32)
	resolvedConfig := *configPath
	resolvedLayouts := *layoutsPath

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		if resolvedConfig == "config.json" {
			if _, err := os.Stat(resolvedConfig); os.IsNotExist(err) {
				candidate := filepath.Join(exeDir, "config.json")
				if _, err := os.Stat(candidate); err == nil {
					resolvedConfig = candidate
				}
			}
		}
		if resolvedLayouts == "layouts.json" {
			if _, err := os.Stat(resolvedLayouts); os.IsNotExist(err) {
				candidate := filepath.Join(exeDir, "layouts.json")
				if _, err := os.Stat(candidate); err == nil {
					resolvedLayouts = candidate
				}
			}
		}
	}

	// 1. Muat Konfigurasi
	cfg, err := config.LoadConfig(resolvedConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error memuat konfigurasi: %v\n", err)
		os.Exit(1)
	}

	layoutCfg, err := config.LoadLayoutConfig(resolvedLayouts)
	if err != nil {
		slog.Warn("Gagal memuat layout, menggunakan default", "error", err)
		layoutCfg = config.DefaultLayoutConfig()
	}

	// 2. Setup Logging
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("==================================================")
	slog.Info("  Passbook Bridge - Dot-Matrix Printer Daemon     ")
	slog.Info("==================================================",
		"version", api.AppVersion,
		"port", cfg.Port,
		"printer_type", cfg.PrinterType,
		"printer_name", cfg.PrinterName,
	)

	// 3. Inisialisasi Hardware Printer
	p, err := printer.NewPrinter(cfg)
	if err != nil {
		slog.Error("Gagal menginisialisasi printer", "error", err)
		os.Exit(1)
	}
	defer p.Close()

	// Periksa koneksi awal printer
	connected, pingErr := p.Ping()
	if pingErr != nil || !connected {
		slog.Warn("Peringatan: Printer belum terhubung atau port tidak aktif saat startup", "error", pingErr)
	} else {
		slog.Info("Status Printer: SIAP (Connected)", "printer", p.Name())
	}

	// 4. Setup Router & Server HTTP
	handler := api.NewHandler(p, cfg)
	handler.SetLayoutConfig(layoutCfg)
	handler.SetPaths(resolvedConfig, resolvedLayouts)
	router := api.SetupRouter(handler)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 5. Cek apakah berjalan di bawah Windows Service Control Manager (SCM)
	if isRunningAsService() {
		slog.Info("Passbook Bridge berjalan sebagai Windows Service resmi...", "service", "PassbookBridge")
		if err := runAsService(server); err != nil {
			slog.Error("Windows Service berhenti dengan error", "error", err)
			os.Exit(1)
		}
		slog.Info("Passbook Bridge Windows Service selesai ditutup.")
		return
	}

	// 6. Jalankan HTTP Daemon di goroutine terpisah (Mode Interaktif / Konsol / Systemd)
	go func() {
		slog.Info(fmt.Sprintf("Passbook Bridge aktif mendengarkan di http://127.0.0.1:%d", cfg.Port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server HTTP berhenti tidak normal", "error", err)
			os.Exit(1)
		}
	}()

	// 7. Graceful Shutdown (Mode Interaktif)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("Menerima sinyal shutdown, menghentikan server dengan bersih...", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Terjadi error saat shutdown server", "error", err)
	}

	slog.Info("Passbook Bridge selesai ditutup.")
}
