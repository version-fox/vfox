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

**用户级安装（无需 sudo）**

如果您想将 `vfox` 安装到用户目录（`~/.local/bin`）而不是系统范围内，请使用 `--user` 标志。这对于没有 sudo 访问权限或系统目录为临时目录的环境（例如 Coder 工作区）特别有用：

```shell
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash -s -- --user
```

此命令将：
- 将 `vfox` 安装到 `~/.local/bin`（无需 sudo）
- 如果目录不存在，会自动创建
- 如果需要，会提供将 `~/.local/bin` 添加到 `PATH` 的说明

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

::: details Nushell

```shell
vfox activate nushell $nu.default-config-dir | save --append $nu.config-path
```

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

| 作用域   | 命令示例                  | 路径                                           | 作用范围         |
|----------|---------------------------|----------------------------------------------|--------------|
| Global  | `vfox use -g`     | `$HOME/.vfox/sdks`                          | 全局生效(用户级别）   |
| Project | `vfox use -p`     | `$PWD/.vfox/sdks`                            | 项目目录内生效      |
| Session | `vfox use -s`     | `$HOME/.vfox/tmp/<shell-pid>` | 当前Shell会话内生效 |

::: warning 作用域范围原理

`vfox` 针对不同作用域，会在不同路径下生成对应的`sdks`目录来存放对应版本的运行时，并将这些路径添加到环境变量`PATH`中，从而实现不同作用域下的版本切换。

举个例子:
- 全局作用域: `$HOME/.vfox/sdks/nodejs`
- 项目作用域: `$PWD/.vfox/sdks/nodejs`
- 会话作用域: `$HOME/.vfox/tmp/<shell-pid>/nodejs`

在`PATH`中，`vfox`会将这些路径按作用域优先级顺序添加到`PATH`中:

```shell
$PWD/.vfox/sdks/nodejs: $HOME/.vfox/tmp/<shell-pid>/nodejs: $HOME/.vfox/sdks/nodejs: $PATH
```
:::

### Project

**不同项目不同版本**

```shell
$ vfox use -p nodejs@20.9.0
```

当时你执行此命令后，`vfox`将会在当前目录下生成`.vfox/sdks/nodejs`目录软链到对应版本的运行时, 并将该路径添加到环境变量`PATH`中。

```shell
$ ls -alh .vfox/sdks/       
drwxr-xr-x  3 lihan  staff    96B Jan 11 12:44 .
drwxr-xr-x  3 lihan  staff    96B Jan 11 12:44 ..
lrwxr-xr-x  1 lihan  staff    54B Jan 11 12:44 nodejs -> /Users/lihan/.vfox/cache/nodejs/v-20.9.0/nodejs-20.9.0

$ echo $PATH
/project/docs/.vfox/sdks/nodejs/bin:$PATH
```

除此之外， 会将版本信息写入到当前目录下的`.vfox.toml`文件中:

```toml
[tools]
nodejs = "20.9.0"
```

针对团队协作, 你只需将`.vfox.toml`文件提交到代码仓库中, **`.vfox`目录添加到`.gitignore`中**。

::: danger 关于目录软链行为

为了方便管理和隔离作用域, `vfox` 会在不同作用域下创建对应的目录软链到实际安装的运行时目录。

如果你**不希望`vfox`在项目目录下创建软链**, 你可以通过`--unlink`来禁用该行为。 之后`vfox`只会在`.vfox.toml`中记录版本信息, 不会创建软链, 并在session级别生效。

```shell
$ vfox use -p --unlink nodejs@20.9.0
```

**强烈建议您，保持vfox默认行为!!!**
:::

### Session

**不同Shell不同版本**

```shell
$ vfox use -s nodejs
```

当前作用域的作用主要是满足**临时需求**，当你关闭当前终端时，`vfox` 会**自动切换回全局版本/项目版本**。

::: tip
默认配置在`$HOME/.version-fox/tmp/<shell-pid>/.vfox.toml` 文件中（临时目录）。
:::

### Global

**全局唯一**

使用以下命令可以设置一个全局版本：
```shell
$ vfox use -g nodejs
```

::: tip
默认配置在`$HOME/.vfox/.vfox.toml`文件中进行管理。
:::

## 效果演示

::: tip
文字表达远不如图片来的更直观, 我们直接上效果图!
:::

![nodejs](/demo-full.gif)

## 完成指南！

恭喜你完成了 `vfox` 的快速上手 🎉 你现在可以管理你的项目的 `nodejs` 版本了。对于项目中的其他工具类型可以执行类似步骤即可！

`vfox` 还有更多命令需要熟悉，你可以通过运行 `vfox --help` 或者 `vfox` 来查看它们。
