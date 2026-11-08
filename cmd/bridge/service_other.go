//go:build !windows

package main

import "net/http"

func isRunningAsService() bool {
	return false
}

func runAsService(server *http.Server) error {
	return nil
}
