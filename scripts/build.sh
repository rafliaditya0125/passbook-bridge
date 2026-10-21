#!/usr/bin/env bash
set -e

echo "Building Passbook Bridge..."
mkdir -p dist

echo "[1/3] Building Linux Service binary..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/passbook-bridge ./cmd/bridge

echo "[2/3] Building Windows Service binary (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/passbook-bridge.exe ./cmd/bridge

echo "[3/3] Building Desktop GUI (Fyne)..."
go build -ldflags="-s -w" -o dist/passbook-bridge-gui ./cmd/gui

cp config.json.example dist/config.json
cp layouts.json dist/layouts.json

echo ""
echo "=========================================================="
echo "  Build selesai! File tersedia di folder dist/:"
echo "=========================================================="
ls -lh dist/
echo ""
echo "Untuk membuat Windows Installer (.exe) menggunakan Inno Setup:"
echo "1. Pastikan passbook-bridge-gui.exe sudah di-build di Windows:"
echo "   go build -ldflags=\"-H windowsgui -s -w\" -o dist/passbook-bridge-gui.exe ./cmd/gui"
echo "2. Buka dan Compile installer/passbook-bridge-setup.iss di Inno Setup 6."
echo "   Output installer akan otomatis tersimpan di dist/installer/."
echo "=========================================================="
