package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	ServiceName        = "passbook-bridge"
	WindowsServiceName = "PassbookBridge"
	ServiceDisplayName = "Passbook Bridge Dot-Matrix Service"
)

// ServiceStatus merepresentasikan status service di sistem operasi
type ServiceStatus string

const (
	StatusRunning      ServiceStatus = "Running (Berjalan)"
	StatusStopped      ServiceStatus = "Stopped (Berhenti)"
	StatusNotInstalled ServiceStatus = "Not Installed (Belum Terpasang)"
	StatusUnknown      ServiceStatus = "Status Tidak Diketahui"
)

func getPassword(pw []string) string {
	if len(pw) > 0 {
		return pw[0]
	}
	return ""
}

// runPrivilegedCommand mengeksekusi perintah sistem dengan sudo -S (Linux) atau direct/elevated (Windows)
func runPrivilegedCommand(password string, name string, args ...string) (string, error) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(name, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("%s (%w)", strings.TrimSpace(string(out)), err)
		}
		return string(out), nil
	}

	// Linux: jika password diberikan, jalankan via sudo -S agar membaca stdin langsung dari GUI (bukan terminal)
	if password != "" {
		fullArgs := append([]string{"-S", "-p", "", name}, args...)
		cmd := exec.Command("sudo", fullArgs...)
		cmd.Stdin = strings.NewReader(password + "\n")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("%s (%w)", strings.TrimSpace(string(out)), err)
		}
		return string(out), nil
	}

	// Coba sudo -n (jika sistem sudah mengizinkan tanpa password)
	cmdSudo := exec.Command("sudo", append([]string{"-n", name}, args...)...)
	outSudo, errSudo := cmdSudo.CombinedOutput()
	if errSudo == nil {
		return string(outSudo), nil
	}

	// Coba langsung (jika running sebagai root)
	cmdDirect := exec.Command(name, args...)
	outDirect, errDirect := cmdDirect.CombinedOutput()
	if errDirect == nil {
		return string(outDirect), nil
	}

	return string(outDirect), fmt.Errorf("membutuhkan otorisasi administrator/root: %s (%w)", strings.TrimSpace(string(outDirect)), errDirect)
}

// CheckStatus memeriksa status background service saat ini
func CheckStatus() (ServiceStatus, string) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("sc", "query", WindowsServiceName)
		out, err := cmd.CombinedOutput()
		output := string(out)
		if err != nil {
			if strings.Contains(output, "1060") || strings.Contains(output, "does not exist") {
				return StatusNotInstalled, "Service belum terpasang di Windows Service Manager."
			}
			return StatusNotInstalled, fmt.Sprintf("Service tidak ditemukan: %s", output)
		}
		if strings.Contains(output, "RUNNING") {
			return StatusRunning, "Service sedang aktif dan berjalan di background."
		} else if strings.Contains(output, "STOPPED") {
			return StatusStopped, "Service terpasang tetapi saat ini dalam keadaan berhenti."
		}
		return StatusUnknown, output
	}

	// Linux: periksa keberadaan file unit /etc/systemd/system/passbook-bridge.service
	serviceFilePath := "/etc/systemd/system/passbook-bridge.service"
	if _, err := os.Stat(serviceFilePath); os.IsNotExist(err) {
		return StatusNotInstalled, "Service belum terpasang di systemd (/etc/systemd/system/passbook-bridge.service)."
	}

	cmd := exec.Command("systemctl", "is-active", ServiceName)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err == nil && output == "active" {
		return StatusRunning, "Systemd service sedang aktif dan berjalan otomatis di background."
	}
	return StatusStopped, fmt.Sprintf("Systemd service terpasang (Status: %s).", output)
}

// InstallService otomatis membuat dan memasang background service ke OS (systemd di Linux, sc di Windows)
func InstallService(exePath string, password ...string) (string, error) {
	pwd := getPassword(password)

	absExe, err := filepath.Abs(exePath)
	if err != nil {
		absExe = exePath
	}

	dir := filepath.Dir(absExe)
	if runtime.GOOS == "windows" {
		daemonExe := filepath.Join(dir, "passbook-bridge.exe")
		if _, err := os.Stat(daemonExe); err == nil {
			absExe = daemonExe
		}
	} else {
		daemonExe := filepath.Join(dir, "passbook-bridge")
		if _, err := os.Stat(daemonExe); err == nil {
			absExe = daemonExe
		}
	}

	workDir, _ := os.Getwd()
	if absWd, err := filepath.Abs(workDir); err == nil {
		workDir = absWd
	}

	configPath := filepath.Join(workDir, "config.json")
	layoutsPath := filepath.Join(workDir, "layouts.json")

	if runtime.GOOS == "windows" {
		binPathArg := fmt.Sprintf("binPath= \"%s -config \"%s\" -layouts \"%s\"\"", absExe, configPath, layoutsPath)
		cmd := exec.Command("sc", "create", WindowsServiceName, binPathArg, "start=", "auto", fmt.Sprintf("DisplayName= \"%s\"", ServiceDisplayName))
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Coba elevated powershell jika permission denied
			psCmd := fmt.Sprintf("Start-Process sc.exe -ArgumentList 'create %s %s start= auto DisplayName= \"%s\"' -Verb RunAs", WindowsServiceName, binPathArg, ServiceDisplayName)
			psOut, psErr := exec.Command("powershell", "-Command", psCmd).CombinedOutput()
			if psErr != nil {
				return string(out), fmt.Errorf("gagal memasang service Windows: %s (%w)", string(out), err)
			}
			return string(psOut), nil
		}
		// Jalankan service setelah dibuat
		_ = exec.Command("sc", "start", WindowsServiceName).Run()
		return fmt.Sprintf("Service Windows '%s' berhasil dipasang dan diatur ke auto-start!", WindowsServiceName), nil
	}

	// Linux: Otomatis generate unit file systemd
	curUser := os.Getenv("USER")
	if curUser == "" {
		curUser = "root"
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=Passbook Bridge Dot-Matrix Service
After=network.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s
ExecStart=%s -config %s -layouts %s
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, curUser, workDir, absExe, configPath, layoutsPath)

	tmpServiceFile := filepath.Join(os.TempDir(), "passbook-bridge.service")
	if err := os.WriteFile(tmpServiceFile, []byte(serviceContent), 0644); err != nil {
		return "", fmt.Errorf("gagal membuat file temporary service: %w", err)
	}
	defer os.Remove(tmpServiceFile)

	targetServiceFile := "/etc/systemd/system/passbook-bridge.service"

	// 1. Salin unit file ke /etc/systemd/system/
	if out, err := runPrivilegedCommand(pwd, "cp", tmpServiceFile, targetServiceFile); err != nil {
		return out, fmt.Errorf("gagal menyalin unit service ke %s: %w", targetServiceFile, err)
	}

	// 2. Set permission 644
	_, _ = runPrivilegedCommand(pwd, "chmod", "644", targetServiceFile)

	// 3. systemctl daemon-reload
	if out, err := runPrivilegedCommand(pwd, "systemctl", "daemon-reload"); err != nil {
		return out, fmt.Errorf("gagal me-reload systemd daemon: %w", err)
	}

	// 4. systemctl enable passbook-bridge
	if out, err := runPrivilegedCommand(pwd, "systemctl", "enable", ServiceName); err != nil {
		return out, fmt.Errorf("gagal mengaktifkan auto-start systemd service: %w", err)
	}

	// 5. systemctl start passbook-bridge
	if out, err := runPrivilegedCommand(pwd, "systemctl", "start", ServiceName); err != nil {
		return out, fmt.Errorf("gagal memulai systemd service: %w", err)
	}

	return fmt.Sprintf("Service systemd berhasil dibuat di %s, diaktifkan, dan dijalankan!", targetServiceFile), nil
}

// UninstallService mencopot background service dari sistem
func UninstallService(password ...string) (string, error) {
	pwd := getPassword(password)

	if runtime.GOOS == "windows" {
		_ = exec.Command("sc", "stop", WindowsServiceName).Run()
		cmd := exec.Command("sc", "delete", WindowsServiceName)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("gagal menghapus service Windows: %s (%w)", string(out), err)
		}
		return "Service Windows berhasil dihapus dari sistem.", nil
	}

	// Linux: Stop, disable, remove file, daemon-reload
	_, _ = runPrivilegedCommand(pwd, "systemctl", "stop", ServiceName)
	_, _ = runPrivilegedCommand(pwd, "systemctl", "disable", ServiceName)
	targetServiceFile := "/etc/systemd/system/passbook-bridge.service"
	if out, err := runPrivilegedCommand(pwd, "rm", "-f", targetServiceFile); err != nil {
		return out, fmt.Errorf("gagal menghapus file %s: %w", targetServiceFile, err)
	}
	_, _ = runPrivilegedCommand(pwd, "systemctl", "daemon-reload")

	return "Background service systemd berhasil dihapus dari sistem.", nil
}

// StartService menyalakan background service
func StartService(password ...string) (string, error) {
	pwd := getPassword(password)

	if runtime.GOOS == "windows" {
		cmd := exec.Command("sc", "start", WindowsServiceName)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("gagal menyalakan service Windows: %s (%w)", string(out), err)
		}
		return string(out), nil
	}

	out, err := runPrivilegedCommand(pwd, "systemctl", "start", ServiceName)
	return out, err
}

// StopService mematikan background service
func StopService(password ...string) (string, error) {
	pwd := getPassword(password)

	if runtime.GOOS == "windows" {
		cmd := exec.Command("sc", "stop", WindowsServiceName)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("gagal menghentikan service Windows: %s (%w)", string(out), err)
		}
		return string(out), nil
	}

	out, err := runPrivilegedCommand(pwd, "systemctl", "stop", ServiceName)
	return out, err
}
