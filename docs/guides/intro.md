# Introduction

If you frequently switch between projects that require different runtime environments or runtime versions, or are tired of complicated environment configuration, `vfox` is your best choice.

`vfox` is a cross-platform, extensible universal version manager that supports **Windows (native)** and **Unix-like** systems, enabling you to **quickly install and switch** development environments.

It saves all tool version information in a `.vfox.toml` file, making it convenient to share configuration across projects and ensuring team members use the same tool versions.

Traditional solutions require installing multiple version managers (such as `nvm`, `fvm`, `sdkman`, `asdf-vm`, etc.), each with different APIs, configuration files, and implementations (involving `$PATH` operations, shims, environment variables, etc.). `vfox` provides a unified interaction method and configuration file to simplify the workflow, and can be extended to any tool and runtime environment through a simple plugin interface.

## Why Choose vfox?

- 💻 **Cross-platform support**: **Windows (native)**, Linux, macOS
- 🎯 **Flexible version scopes**: **Project-level**, **Session-level**, and **Global** version management
- 🔌 **Simple plugin system**: Easily extend support for any language
- 🔄 **Intelligent version switching**: Automatically switches to the appropriate version when entering a project directory
- 🔗 **Configuration file compatibility**: Supports existing formats like `.node-version`, `.nvmrc`, `.sdkmanrc`
- 🐚 **Full shell support**: Bash, ZSH, Fish, PowerShell, Clink, and more with command completion

## Supported Shells

| Shell      | Support | Note                                                                             |
|------------|---------|----------------------------------------------------------------------------------|
| PowerShell | ✅       |                                                                                  |
| Git Bash   | ✅       | [FAQ](./faq.md#why-can-t-i-select-when-use-use-and-search-commands-in-gitbash) |
| Bash       | ✅       |                                                                                  |
| ZSH        | ✅       |                                                                                  |
| Fish       | ✅       |                                                                                  |
| CMD        | ✅       | ⚠️ Only supports Global scope, not recommended                                  |
| Clink      | ✅       |                                                                                  |
| Cmder      | ✅       |                                                                                  |
| Nushell    | ✅       |                                                                                  |

## Contributors

> [!TIP]
> Thanks to all contributors for their support and contributions to this project! 🎉🙏

### Main Repository

![contributors](https://contrib.rocks/image?repo=version-fox/vfox)

### Plugins Repository

![contributors](https://contrib.rocks/image?repo=version-fox/vfox-plugins)
