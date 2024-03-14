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
$ brew tap version-fox/tap
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

::: details Powershell

打开 PowerShell 配置文件:

```shell
New-Item -Type File -Path $PROFILE # 无需在意 `文件已存在` 错误
Invoke-Item $PROFILE
```

将下面一行添加到你的 $PROFILE 文件末尾并保存:

```shell
Invoke-Expression "$(vfox activate pwsh)"
```

:::

## 3. 添加插件

**命令**: `vfox add <plugin-name>`

安装了[vfox](https://github.com/version-fox/vfox)
后，你还做不了任何事情，您需要先安装相应的插件。

::: tip 注意
如果你不知道添加哪个插件, 你可以使用 `vfox available` 命令查看所有可用插件
:::

为了得到更好的体验, 我们使用`npmmirror`镜像源插件!

```bash 
$ vfox add nodejs/npmmirror
```
::: tip 关于插件和SDK的关系
在`vfox`理念中, 插件即SDK、SDK即插件. 你可以将插件理解为`vfox`的一种扩展, 用于管理不同的工具和运行环境。

以`nodejs/npmmirror`插件为例, `nodejs`是分类, `npmmirror`是插件名, 插件内部`name`字段标注的叫**SDK名**。

所以, 在删除插件时, 需要使用**SDK名**进行删除, 而不是插件名`nodejs/npmirror`或`npmmirror`。

:::



## 4. 安装运行时

在插件成功安装之后, 你就可以安装对应版本的Nodejs了。

**命令**: `vfox install nodejs@<version>`

我们将只安装最新可用的 `latest` 版本:

```
$ vfox install nodejs@latest
```

::: tip 注意
`vfox` 强制使用准确的版本。`latest` 是一个通过交给插件来解析到执行时刻的实际版本号的行为, 是否支持取决于插件的实现。
:::

当然,我们也可以安装指定版本:

```bash
$ vfox install nodejs@21.5.0
```

## 5. 切换运行时

**命令**: `vfox use [-p -g -s] nodejs[@<version>]`

`vfox` 支持三种作用域, 每个作用域生效的范围不同:

### Global

全局默认配置在`$HOME/.version-fox/.tool-versions`文件中进行管理。使用以下命令可以设置一个全局版本：

```shell
$ vfox use -g nodejs
```

`$HOME/.version-fox/.tool-versions` 文件内容将会如下所示：

```text
nodejs 21.5.0
```

::: danger 执行之后不生效?
请检查`$PATH`中, 是否存在**之前**通过其他方式安装的运行时!

对于**Windows**用户:

1.请确保系统环境变量`Path`中不存在**之前**通过其他方式安装的运行时!

2.`vfox` 会自动将安装的运行时添加到**用户环境变量** `Path`中。

3.如果你的`Path`中存在**之前**通过其他方式安装的运行时, 请手动删除!
:::

### Session

该作用域生效范围为Shell会话。也就是说Shell之间不会共享版本。

会话作用域被定义在`$HOME/.version-fox/tmp/<shell-pid>/.tool-versions` 文件中（临时目录）。使用以下命令可以设置一个会话版本：

```shell
$ vfox use -s nodejs
```

### Project

项目作用域被定义在 `$PWD/.tool-versions` 文件中（当前工作目录）。通常，这将会是一个项目的 Git 存储库。当在你想要的目录执行：

```shell
$ vfox use -p nodejs
```

::: warning 默认作用域

如果你不指定作用域，`vfox` 将会使用默认作用域。不同系统, 作用域不同:

对于**Windows**: 默认作用域为`Global`

对于**Unix-like**: 默认作用域为`Session`
:::

## 效果演示

::: tip
文字表达远不如图片来的更直观, 我们直接上效果图!
:::

![nodejs](/demo-full.gif)

## 完成指南！

恭喜你完成了 `vfox` 的快速上手 🎉 你现在可以管理你的项目的 `nodejs` 版本了。对于项目中的其他工具类型可以执行类似步骤即可！

`vfox` 还有更多命令需要熟悉，你可以通过运行 `vfox --help` 或者 `vfox` 来查看它们。
