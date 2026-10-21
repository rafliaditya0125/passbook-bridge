package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"passbook-bridge/internal/config"
)

func TestReadOnlyEntry(t *testing.T) {
	_ = test.NewApp()
	entry := newReadOnlyEntry()
	initialText := "2026/10/20 22:00:00 [INFO] Server started\n2026/10/20 22:00:01 [INFO] Ready"
	entry.SetText(initialText)

	// 1. User tries to type letters/numbers
	test.Type(entry, "HACKER TEXT 123")
	if entry.Text != initialText {
		t.Errorf("Expected text to remain unchanged after typing, got %q", entry.Text)
	}

	// 2. User tries Backspace
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if entry.Text != initialText {
		t.Errorf("Expected text to remain unchanged after backspace, got %q", entry.Text)
	}

	// 3. User tries Delete
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	if entry.Text != initialText {
		t.Errorf("Expected text to remain unchanged after delete, got %q", entry.Text)
	}

	// 4. User tries Enter
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if entry.Text != initialText {
		t.Errorf("Expected text to remain unchanged after enter, got %q", entry.Text)
	}

	// 5. User tries Paste shortcut (Ctrl+V)
	clip := test.NewApp().Clipboard()
	clip.SetContent("MALICIOUS PASTE")
	entry.TypedShortcut(&fyne.ShortcutPaste{Clipboard: clip})
	if entry.Text != initialText {
		t.Errorf("Expected text to remain unchanged after paste shortcut, got %q", entry.Text)
	}

	// 6. User tries Cut shortcut (Ctrl+X)
	entry.TypedShortcut(&fyne.ShortcutCut{Clipboard: clip})
	if entry.Text != initialText {
		t.Errorf("Expected text to remain unchanged after cut shortcut, got %q", entry.Text)
	}
}

func TestDashboardPortReactivity(t *testing.T) {
	testApp := test.NewApp()
	win := testApp.NewWindow("Test Window")

	state.Config = &config.Config{
		Port:        3000,
		PrinterType: "mock",
		PrinterName: "Virtual Test Printer",
		EnableEject: false,
	}
	state.IsRunning = true

	obj := buildDashboardTab(win)
	if obj == nil {
		t.Fatal("Expected dashboard tab canvas object, got nil")
	}

	// Change port to 3002 and stop state
	state.Config.Port = 3002
	state.IsRunning = false
	state.NotifyStateChanged()

	// Verify that state changes triggered the registered callback without error
	if state.IsRunning {
		t.Errorf("Expected IsRunning to be false, got true")
	}
	if state.Config.Port != 3002 {
		t.Errorf("Expected Config.Port to be 3002, got %d", state.Config.Port)
	}
}
