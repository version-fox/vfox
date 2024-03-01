<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>

# VersionFox

[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub License](https://img.shields.io/github/license/version-fox/vfox?style=for-the-badge)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/version-fox/vfox?display_name=tag&style=for-the-badge)](https://github.com/version-fox/vfox/releases)
[![Discord](https://img.shields.io/discord/1191981003204477019?style=for-the-badge&logo=discord)](https://discord.gg/85c8ptYgb7)

[[English]](./README.md)  [[中文文档]](./README_CN.md)

## 介绍

`vfox` 是一个跨平台的 SDK 版本管理工具，通过插件机制进行拓展。它允许您通过命令行快速安装和切换不同版本的 SDK。

## 为什么选择 vfox？

- 支持**Windows(非WSL)**、Linux、macOS!
- **一致的命令** 用于管理你所有的语言
- 支持**Global**、**Project**、**Session** 三种作用域
- 简单的 **插件系统** 来添加对你选择的语言的支持
- 在您切换项目时, 帮您**自动切换**运行时版本
- 支持常用Shell(Powershell、bash、zsh),并提供补全功能
- **比 `asdf-vm` 更快**，并提供更简单的命令和真正的跨平台统一。参见 [与asdf-vm对比](https://vfox.lhan.me/zh-hans/misc/vs-asdf.html)。

## 演示

[![asciicast](https://asciinema.org/a/630778.svg)](https://asciinema.org/a/630778)

## 快速入门

> 详细的安装指南请参见 [快速入门](https://vfox.lhan.me/zh-hans/guides/quick-start.html)

#### 1.选择一个适合你的[安装方式](https://vfox.lhan.me/zh-hans/guides/quick-start.html#_1-%E5%AE%89%E8%A3%85vfox)。

#### 2. ⚠️ **挂载vfox到你的Shell (从下面选择一条适合你shell的命令)** ⚠️

```bash
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish

# PowerShell, 请将下面一行添加到你的$PROFILE文件中:
Invoke-Expression "$(vfox activate pwsh)"
```

#### 3.添加插件
```bash 
$ vfox add nodejs/nodejs
```

#### 4. 安装运行时

```bash
$ vfox install nodejs@21.5.0
```

#### 5. 切换运行时

```bash
$ vfox use nodejs@21.5.0
$ node -v
21.5.0
```

## 完整文档

请浏览 [vfox.lhan.me](https://vfox.lhan.me) 查看完整文档。

## 目前支持的插件

> 如果您已经安装了 `vfox`，您可以使用 `vfox available` 命令查看所有可用的插件。

[![plugins](https://skillicons.dev/icons?i=java,kotlin,nodejs,flutter,dotnet,python,dart,golang,gradle,maven,zig,deno&theme=light)](https://github.com/version-fox/version-fox-plugins)

详细内容,请看 [version-fox-plugins](https://github.com/version-fox/version-fox-plugins)

## 贡献者

> 感谢以下贡献者对本项目的贡献。🎉🎉🙏🙏

<a href="https://github.com/version-fox/vfox/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=version-fox/vfox" />
</a>


## Star History

![Star History Chart](https://api.star-history.com/svg?repos=version-fox/vfox&type=Date)

## COPYRIGHT

[Apache 2.0 license](./LICENSE) - Copyright (C) 2024 Han Li
and [contributors](https://github.com/version-fox/vfox/graphs/contributors)

