# 快速入门

这里以 `Nodejs` 为例，介绍如何使用 `vfox`。

## 1. 安装 vfox

### Windows

<Tabs>
<TabItem label="Scoop">

```shell
scoop install vfox
```

</TabItem>
<TabItem label="winget">

```shell
winget install vfox
```

</TabItem>
<TabItem label="Setup 安装器">

前往 [Releases](https://github.com/version-fox/vfox/releases) 页面下载最新版本的 `setup` 安装器，然后按照安装向导进行安装。

</TabItem>
</Tabs>

### Unix-like

<Tabs>
<TabItem label="Homebrew">

```shell
brew install vfox
```

</TabItem>
<TabItem label="APT (Debian/Ubuntu)">

```shell
echo "deb [trusted=yes lang=none] https://apt.fury.io/versionfox/ /" | sudo tee /etc/apt/sources.list.d/versionfox.list
sudo apt-get update
sudo apt-get install vfox
```

</TabItem>
<TabItem label="YUM (CentOS/Fedora)">

```shell
echo '[vfox]
name=VersionFox Repo
baseurl=https://yum.fury.io/versionfox/
enabled=1
gpgcheck=0' | sudo tee /etc/yum.repos.d/versionfox.repo

sudo yum install vfox
```

</TabItem>
<TabItem label="安装脚本">

```shell
curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

**用户级安装（无需 sudo）**

如果你想将 `vfox` 安装到用户目录（`~/.local/bin`）而不是系统范围内，可以使用 `--user` 标志。这对于没有 sudo 访问权限的环境特别有用：

```shell
curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash -s -- --user
```

此命令将：

- 将 `vfox` 安装到 `~/.local/bin`（无需 sudo）
- 如果目录不存在，会自动创建
- 提供将 `~/.local/bin` 添加到 `PATH` 的说明

</TabItem>
</Tabs>

## 2. 挂载 vfox 到 Shell

> [!WARNING] ⚠️注意
> 请根据你使用的 Shell 类型，选择对应的配置方式

<Tabs>
<TabItem label="Bash">

```shell
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
source ~/.bashrc
```

</TabItem>
<TabItem label="ZSH">

```shell
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
```

</TabItem>
<TabItem label="Fish">

```shell
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish
```

</TabItem>
<TabItem label="PowerShell">

创建 PowerShell 配置：

```powershell
if (-not (Test-Path -Path $PROFILE)) { New-Item -Type File -Path $PROFILE -Force | Out-Null }
$vfoxLine = 'Invoke-Expression "$(vfox activate pwsh)"'
$profileContent = Get-Content -Path $PROFILE -Raw
if ($profileContent -notmatch [regex]::Escape($vfoxLine)) {
  if ($profileContent.Length -gt 0 -and -not $profileContent.EndsWith("`r`n") -and -not $profileContent.EndsWith("`n")) {
    Add-Content -Path $PROFILE -Value ""
  }
  Add-Content -Path $PROFILE -Value $vfoxLine
}
```

这段脚本可以重复执行。它会避免重复写入 vfox 激活行，并确保该行写入到新的一行中。

如果 PowerShell 提示「在此系统上禁止运行脚本」，请**以管理员身份运行 PowerShell** 并执行：

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
```

输入 `Y` 后按回车确认。

</TabItem>
<TabItem label="Clink & Cmder">

1. 找到脚本存放路径：

    ```shell
    clink info | findstr scripts
    ```

2. 复制 [clink_vfox.lua](https://github.com/version-fox/vfox/blob/main/internal/shell/clink_vfox.lua) 到脚本目录
3. 重启 Clink 或 Cmder

</TabItem>
<TabItem label="Nushell">

```shell
vfox activate nushell $nu.default-config-dir | save --append $nu.config-path
```

</TabItem>
</Tabs>


## 3. 添加插件

**命令**: `vfox add <plugin-name>`

安装了 vfox 后，您还需要安装相应的插件才能管理 SDK。

::: tip 💡提示
可以使用 `vfox available` 命令查看所有可用插件。
:::

```bash
vfox add nodejs
```

## 4. 安装运行时

在插件成功安装之后，您就可以安装对应版本的 Node.js 了。

**命令**: `vfox install nodejs@<version>`

```bash
vfox install nodejs@21.5.0
```

::: warning ⚠️ Latest 版本说明

`latest` 是一个特殊标记，取插件返回的可用版本列表中的第一个版本（通常是最新版）。`install` 和 `use` 命令均支持此标记，但 **不推荐在生产环境使用**。

```bash
vfox install nodejs@latest
vfox use -g nodejs@latest
```

**为什么不推荐？** `latest` 会指向当前最新版本，但新版本可能包含破坏性变更或不稳定特性，容易导致项目出现兼容性问题。

::: tip 💡 推荐做法
始终使用准确的版本号，以确保项目的稳定性和可复现性。可通过 `vfox search nodejs` 查询所有可用版本。
:::

::: tip 💡 自动安装插件
`install` 和 `search` 命令会自动检测并安装缺失的插件。
:::

## 5. 切换运行时

**命令**: `vfox use [-p -g -s] [--unlink] nodejs[@<version>]`

`vfox` 支持三种作用域，版本优先级从高到低为：

**Project > Session > Global > System**

::: info 📌 默认行为
**vfox 默认是 Session 级别**，如果不指定 `-p`（项目级）或 `-g`（全局）标志，直接使用 `vfox use` 命令等同于 `vfox use -s`。关闭 Shell 会话时，Session 级别的配置会自动清理销毁。
:::

### 作用域概览

| 作用域         | 命令            | SDK 路径                   | 作用范围        |
|-------------|---------------|--------------------------|-------------|
| **Project** | `vfox use -p` | `$PWD/.vfox/sdks`        | 当前项目目录      |
| **Session** | `vfox use -s` | `~/.vfox/tmp/<pid>/sdks` | 当前 Shell 会话 |
| **Global**  | `vfox use -g` | `~/.vfox/sdks`           | 全局生效        |

::: info 📖 工作原理

vfox 通过在不同作用域创建目录软链来指向实际 SDK 安装目录，并将这些路径按优先级添加到 `PATH` 环境变量中，实现版本切换。

**PATH 优先级示例**：

```bash
# Project > Session > Global > System
$PWD/.vfox/sdks/nodejs/bin:~/.vfox/tmp/<pid>/nodejs/bin:~/.vfox/sdks/nodejs/bin:/usr/bin:...
```

:::

---

### Project（项目作用域）

::: tip 💡 推荐
用于项目开发，每个项目可以有独立的工具版本。
:::

**用法**：

```bash
# 在当前项目目录下使用 nodejs
vfox use -p nodejs@20.9.0
```

**执行后，vfox 会做如下操作**：

1. **创建目录软链**：在 `$PWD/.vfox/sdks/nodejs` 下创建符号链接，指向实际安装目录
2. **主动添加.gitignore**: 如果检测到存在 `.gitignore` 文件，vfox 会自动将 `.vfox/` 目录添加到忽略列表中，防止提交到代码仓库中
3. **更新 PATH**：将 `$PWD/.vfox/sdks/nodejs/bin` 插入到 `PATH` 的最前面
4. **保存配置**：将版本信息写入 `.vfox.toml` 文件

这样当你在该项目目录执行 `node` 命令时，Shell 会从 PATH 最前面查找到你的项目级 nodejs，确保版本符合项目要求。

**可视化示例**：

```bash
# 1. 执行命令
$ vfox use -p nodejs@20.9.0

# 2. 查看创建的符号链接
$ ls -la .vfox/sdks/nodejs
lrwxr-xr-x  1 user  staff  nodejs -> /Users/user/.vfox/cache/nodejs/v-20.9.0/nodejs-20.9.0

# 3. 查看更新的 PATH
$ echo $PATH
/project/path/.vfox/sdks/nodejs/bin:/previous/paths:...
#                  ↑ 项目级 nodejs 在最前面

# 4. 查看配置文件
$ cat .vfox.toml
[tools]
nodejs = "20.9.0"

# 5. 验证版本（使用的是项目级版本）
$ node -v
v20.9.0
```

::: warning 💡 强烈推荐
将 `.vfox.toml` 提交到代码仓库，将 `.vfox` 目录添加到 `.gitignore`。这样团队成员可以共享版本配置。
:::

::: danger ⚠️ 关于 --unlink 参数

如果不想在项目目录创建符号链接，可以使用 `--unlink` 参数：

```bash
vfox use -p --unlink nodejs@20.9.0
```

**注意**：使用 `--unlink` 后，Project 作用域会降级为 Session 作用域（配置记录在 .vfox.toml 但不创建软链），**强烈建议保持默认行为**（创建软链）。
:::

---

### Session（会话作用域）

::: tip 💡 临时测试
用于临时测试特定版本，关闭当前 Shell 窗口时自动失效。
:::

**用法**：

```bash
vfox use -s nodejs@18.0.0
```

::: warning 📝 重要提醒
- **会话级别作用域**：`vfox use -s` 只在当前 Shell 会话内生效
- **自动清理**：关闭 Shell 窗口/会话时，临时目录及配置随之自动清理
- **不影响其他会话**：变更仅对当前会话有效，不会影响其他 Shell 会话或全局设置
:::



---

### Global（全局作用域）

::: tip 💡 用户级默认版本
用于设置用户级别的默认版本，所有项目都可使用（除非被 Project 或 Session 覆盖）。
:::

**用法**：

```bash
vfox use -g nodejs@21.5.0
```

## 效果演示

文字表达不如图片直观，直接看效果演示！

![nodejs](/demo-full.gif)

## 完成快速入门！🎉

恭喜你完成了 `vfox` 的快速上手！现在你可以：

- ✅ 快速安装和切换不同版本的开发工具
- ✅ 为项目配置独立的工具版本
- ✅ 临时测试特定的工具版本
- ✅ 与团队共享一致的开发环境配置

**下一步：**

使用 `vfox --help` 查看更多命令和选项，或访问 [全部命令](../usage/all-commands.md) 了解更多功能。
