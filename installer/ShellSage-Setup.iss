; =====================================================================
; ShellSage Inno Setup Installer Script
; Author: Mahmud Rahman
; Project: ShellSage (Autonomous AI Terminal Assistant & Developer Platform)
; License: MIT License
; =====================================================================

#ifndef MyAppVersion
#define MyAppVersion "4.1.0"
#endif

#define MyAppName "ShellSage"
#define MyAppPublisher "Mahmud Rahman"
#define MyAppURL "https://github.com/mahmud-r-farhan/ShellSage"
#define MyAppExeName "shellsage.exe"

#ifndef SourceExe
#define SourceExe "..\shellsage.exe"
#endif

[Setup]
; Unique application GUID
AppId={{5E973F1C-3A4B-4E65-B624-91F2478DC409}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} v{#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases

; Default Installation Directory
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes

; MIT License agreement displayed during installation
LicenseFile=..\LICENSE

; Detailed information documents shown to user during installation
InfoBeforeFile=installer_info.txt
InfoAfterFile=install_complete.txt

; Visual styling and icons
SetupIconFile=shellsage.ico
UninstallDisplayIcon={app}\{#MyAppExeName}

; Build outputs
OutputDir=..\dist
OutputBaseFilename=ShellSage-Setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ChangesEnvironment=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
; Desktop checkmark option to install the desktop shortcut
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"
; Option to integrate with system PATH environment variable
Name: "envpath"; Description: "Add ShellSage to system &PATH environment variable"; GroupDescription: "System Integration:"

[Files]
; Main CLI binary executable
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "{#MyAppExeName}"; Flags: ignoreversion
; MIT License
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
; Documentation & Information
Source: "..\readme.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "installer_info.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "install_complete.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "shellsage.ico"; DestDir: "{app}"; Flags: ignoreversion
; Helper scripts
Source: "..\scripts\ShellSage.ps1"; DestDir: "{app}\scripts"; Flags: ignoreversion

[Icons]
; Start Menu shortcuts
Name: "{autoprograms}\{#MyAppName}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\shellsage.ico"; WorkingDir: "{userdocs}"; Comment: "Launch ShellSage Autonomous AI Terminal Assistant"
Name: "{autoprograms}\{#MyAppName}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
; Desktop shortcut (controlled by desktopicon checkmark task)
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon; IconFilename: "{app}\shellsage.ico"; WorkingDir: "{userdocs}"; Comment: "Launch ShellSage Autonomous AI Terminal Assistant"

[Run]
; Finish installation action: prompt to launch ShellSage CLI
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
// Environment PATH modification support
const
  UserEnvironmentKey = 'Environment';
  SystemEnvironmentKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment';

procedure AddPathToEnv(PathToAdd: string);
var
  Paths: string;
  RootKey: Integer;
  SubKey: string;
begin
  if IsAdminInstallMode then
  begin
    RootKey := HKEY_LOCAL_MACHINE;
    SubKey := SystemEnvironmentKey;
  end
  else
  begin
    RootKey := HKEY_CURRENT_USER;
    SubKey := UserEnvironmentKey;
  end;

  if RegQueryStringValue(RootKey, SubKey, 'Path', Paths) then
  begin
    if Pos(';' + Uppercase(PathToAdd) + ';', ';' + Uppercase(Paths) + ';') = 0 then
    begin
      if (Length(Paths) > 0) and (Paths[Length(Paths)] <> ';') then
        Paths := Paths + ';';
      Paths := Paths + PathToAdd;
      RegWriteStringValue(RootKey, SubKey, 'Path', Paths);
    end;
  end
  else
  begin
    RegWriteStringValue(RootKey, SubKey, 'Path', PathToAdd);
  end;
end;

procedure RemovePathFromEnv(PathToRemove: string);
var
  Paths, NewPaths, Part: string;
  P, RootKey: Integer;
  SubKey: string;
begin
  if IsAdminInstallMode then
  begin
    RootKey := HKEY_LOCAL_MACHINE;
    SubKey := SystemEnvironmentKey;
  end
  else
  begin
    RootKey := HKEY_CURRENT_USER;
    SubKey := UserEnvironmentKey;
  end;

  if RegQueryStringValue(RootKey, SubKey, 'Path', Paths) then
  begin
    NewPaths := '';
    while Length(Paths) > 0 do
    begin
      P := Pos(';', Paths);
      if P = 0 then
      begin
        Part := Paths;
        Paths := '';
      end
      else
      begin
        Part := Copy(Paths, 1, P - 1);
        Paths := Copy(Paths, P + 1, Length(Paths) - P);
      end;
      if (Uppercase(Trim(Part)) <> Uppercase(Trim(PathToRemove))) and (Trim(Part) <> '') then
      begin
        if NewPaths <> '' then
          NewPaths := NewPaths + ';';
        NewPaths := NewPaths + Part;
      end;
    end;
    RegWriteStringValue(RootKey, SubKey, 'Path', NewPaths);
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    if WizardIsTaskSelected('envpath') then
    begin
      AddPathToEnv(ExpandConstant('{app}'));
    end;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
  begin
    RemovePathFromEnv(ExpandConstant('{app}'));
  end;
end;
