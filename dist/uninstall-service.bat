@echo off
echo ====================================================
echo   Mencopot Passbook Bridge Windows Service
echo ====================================================
echo.

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Harap jalankan script ini sebagai Administrator!
    echo Klik kanan file ini lalu pilih "Run as administrator".
    echo.
    pause
    exit /b 1
)

echo Menghentikan service...
sc stop PassbookBridge
echo Menghapus pendaftaran service...
sc delete PassbookBridge

echo.
echo Service berhasil dicopot dari Windows.
pause
