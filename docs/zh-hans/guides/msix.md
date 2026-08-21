# MSIX 安装包

[Releases](https://github.com/version-fox/vfox/releases) 页面的 Windows 资产中，除 setup `.exe` 安装器外还提供 `.msixbundle` 安装包。MSIX 是微软当前的应用部署格式，用于取代 `.msi`/`.exe` 安装器。一个 bundle 内含 `x86`、`x64`、`arm64` 三种架构的构建，安装时由 Windows 自动选择匹配的架构。

## 环境要求

- Windows 10（1809+）或 Windows Server 2025+
- 可选：[应用安装程序](https://learn.microsoft.com/windows/msix/app-installer/installing-apps-pkg)（用于双击安装）

## 安装

从 [Releases](https://github.com/version-fox/vfox/releases) 页面下载最新版本的 `vfox_<version>_windows.msixbundle`，双击安装，或在 PowerShell 中执行：

```powershell
Add-AppxPackage -Path .\vfox_<version>_windows.msixbundle
```

安装包会注册 `vfox.exe` 的应用执行别名，因此安装完成后 `vfox` 命令即可直接使用，无需手动配置 PATH。

::: warning ⚠️ 当前发布的安装包未签名
Windows 会拒绝安装未签名包。安装前请先使用自己的证书对安装包重新签名（方法见下文）。
:::

### 为未签名安装包签名

构建过程无法代替用户完成签名。Windows 在安装时校验签名证书是否属于目标机器信任的证书，打包阶段生成的临时证书与不签名效果相同；此外 MSIX 要求升级包与已安装包使用同一证书签名，每次构建使用不同证书会导致无法覆盖升级。因此应使用一张长期持有的证书完成签名：证书受信任后，后续版本均可直接安装。

首次操作需创建一张主题与包发布者（`CN=VersionFox`）一致的自签名代码签名证书，并导入本机受信任存储。导入 `Cert:\LocalMachine\TrustedPeople` 需要以管理员身份运行 PowerShell：

```powershell
$cert = New-SelfSignedCertificate -Type Custom -Subject "CN=VersionFox" `
    -KeyUsage DigitalSignature -CertStoreLocation "Cert:\CurrentUser\My" `
    -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3", "2.5.29.19={text}")
$password = ConvertTo-SecureString -String "pick-a-password" -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath vfox.pfx -Password $password | Out-Null
Import-PfxCertificate -FilePath vfox.pfx -CertStoreLocation Cert:\LocalMachine\TrustedPeople -Password $password
```

随后对安装包签名并安装（[signtool](https://learn.microsoft.com/windows/win32/seccrypto/signtool) 随 Windows SDK 提供）：

```powershell
signtool sign /fd SHA256 /f vfox.pfx /p pick-a-password .\vfox_<version>_windows.msixbundle
Add-AppxPackage -Path .\vfox_<version>_windows.msixbundle
```

## 卸载

```powershell
Get-AppxPackage *vfox* | Remove-AppxPackage
```

也可以在 **设置** > **应用** > **已安装的应用** 中选择 **vfox** 并点击 **卸载**。卸载会同时移除应用执行别名；`~/.vfox` 数据目录会被保留。

## 维护者须知

发布版 bundle 由 `compile-msix` 工作流在每次 release 时构建。签名可选，有两种方式：

- **自签证书（无需外部账号）**：生成一张 PFX 证书并在各次发布间保持不变，然后配置仓库 secrets：`MSIX_PFX_BASE64`（Base64 编码的证书文件）与 `MSIX_PFX_PASSWORD`；证书主题必须与清单中的发布者一致（`CN=VersionFox`）。此后每次发布都会自动签名。同时将公开证书（`.cer` 文件）随 release 一并分发，用户只需导入一次；由于各版本签名保持一致，覆盖升级不受影响。
- **受信 CA 证书**：由公共 CA 签发的代码签名证书或 [Azure Trusted Signing](https://learn.microsoft.com/azure/trusted-signing/overview) 服务签名的包，用户机器默认信任，安装时无需任何额外配置。但证书申请需要实名验证，OV/EV 证书的私钥按规定必须保存在硬件介质中，无法放入 CI secrets。

打包相关文件位于仓库的 [`packaging/msix/`](https://github.com/version-fox/vfox/tree/main/packaging/msix) 目录：

| 文件 | 用途 |
|------|------|
| `AppxManifest.xml` | 清单模板；`@@VERSION@@`、`@@ARCHITECTURE@@`、`@@PUBLISHER@@` 在构建时替换。 |
| `gen-assets.ps1` | 通过 System.Drawing 从仓库 logo（`logo.png`）生成磁贴图标：裁剪透明边距并将图案缩放至透明画布。PNG 是 AppX 部署的强制要求（不支持 SVG），因此这些二进制文件不入库，在每次构建时重新生成。 |
| `make-msix.ps1` | 使用 Go 从源码构建 `vfox`，为每种架构渲染清单，经 `MakeAppx.exe` 打包后合并为 `.msixbundle`，并可选通过 `SignTool.exe` 签名。 |

本地打包需要 Windows、[Windows SDK](https://developer.microsoft.com/windows/downloads/windows-sdk/)（提供 `MakeAppx.exe` / `SignTool.exe`）与 [Go](https://go.dev/dl/)：

```powershell
./packaging/msix/make-msix.ps1 -Version 1.2.3
```

产物输出到 `packaging/msix/Output/`；`assets/`、`staging/`、`build/` 为临时目录，均已 gitignore。仅支持不含预发布或构建元数据后缀的正式版本，并会归一化为四段版本号（`1.2.3` → `1.2.3.0`）。
