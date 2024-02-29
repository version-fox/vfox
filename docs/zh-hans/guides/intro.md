# 项目简介

`vfox` 是一款跨平台、可拓展的通用版本管理器。支持**原生Windows**, 当然**Unix-like**也一定支持! 通过它，您可以**快速安装和切换**工具版本。

它将所有的工具版本信息保存在一个名为 `.tool-versions` 的文件中，这样您就可以在项目中共享这些信息，确保团队中的每个人都使用相同的工具版本。

传统工作方式需要多个命令行版本管理器，而且每个管理器都有其不同的 API、配置文件和实现方式（比如，`$PATH`
操作、垫片、环境变量等等）。`vfox` 提供单个交互方式和配置文件来简化开发工作流程，并可通过简单的插件接口扩展到所有工具和运行环境。


## 为什么选择 vfox？

- 支持**Windows(非WSL)**、Linux、macOS!
- **一致的命令** 用于管理你所有的语言
- 支持**Global**、**Project**、**Session** 三种作用域
- 简单的 **插件系统** 来添加对你选择的语言的支持
- 在您切换项目时, 帮您**自动切换**运行时版本
- 支持常用Shell(Powershell、bash、zsh),并提供补全功能
- **比 `asdf-vm` 更快**，并提供更简单的命令和真正的跨平台统一。参见 [与asdf-vm对比](../misc/vs-asdf.md)。


## 贡献者

> [!TIP]
> 感谢以下贡献者对本项目的贡献。🎉🎉🙏🙏

#### [核心仓库](https://github.com/version-fox/vfox)


![pluigns](https://contrib.rocks/image?repo=version-fox/vfox)

#### [插件仓库](https://github.com/version-fox/version-fox-plugins)


![pluigns](https://contrib.rocks/image?repo=version-fox/version-fox-plugins)
