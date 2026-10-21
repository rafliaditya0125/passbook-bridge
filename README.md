# Passbook Bridge Lokal (Go)

Aplikasi background daemon lokal untuk menjembatani sistem perbankan web (**Bank Mini**) dengan printer buku tabungan dot-matrix fisik (*passbook printer* seperti Epson PLQ-20 / PLQ-30, Olivetti PR2 Plus) melalui protokol HTTP lokal port `3001`.

Dibuat mengacu pada spesifikasi [PASSBOOK_BRIDGE_REQUIREMENTS.md](PASSBOOK_BRIDGE_REQUIREMENTS.md).

---

## Fitur Utama

- **Single Binary Executable**: Ukuran file ringkas, penggunaan RAM < 20 MB (jauh di bawah batas toleransi 60 MB).
- **Stitch Layout Manager & Dashboard UI**: Antarmuka visual web perbankan modern yang di-embed langsung ke binary (buka di `http://localhost:3001` atau via `./dist/passbook-bridge-gui`):
  - **Visual Layout Editor**: Kanvas buku tabungan interaktif, penggaris millimeter, watermark pola guilloche, penyesuaian batas kolom, dan live inspector koordinat printhead.
  - **Printer & Port Bridge**: Diagnostik hardware (Olivetti PR2+, Epson PLQ-20, Compuprint), pemilihan interface (Serial RS232 COM, USB Virtual, LPT, TCP 9100), ping hardware, eject buku, dan reset printhead.
  - **Windows Service & Daemon**: Manajemen background service (Start, Stop, Restart, Install, Uninstall), port listeners, dan live streaming log terminal.
  - **Pengaturan Sistem**: Konfigurasi webhook callback core banking, anti-fraud rules (masking rupiah `***Rp`), format tanggal, dan import/export template buku (BRI, BCA, Mandiri, BNI).
- **Multi-backend Driver**:
  - `spooler`: Menggunakan Win32 Raw Print Spooler (`winspool.drv`) untuk Windows (driver Generic / Text Only atau driver resmi Epson).
  - `serial`: Komunikasi langsung ke port COM / Serial (`go.bug.st/serial`).
  - `mock`: Driver simulasi untuk pengujian lokal tanpa printer fisik (output ke console & `simulated_passbook.log`).
- **ESC/P Layout Engine**:
  - Auto-format 6 kolom standar buku tabungan (Tanggal, Sandi, Debet, Kredit, Saldo, Validasi).
  - Mode Font Condensed (`0x0F` SI) dan inisialisasi ESC/P (`0x1B, 0x40`).
  - Dukungan baris ke-n (line skipping) opsional.
- **CORS Enabled**: Siap menerima request dari browser tanpa terblokir *Same-Origin Policy*.

---

## Struktur Direktori

```
passbook-bridge/
├── cmd/
│   └── bridge/
│       └── main.go                  # Entry point daemon
├── config.json                      # Konfigurasi aktif
├── config.json.example              # Template konfigurasi
├── dist/
│   ├── passbook-bridge              # Binary Linux (amd64)
│   ├── passbook-bridge.exe          # Binary Windows (amd64)
│   └── config.json
├── internal/
│   ├── api/
│   │   ├── handler.go               # Handler /ping dan /print
│   │   ├── handler_test.go          # Unit tests API
│   │   ├── middleware.go            # Middleware CORS & logging
│   │   └── router.go                # Routing setup
│   ├── config/
│   │   └── config.go                # Loader & validator konfigurasi
│   ├── escp/
│   │   ├── formatter.go             # Format 6 kolom (lebar 67 karakter)
│   │   ├── generator.go             # Generator byte command ESC/P
│   │   └── generator_test.go        # Unit test format dot-matrix
│   ├── model/
│   │   ├── request.go               # Struct payload print & validasi
│   │   └── response.go              # Struct respon standar
│   └── printer/
│       ├── mock.go                  # Mock printer untuk dev
│       ├── printer.go               # Interface printer & factory
│       ├── serial.go                # Driver Virtual COM/Serial
│       ├── spooler_windows.go       # Driver Windows Spooler Raw
│       └── spooler_other.go         # Fallback non-Windows
└── scripts/
    ├── build.sh                     # Script build Linux/macOS
    ├── build.bat                    # Script build Windows
    └── install-service-nssm.bat     # Installer Windows Service
```

---

## Cara Menjalankan

### 1. Menjalankan Binary Langsung

```bash
# Menggunakan file config.json default
./dist/passbook-bridge

# Atau pada Windows:
.\dist\passbook-bridge.exe
```

### 2. Konfigurasi (`config.json`)

```json
{
  "port": 3001,
  "printer_type": "spooler",
  "printer_name": "EPSON PLQ-20 ESC/P2",
  "serial_port": "COM3",
  "baud_rate": 9600,
  "log_level": "info",
  "enable_eject": false
}
```

- Untuk pengujian di komputer dev (tanpa printer): ganti `"printer_type": "mock"`.
- Untuk koneksi USB Serial / COM: ganti `"printer_type": "serial"`, dan sesuaikan `"serial_port": "COM3"`.
- Untuk Windows Spooler resmi: gunakan `"printer_type": "spooler"`, sesuaikan `"printer_name"` dengan nama printer di Windows *Control Panel / Printers*.

---

## Menjalankan Unit Test

```bash
go test -v ./...
```

---

## Kompilasi / Build Ulang

```bash
# Di Linux / macOS:
./scripts/build.sh

# Di Windows:
.\scripts\build.bat
```
