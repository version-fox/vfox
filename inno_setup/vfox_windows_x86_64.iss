#include "environment.iss"

#define MyAppName "vfox"
#define MyAppVersion GetEnv("VFOX_VERSION")
#define MyAppPublisher "Han Li"
#define MyAppURL "https://github.com/version-fox/vfox"

[Setup]
AppId={#MyAppName}-fc742fc3-7013-49b7-adcb-96f2d6ddbda0
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DisableDirPage=yes
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputBaseFilename=vfox_{#MyAppVersion}_windows_setup_x86_64
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ChangesEnvironment=true
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "{#MyAppName}_{#MyAppVersion}_windows_x86_64/vfox.exe"; DestDir: "{app}"; Flags: ignoreversion

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
begin
    if CurStep = ssPostInstall
     then EnvAddPath(ExpandConstant('{app}'));
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
    if CurUninstallStep = usPostUninstall
    then EnvRemovePath(ExpandConstant('{app}'));
end;