@echo off
echo ==========================================================
echo   Building Passbook Bridge (Windows Release)
echo ==========================================================

if not exist dist mkdir dist

echo [1/3] Building Background Service (passbook-bridge.exe)...
go build -ldflags="-s -w" -o dist\passbook-bridge.exe .\cmd\bridge

echo [2/3] Building GUI Controller (passbook-bridge-gui.exe)...
go build -ldflags="-H windowsgui -s -w" -o dist\passbook-bridge-gui.exe .\cmd\gui

echo [3/3] Copying Default Configs...
if not exist dist\config.json copy config.json.example dist\config.json
if not exist dist\layouts.json copy layouts.json dist\layouts.json

echo.
echo ==========================================================
echo   Build Binaries Selesai!
echo ==========================================================
echo.

:: Cek apakah Inno Setup Compiler terpasang
set "ISCC="
if exist "%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe" set "ISCC=%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe"
if exist "%ProgramFiles%\Inno Setup 6\ISCC.exe" set "ISCC=%ProgramFiles%\Inno Setup 6\ISCC.exe"

if defined ISCC (
    echo Menemukan Inno Setup Compiler di: "%ISCC%"
    echo Mengompilasi installer Windows otomatis...
    "%ISCC%" installer\passbook-bridge-setup.iss
    echo.
    echo ==========================================================
    echo   Installer selesai dibuat di: dist\installer\
    echo ==========================================================
) else (
    echo [INFO] Inno Setup 6 belum terdeteksi di lokasi standar.
    echo Untuk membuat file installer (.exe):
    echo 1. Unduh Inno Setup 6 dari https://jrsoftware.org/isdl.php (gratis).
    echo 2. Klik kanan file "installer\passbook-bridge-setup.iss" lalu pilih "Compile".
)

pause
