# Petunjuk Pembuatan Windows Installer (Passbook Bridge)

Installer dibuat menggunakan **Inno Setup**, standar industri untuk installer software Windows yang ringan, stabil, dan profesional.

---

## Isi Paket yang Dipasang Installer:
1. `passbook-bridge.exe` — Background Windows Service (Daemon REST API port sesuai `config.json`).
2. `passbook-bridge-gui.exe` — Desktop GUI Controller & Visual Layout Designer.
3. `config.json` & `layouts.json` — Konfigurasi server dan template buku tabungan default.
4. Pendaftaran otomatis Windows Service (`sc.exe create PassbookBridge ...`) agar bridge langsung aktif saat PC teller booting.
5. Pembuatan Shortcut Start Menu dan Desktop.
6. Uninstaller bersih di Windows Control Panel / Settings.

---

## Cara Membuat File Installer (`PassbookBridge-Setup-v1.0.0.exe`):

### Cara 1: Menggunakan Inno Setup GUI (di Komputer Windows)
1. Download dan instal [Inno Setup 6](https://jrsoftware.org/isdl.php) (gratis & open-source).
2. Pastikan file binary hasil build sudah tersedia di folder `dist/`:
   - `dist/passbook-bridge.exe`
   - `dist/passbook-bridge-gui.exe`
   - `dist/config.json`
   - `dist/layouts.json`
3. Klik kanan pada file `installer/passbook-bridge-setup.iss` lalu pilih **Compile**.
4. Hasil installer akan otomatis dibuat di folder `dist/installer/PassbookBridge-Setup-v1.0.0.exe`.

### Cara 2: Menggunakan Command Line (Terminal Windows)
```cmd
"C:\Program Files (x86)\Inno Setup 6\ISCC.exe" installer\passbook-bridge-setup.iss
```

### Cara 3: Melalui CI/CD (GitHub Actions)
Repository ini telah dilengkapi dengan workflow GitHub Actions otomatis di [`.github/workflows/build-installer.yml`](../.github/workflows/build-installer.yml).
Workflow ini akan:
1. Menyiapkan environment Windows (`windows-latest`), Go, dan MinGW (GCC untuk CGO/Fyne).
2. Menjalankan unit test (`go test ./...`).
3. Mengompilasi binary `passbook-bridge.exe` dan `passbook-bridge-gui.exe`, serta menyalin file konfigurasi default ke `dist/`.
4. Memasang Inno Setup 6 dan memastikan file lokalisasi `Indonesian.isl` siap digunakan.
5. Mengompilasi installer `.exe` menggunakan `ISCC.exe`.
6. Mengunggah file installer sebagai GitHub Artifact yang dapat diunduh langsung dari tab **Actions**.
7. Membuat GitHub Release otomatis jika di-trigger oleh push Git Tag (misal `v0.1.0`).

Workflow dapat dipicu melalui:
- Push atau Pull Request ke branch `main`.
- Push Git tag berformat `v*` (contoh: `git tag v0.1.0 && git push origin v0.1.0`).
- Manual trigger melalui menu **Actions** > **Build & Package Windows Installer** > **Run workflow** (dapat menentukan custom version).
