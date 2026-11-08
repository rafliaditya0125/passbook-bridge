//go:build windows

package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sys/windows/svc"
)

type bridgeService struct {
	server *http.Server
}

func (s *bridgeService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	go func() {
		slog.Info("Passbook Bridge HTTP Server aktif di background Windows Service", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server HTTP Windows Service berhenti tidak normal", "error", err)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for req := range r {
		switch req.Cmd {
		case svc.Interrogate:
			changes <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = s.server.Shutdown(ctx)
			cancel()
			return
		default:
		}
	}
	return
}

func isRunningAsService() bool {
	isService, err := svc.IsWindowsService()
	return err == nil && isService
}

func runAsService(server *http.Server) error {
	return svc.Run("PassbookBridge", &bridgeService{server: server})
}
