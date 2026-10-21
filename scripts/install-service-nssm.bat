@echo off
:: Script untuk mendaftarkan Passbook Bridge sebagai Windows Service menggunakan NSSM
:: Jalankan script ini sebagai Administrator (Run as Administrator)

set SERVICE_NAME=PassbookBridge
set APP_DIR=%~dp0..\dist
set APP_EXE=%APP_DIR%\passbook-bridge.exe

echo ========================================================
echo Mendaftarkan %SERVICE_NAME% ke Windows Services...
echo Direktori Aplikasi: %APP_DIR%
echo Executable: %APP_EXE%
echo ========================================================

where nssm >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] NSSM (Non-Sucking Service Manager) tidak ditemukan di PATH.
    echo Silakan unduh nssm.exe dari https://nssm.cc dan letakkan di PATH atau folder ini.
    pause
    exit /b 1
)

nssm stop %SERVICE_NAME% >nul 2>&1
nssm remove %SERVICE_NAME% confirm >nul 2>&1

nssm install %SERVICE_NAME% "%APP_EXE%"
nssm set %SERVICE_NAME% AppDirectory "%APP_DIR%"
nssm set %SERVICE_NAME% Description "Daemon Passbook Bridge Lokal untuk Printer Buku Tabungan Dot-Matrix"
nssm set %SERVICE_NAME% Start SERVICE_AUTO_START
nssm start %SERVICE_NAME%

echo.
echo Layanan %SERVICE_NAME% berhasil didaftarkan dan dijalankan!
echo Cek status dengan membuka http://localhost:<port>/ping di browser (sesuai config.json).
pause
