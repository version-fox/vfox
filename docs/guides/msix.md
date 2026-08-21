# MSIX Bundle

The Windows assets on the [Releases](https://github.com/version-fox/vfox/releases) page include a `.msixbundle` package in addition to the setup `.exe`. MSIX is Microsoft's current application deployment format, intended to replace `.msi`/`.exe` installers. A bundle contains builds for `x86`, `x64` and `arm64`; Windows selects the matching architecture at install time.

## Requirements

- Windows 10 (1809+) or Windows Server 2025+
- Optional: [App Installer](https://learn.microsoft.com/windows/msix/app-installer/installing-apps-pkg) for double-click installation

## Installation

Download the latest `vfox_<version>_windows.msixbundle` from the [Releases](https://github.com/version-fox/vfox/releases) page and install it by double-click, or from PowerShell:

```powershell
Add-AppxPackage -Path .\vfox_<version>_windows.msixbundle
```

The package registers an app execution alias for `vfox.exe`, so the `vfox` command is available on PATH after installation without further configuration.

::: warning ⚠️ Release bundles are currently unsigned
Windows refuses to install unsigned packages. Re-sign the bundle with your own certificate before installing (see below).
:::

### Signing an unsigned bundle

The build cannot perform signing on behalf of users. At install time Windows verifies the signer against certificates trusted on the target machine, so a certificate generated during packaging is treated the same as no signature at all. In addition, MSIX requires an upgrade package to be signed with the same certificate as the installed one; per-build certificates would therefore break upgrades. Signing with a long-lived certificate of your own is a one-time operation: once the certificate is trusted, subsequent versions install without extra steps.

First create a self-signed code-signing certificate whose subject matches the package publisher (`CN=VersionFox`) and import it into the trusted store:

```powershell
$cert = New-SelfSignedCertificate -Type Custom -Subject "CN=VersionFox" `
    -KeyUsage DigitalSignature -CertStoreLocation "Cert:\CurrentUser\My" `
    -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3", "2.5.29.19={text}")
$password = ConvertTo-SecureString -String "pick-a-password" -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath vfox.pfx -Password $password | Out-Null
Import-PfxCertificate -FilePath vfox.pfx -CertStoreLocation Cert:\LocalMachine\TrustedPeople -Password $password
```

Then sign the bundle and install it ([signtool](https://learn.microsoft.com/windows/win32/seccrypto/signtool) comes with the Windows SDK):

```powershell
signtool sign /fd SHA256 /f vfox.pfx /p pick-a-password .\vfox_<version>_windows.msixbundle
Add-AppxPackage -Path .\vfox_<version>_windows.msixbundle
```

## Uninstallation

```powershell
Get-AppxPackage *vfox* | Remove-AppxPackage
```

Alternatively open **Settings** > **Apps** > **Installed apps**, select **vfox** and click **Uninstall**. Uninstalling removes the execution alias together with the package; the `~/.vfox` data directory is kept.

## Notes for maintainers

Release bundles are built by the `compile-msix` workflow on every release. Signing is optional; two options:

- **Self-signed certificate (no external account needed).** Generate one PFX and keep it stable across releases, then configure the repository secrets `MSIX_PFX_BASE64` (Base64-encoded certificate) and `MSIX_PFX_PASSWORD`; the subject must match the manifest publisher (`CN=VersionFox`). Releases are signed automatically from then on. Publish the public `.cer` file alongside the releases so users only need to import it once — because the signature stays identical across versions, upgrades are unaffected.
- **Publicly trusted certificate.** Packages signed by a CA-issued code-signing certificate or through [Azure Trusted Signing](https://learn.microsoft.com/azure/trusted-signing/overview) install without any user-side trust configuration. Obtaining one involves identity verification, and OV/EV certificates mandate hardware-protected private keys, which cannot be placed in CI secrets.

All packaging files reside in the [`packaging/msix/`](https://github.com/version-fox/vfox/tree/main/packaging/msix) directory:

| File | Purpose |
|------|---------|
| `AppxManifest.xml` | Manifest template; `@@VERSION@@`, `@@ARCHITECTURE@@` and `@@PUBLISHER@@` are substituted at build time. |
| `gen-assets.ps1` | Generates the tile icons from the repository logo (`logo.png`) via System.Drawing: transparent margins are cropped and the artwork is scaled onto a transparent canvas. PNG is required by AppX deployment (SVG is rejected), so these binaries are generated at build time instead of being committed. |
| `make-msix.ps1` | Builds `vfox` from source with Go, renders the manifest for each architecture, packs them with `MakeAppx.exe`, combines the packages into a `.msixbundle` and optionally signs it with `SignTool.exe`. |

Local packaging requires Windows, the [Windows SDK](https://developer.microsoft.com/windows/downloads/windows-sdk/) (provides `MakeAppx.exe` / `SignTool.exe`) and [Go](https://go.dev/dl/):

```powershell
./packaging/msix/make-msix.ps1 -Version 1.2.3
```

Artifacts are written to `packaging/msix/Output/`; `assets/`, `staging/` and `build/` are scratch directories and gitignored. Prerelease suffixes are normalized to four-part versions (`1.2.3-rc1` → `1.2.3.0`).
