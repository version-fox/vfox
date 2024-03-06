# Introduction

If you switch between development projects which expect different environments, specifically different runtime versions or ambient libraries,
or you are tired of all kinds of cumbersome environment configurations, `vfox` is the ideal choice for you.

`vfox` is a cross-platform, extensible version manager. It supports **native Windows**, and of course **Unix-like**!
With it, you can **quickly install and switch** different environment.

It saves all tool version information in a file named `.tool-versions`, so you can share this information in your
project to ensure that everyone on your team is using the same tool versions.

Traditional work requires multiple cli version managers, each with its own API, configuration files, and
implementation (e.g., `$PATH` operations, shims, environment variables, etc.). `vfox` provides a single interactive way
and configuration file to simplify the development workflow and can be extended to all tools and runtime environments
through a simple plugin interface.

## Why use VersionFox?

- **Cross-platform** Supports Windows (non-WSL), Linux, macOS!
- **Consistent commands** for managing all your languages
- Supports **Global**, **Project**, **Session** three scopes
- Simple **plugin system** to add support for the languages you choose
- **Automatically switch** runtime versions for you when you switch projects
- Supports common Shells (Powershell, bash, zsh), and provides autocompletion
- **Faster than `asdf-vm`**, and provides simpler commands and true cross-platform unification.
  See [Comparison to asdf](../misc/vs-asdf.md)。

## Contributors


> [!TIP]
> Thanks to the following contributors for their contributions.🎉🎉🙏🙏

#### [vfox](https://github.com/version-fox/vfox)

![pluigns](https://contrib.rocks/image?repo=version-fox/vfox)

#### [vfox-plugins](https://github.com/version-fox/version-fox-plugins)

![pluigns](https://contrib.rocks/image?repo=version-fox/version-fox-plugins)
