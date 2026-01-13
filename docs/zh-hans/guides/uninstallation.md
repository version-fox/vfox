# 卸载

本指南将帮助您从系统中完全删除 `vfox`。

## 1. 移除 Shell 钩子

首先，您需要从 Shell 配置文件中删除 `vfox` 激活命令。

::: warning 注意!!!!!
请选择适合您 Shell 的说明！
:::

::: details Bash

打开您的 `~/.bashrc` 文件并删除以下行：

```shell
eval "$(vfox activate bash)"
```

保存文件后，重新加载您的 shell 配置：

```shell
source ~/.bashrc
```

:::

::: details ZSH

打开您的 `~/.zshrc` 文件并删除以下行：

```shell
eval "$(vfox activate zsh)"
```

保存文件后，重新加载您的 shell 配置：

```shell
source ~/.zshrc
```

:::

::: details Fish

打开您的 `~/.config/fish/config.fish` 文件并删除以下行：

```shell
vfox activate fish | source
```

保存文件后，重新加载您的 shell 配置：

```shell
source ~/.config/fish/config.fish
```

:::

::: details PowerShell

打开您的 PowerShell 配置文件。您可以通过运行以下命令找到其位置：

```powershell
$PROFILE
```

常见位置：
- `C:\Users\<用户名>\Documents\PowerShell\Microsoft.PowerShell_profile.ps1` (PowerShell 7+)
- `C:\Users\<用户名>\Documents\WindowsPowerShell\Microsoft.PowerShell_profile.ps1` (Windows PowerShell)

从配置文件中删除以下行：

```powershell
Invoke-Expression "$(vfox activate pwsh)"
```

保存文件后，重新加载您的 PowerShell 配置文件：

```powershell
. $PROFILE
```

:::

::: details Clink & Cmder

1. 找到脚本存放路径：
   ```shell
   clink info | findstr scripts
   ```
2. 从脚本目录中删除 `clink_vfox.lua` 文件
3. 重启 Clink 或 Cmder

:::

::: details Nushell

打开您的 Nushell 配置文件（位置由 `$nu.config-path` 显示）并删除安装期间添加的 vfox 激活行。

:::

## 2. 卸载 vfox 二进制文件

从系统中删除 `vfox` 可执行文件。

### Windows

::: details Scoop

```shell
scoop uninstall vfox
```

:::

::: details winget

```shell
winget uninstall vfox
```

:::

::: details Setup 安装器

1. 打开 **设置** > **应用** > **已安装的应用**（或 **控制面板** > **程序** > **卸载程序**）
2. 在列表中找到 **vfox**
3. 点击 **卸载** 并按照向导操作

:::

::: details 手动安装

1. 删除您解压 vfox 的目录
2. 从 `PATH` 环境变量中删除 `vfox` 安装目录：
   - 打开 **系统属性** > **环境变量**
   - 在 **用户变量** 或 **系统变量** 中找到 `Path`
   - 删除指向 vfox 目录的条目
   - 点击 **确定** 保存

:::

### Unix-like

::: details Homebrew

```shell
brew uninstall vfox
```

:::

::: details APT

```shell
sudo apt-get remove vfox
```

同时删除仓库配置：

```shell
sudo rm /etc/apt/sources.list.d/versionfox.list
```

:::

::: details YUM

```shell
sudo yum remove vfox
```

同时删除仓库配置：

```shell
sudo rm /etc/yum.repos.d/versionfox.repo
```

:::

::: details 手动安装

删除 vfox 二进制文件：

```shell
sudo rm /usr/local/bin/vfox
```

:::

## 3. 清理 vfox 数据（可选）

如果您想完全删除 `vfox` 存储的所有数据，包括已安装的 SDK、插件和配置文件：

::: warning 警告
这将永久删除您通过 vfox 安装的所有 SDK 版本！
:::

### 删除 vfox 数据目录

```shell
rm -rf ~/.version-fox
```

此目录包含：
- 已安装的 SDK 版本
- 插件文件
- 配置文件
- 全局 `.vfox.toml` 文件
- 缓存和临时文件

## 验证卸载

要验证 `vfox` 是否已完全删除：

```shell
which vfox
# 或
vfox --version
```

两个命令都应该返回"command not found"或类似的错误消息。

## 故障排除

### 卸载后 vfox 命令仍然有效

- 确保在删除 shell 钩子后关闭并重新打开所有终端窗口
- 检查是否有多个 shell 配置文件（例如 `.bash_profile`、`.profile`、`.bashrc`），并确保从所有文件中删除 vfox 激活行
- 在 Windows 上，重新启动计算机以确保所有环境变量更改生效

### 通过 vfox 安装的 SDK 仍然出现在 PATH 中

- 检查当前目录或主目录中是否有 `.vfox.toml` 文件
- 按照上述步骤 3 中的说明删除 vfox 数据目录
- 在 Windows 上，手动检查您的用户环境变量并删除 vfox 添加的任何与 SDK 相关的路径
