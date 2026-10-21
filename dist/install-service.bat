@echo off
echo ====================================================
echo   Memasang Passbook Bridge sebagai Windows Service
echo ====================================================
echo.

:: Cek Administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Harap jalankan script ini sebagai Administrator!
    echo Klik kanan file ini lalu pilih "Run as administrator".
    echo.
    pause
    exit /b 1
)

set "CURR_DIR=%~dp0"
set "EXE_PATH=%CURR_DIR%passbook-bridge.exe"
set "CFG_PATH=%CURR_DIR%config.json"
set "LYT_PATH=%CURR_DIR%layouts.json"

echo Menghentikan service lama jika ada...
sc stop PassbookBridge >nul 2>&1
sc delete PassbookBridge >nul 2>&1

echo Mendaftarkan service baru ke Windows...
sc create PassbookBridge binPath= "\"%EXE_PATH%\" -config \"%CFG_PATH%\" -layouts \"%LYT_PATH%\"" start= auto DisplayName= "Passbook Bridge Service"
sc description PassbookBridge "Layanan latar belakang perantara cetak buku tabungan dot-matrix untuk Core Banking."

echo Menjalankan service...
sc start PassbookBridge

echo.
echo ====================================================
echo   Instalasi Selesai! Service aktif di background.
echo   Port HTTP dan setelan server berjalan sesuai config.json.
echo ====================================================
pause
