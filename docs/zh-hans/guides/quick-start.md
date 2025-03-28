# 快速入门

这里以`Nodejs`为例，介绍如何使用`vfox`。

## 1. 安装vfox

### Windows

::: details Scoop

```shell
scoop install vfox
```

:::

::: details winget

```shell
winget install vfox
```

:::

::: details Setup安装器
前往 [Releases](https://github.com/version-fox/vfox/releases) 页面下载最新版本的`setup`安装器，然后按照安装向导进行安装。
:::

::: details 手动安装

1. 在[Releases](https://github.com/version-fox/vfox/releases)下载最新版本的`zip`安装包
2. 配置`PATH`环境变量，将`vfox`安装目录添加到`PATH`环境变量中。
   :::

### Unix-like

::: details Homebrew

```shell
$ brew install vfox
```

:::

::: details APT

```shell
 echo "deb [trusted=yes] https://apt.fury.io/versionfox/ /" | sudo tee /etc/apt/sources.list.d/versionfox.list
 sudo apt-get update
 sudo apt-get install vfox
```

:::

::: details YUM

```shell
echo '[vfox]
name=VersionFox Repo
baseurl=https://yum.fury.io/versionfox/
enabled=1
gpgcheck=0' | sudo tee /etc/yum.repos.d/versionfox.repo

sudo yum install vfox
```

:::

::: details 手动安装

```shell
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

:::

## 2. 挂载`vfox`到你的`Shell`

::: warning 注意!!!!!
请从下面选择一条适合你Shell的命令执行!
:::

::: details Bash

```shell
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
```

:::

::: details ZSH

```shell
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
```

:::

::: details Fish

```shell
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish
```

:::

::: details PowerShell

创建 PowerShell 配置:

```shell
if (-not (Test-Path -Path $PROFILE)) { New-Item -Type File -Path $PROFILE -Force }; Add-Content -Path $PROFILE -Value 'Invoke-Expression "$(vfox activate pwsh)"'
```

如果 PowerShell 提示：`在此系统上禁止运行脚本`，那么请你**以管理员身份重新运行 PowerShell**输入如下命令

```shell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
# 之后输入 Y，按回车
y
```

:::

::: details Clink & Cmder

1. 找到脚本存放路径:
   ```shell
   clink info | findstr scripts
   ```
2. 复制 [clink_vfox.lua](https://github.com/version-fox/vfox/blob/main/internal/shell/clink_vfox.lua) 到脚本目录下，`clink_vfox.lua`脚本只需放置在其中一个目录中，无需放入每个目录。
3. 重启 Clink 或 Cmder

:::

然后，打开一个新终端。

## 3. 添加插件

**命令**: `vfox add <plugin-name>`

安装了[vfox](https://github.com/version-fox/vfox)后，你还做不了任何事情，您**需要先安装相应的插件**。

::: tip 注意
你可以使用 `vfox available` 命令查看所有可用插件。
:::

```bash 
$ vfox add nodejs
```

## 4. 安装运行时

在插件成功安装之后, 你就可以安装对应版本的Nodejs了。

**命令**: `vfox install nodejs@<version>`

我们将只安装最新可用的 `latest` 版本:

```
$ vfox install nodejs@latest
```

::: warning 版本问题
`vfox` 强制使用准确的版本。`latest` 是一个通过交给插件来解析到执行时刻的实际版本号的行为, 是否支持取决于插件的实现。

如果你**不知道具体版本号**, 可通过 `vfox search nodejs` 来查看所有可用版本。
:::

::: tip 自动安装
`install`和 `search`命令会检测本地是否已经安装了插件，如果没有，它们会**自动安装插件**。
:::

当然,我们也可以安装指定版本:

```bash
$ vfox install nodejs@21.5.0
```

## 5. 切换运行时

**命令**: `vfox use [-p -g -s] nodejs[@<version>]`

`vfox` 支持三种作用域, 每个作用域生效的范围不同:

### Global

**全局唯一**

使用以下命令可以设置一个全局版本：
```shell
$ vfox use -g nodejs
```

::: tip
默认配置在`$HOME/.version-fox/.tool-versions`文件中进行管理。

`$HOME/.version-fox/.tool-versions` 文件内容将会如下所示：

```text
nodejs 21.5.0
```
:::


::: danger 执行之后不生效?
请检查`$PATH`中, 是否存在**之前**通过其他方式安装的运行时!

对于**Windows**用户:

1.请确保系统环境变量`Path`中不存在**之前**通过其他方式安装的运行时!

2.`vfox` 会自动将安装的运行时添加到**用户环境变量** `Path`中。

3.如果你的`Path`中存在**之前**通过其他方式安装的运行时, 请手动删除!
:::

### Project

**不同项目不同版本**

```shell
$ vfox use -p nodejs
```

当你进入到一个目录时，`vfox` 会**自动检测该目录下是否存在 `.tool-versions` 文件**，如果存在，`vfox` 会**自动切换到该项目指定的版本**。

::: tip

配置放置在 `$PWD/.tool-versions` 文件中（当前工作目录）。

:::

::: warning 默认作用域

如果你不指定作用域，`vfox` 将会使用默认作用域。不同系统, 作用域不同:

对于**Windows**: 默认作用域为`Global`

对于**Unix-like**: 默认作用域为`Session`
:::

### Session

**不同Shell不同版本**

```shell
$ vfox use -s nodejs
```

当前作用域的作用主要是满足**临时需求**，当你关闭当前终端时，`vfox` 会**自动切换回全局版本/项目版本**。

::: tip
默认配置在`$HOME/.version-fox/tmp/<shell-pid>/.tool-versions` 文件中（临时目录）。
:::



## 效果演示

::: tip
文字表达远不如图片来的更直观, 我们直接上效果图!
:::

![nodejs](/demo-full.gif)

## 完成指南！

恭喜你完成了 `vfox` 的快速上手 🎉 你现在可以管理你的项目的 `nodejs` 版本了。对于项目中的其他工具类型可以执行类似步骤即可！

`vfox` 还有更多命令需要熟悉，你可以通过运行 `vfox --help` 或者 `vfox` 来查看它们。
