package service

import (
	"runtime"
	"testing"
)

func TestCheckStatus(t *testing.T) {
	st, msg := CheckStatus()
	t.Logf("OS: %s, ServiceStatus: %s, Message: %s", runtime.GOOS, st, msg)
	if st == "" {
		t.Errorf("CheckStatus returned empty status")
	}
}

func TestGetPassword(t *testing.T) {
	if getPassword(nil) != "" {
		t.Errorf("expected empty password for nil slice")
	}
	if getPassword([]string{}) != "" {
		t.Errorf("expected empty password for empty slice")
	}
	if getPassword([]string{"mysecret"}) != "mysecret" {
		t.Errorf("expected 'mysecret', got %s", getPassword([]string{"mysecret"}))
	}
}
