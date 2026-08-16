#define MyAppName "My Compression"
#ifndef MyAppVersion
  #define MyAppVersion "0.1.0"
#endif

[Setup]
AppId={{8F3A1B2C-4D5E-6789-ABCD-EF0123456789}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
DefaultDirName={autopf}\My Compression
DefaultGroupName={#MyAppName}
OutputDir=..\..\dist
OutputBaseFilename=MyCompression-Setup-windows-amd64
SetupIconFile=..\icon.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "..\..\dist\my-compression.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\icon.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\my-compression.exe"; IconFilename: "{app}\icon.ico"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\my-compression.exe"; IconFilename: "{app}\icon.ico"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional icons:"

[Run]
Filename: "{app}\my-compression.exe"; Description: "Launch My Compression"; Flags: nowait postinstall skipifsilent