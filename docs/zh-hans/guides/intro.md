# 项目简介

如果你经常在多个项目间切换，而这些项目需要不同的运行环境或运行时版本，或者厌倦了繁琐的环境配置，`vfox` 正是你的最佳选择。

`vfox` 是一款跨平台、可扩展的通用版本管理器，支持 **Windows（原生）** 和 **Unix-like** 系统，让你能够**快速安装和切换**开发环境。

它将所有工具版本信息保存在 `.vfox.toml` 文件中，方便在项目间共享配置，确保团队成员使用相同的工具版本。

传统方案需要安装多个版本管理器（如 `nvm`、`fvm`、`sdkman`、`asdf-vm` 等），每个都有不同的 API、配置文件和实现方式（涉及 `$PATH` 操作、垫片、环境变量等）。`vfox` 提供统一的交互方式和配置文件来简化工作流程，并可通过简单的插件接口扩展到任何工具和运行环境。

## 为什么选择 vfox？

- 💻 跨平台支持：**Windows（原生）**、Linux、macOS
- 🎯 灵活的版本作用域：**项目级**、**会话级** 和 **全局** 版本管理
- 🔌 简单的插件系统：轻松扩展对任何语言的支持
- 🔄 智能版本切换：进入项目目录时自动切换对应版本
- 🔗 配置文件兼容：支持 `.node-version`、`.nvmrc`、`.sdkmanrc` 等现有格式
- 🐚 全 Shell 支持：Bash、ZSH、Fish、PowerShell、Clink 等，并提供命令补全

## 已支持 Shell

| Shell      | 支持 | 说明                                                                            |
|------------|------|---------------------------------------------------------------------------------|
| PowerShell | ✅   |                                                                                 |
| Git Bash   | ✅   | [常见问题](./faq.md#how-do-i-select-a-version-with-use-and-search-in-gitbash) |
| Bash       | ✅   |                                                                                 |
| ZSH        | ✅   |                                                                                 |
| Fish       | ✅   |                                                                                 |
| CMD        | ✅   | ⚠️ 仅支持全局作用域，不推荐使用                                                    |
| Clink      | ✅   |                                                                                 |
| Cmder      | ✅   |                                                                                 |
| Nushell    | ✅   |                                                                                 |



## 贡献者

> [!TIP]
> 感谢所有贡献者对本项目的支持与贡献！🎉🙏

### 核心仓库

![contributors](https://contrib.rocks/image?repo=version-fox/vfox)

### 插件仓库

![contributors](https://contrib.rocks/image?repo=version-fox/vfox-plugins)
