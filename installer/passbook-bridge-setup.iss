; =====================================================================
; Inno Setup Script - Passbook Bridge & Controller Designer
; =====================================================================

#define MyAppName "Passbook Bridge"
#define MyAppVersion "0.1.0"
#define MyAppPublisher "Rafli Aditya"
#define MyAppExeName "passbook-bridge-gui.exe"
#define MyServiceExeName "passbook-bridge.exe"

[Setup]
; ID unik aplikasi untuk registrasi uninstaller
AppId={{9B7E3E71-4A59-4D78-9B21-88E6D38B0F2A}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
; Hak akses administrator dibutuhkan untuk memasang Windows Service & akses port COM/USB
PrivilegesRequired=admin
OutputDir=..\dist\installer
OutputBaseFilename=PassbookBridge-Setup-v{#MyAppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "indonesian"; MessagesFile: "compiler:Languages\Indonesian.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "installservice"; Description: "Pasang dan jalankan background service otomatis saat Windows menyala"; GroupDescription: "Konfigurasi Layanan:"

[Files]
; File binary utama
Source: "..\dist\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\{#MyServiceExeName}"; DestDir: "{app}"; Flags: ignoreversion

; File konfigurasi default (tidak menimpa jika sudah ada modifikasi oleh user)
Source: "..\dist\config.json"; DestDir: "{app}"; Flags: onlyifdoesntexist uninsneveruninstall
Source: "..\dist\layouts.json"; DestDir: "{app}"; Flags: onlyifdoesntexist uninsneveruninstall

[Icons]
; Shortcut GUI Controller di Start Menu dan Desktop
Name: "{group}\{#MyAppName} Controller"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName} Controller"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Jika user memilih task pasang service, daftarkan ke Windows Service Manager (services.msc)
Filename: "sc.exe"; Parameters: "create PassbookBridge binPath= ""{app}\{#MyServiceExeName}"" start= auto DisplayName= ""Passbook Bridge Service"""; Flags: runhidden; Tasks: installservice
Filename: "sc.exe"; Parameters: "description PassbookBridge ""Layanan latar belakang perantara cetak buku tabungan dot-matrix untuk Core Banking."""; Flags: runhidden; Tasks: installservice
Filename: "sc.exe"; Parameters: "start PassbookBridge"; Flags: runhidden; Tasks: installservice

; Opsi untuk langsung menjalankan GUI Pengaturan setelah instalasi selesai
Filename: "{app}\{#MyAppExeName}"; Description: "Buka {#MyAppName} Controller GUI sekarang"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; Hentikan dan hapus Windows Service saat aplikasi di-uninstall
Filename: "sc.exe"; Parameters: "stop PassbookBridge"; Flags: runhidden
Filename: "sc.exe"; Parameters: "delete PassbookBridge"; Flags: runhidden
