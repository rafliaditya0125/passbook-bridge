package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"passbook-bridge/internal/api"
	"passbook-bridge/internal/config"
	"passbook-bridge/internal/printer"
)

type AppState struct {
	mu             sync.Mutex
	Config         *config.Config
	Layouts        *config.LayoutConfig
	ConfigPath     string
	LayoutsPath    string
	Server         *http.Server
	Printer        printer.Printer
	IsRunning      bool
	LogEntries     []string
	OnLogUpdated   func(string)
	OnStateChanged func()
	Window         fyne.Window
}

func (s *AppState) NotifyStateChanged() {
	s.mu.Lock()
	callback := s.OnStateChanged
	s.mu.Unlock()

	if callback != nil {
		callback()
	}
}

var state = &AppState{
	ConfigPath:  "config.json",
	LayoutsPath: "layouts.json",
	LogEntries:  make([]string, 0),
}

func (s *AppState) Log(msg string) {
	s.mu.Lock()
	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, msg)
	s.LogEntries = append(s.LogEntries, entry)
	if len(s.LogEntries) > 200 {
		s.LogEntries = s.LogEntries[len(s.LogEntries)-200:]
	}
	callback := s.OnLogUpdated
	s.mu.Unlock()

	if callback != nil {
		callback(entry)
	}
}

func (s *AppState) StartServer() error {
	s.mu.Lock()
	if s.IsRunning {
		s.mu.Unlock()
		return fmt.Errorf("server bridge sudah berjalan")
	}

	p, err := printer.NewPrinter(s.Config)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("gagal menginisialisasi printer: %w", err)
	}
	s.Printer = p

	handler := api.NewHandler(p, s.Config)
	if s.Layouts != nil {
		handler.SetLayoutConfig(s.Layouts)
	}
	handler.SetPaths(s.ConfigPath, s.LayoutsPath)
	router := api.SetupRouter(handler)

	addr := fmt.Sprintf(":%d", s.Config.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	s.Server = server
	s.IsRunning = true
	s.mu.Unlock()

	s.NotifyStateChanged()

	s.Log(fmt.Sprintf("Memulai Passbook Bridge pada port %d (Mode: %s)...", s.Config.Port, s.Config.PrinterType))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.mu.Lock()
			s.IsRunning = false
			s.mu.Unlock()
			s.Log(fmt.Sprintf("Server berhenti dengan error: %v", err))
			s.NotifyStateChanged()
		}
	}()

	s.Log(fmt.Sprintf("Server Bridge aktif mendengarkan di http://127.0.0.1:%d", s.Config.Port))
	return nil
}

func (s *AppState) StopServer() error {
	s.mu.Lock()
	if !s.IsRunning || s.Server == nil {
		s.mu.Unlock()
		return fmt.Errorf("server bridge sedang tidak berjalan")
	}

	server := s.Server
	s.mu.Unlock()

	s.Log("Menghentikan server bridge...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)

	s.mu.Lock()
	if s.Printer != nil {
		_ = s.Printer.Close()
		s.Printer = nil
	}
	s.Server = nil
	s.IsRunning = false
	s.mu.Unlock()

	s.NotifyStateChanged()
	s.Log("Server bridge berhasil dihentikan.")
	return err
}
