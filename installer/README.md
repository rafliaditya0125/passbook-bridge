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
Jika Anda menggunakan GitHub Actions, Inno Setup bisa dijalankan otomatis pada runner Windows dengan menambahkan step:
```yaml
- name: Build Installer
  run: iscc installer/passbook-bridge-setup.iss
```
