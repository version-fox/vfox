<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>

# VersionFox

[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub License](https://img.shields.io/github/license/version-fox/vfox?style=for-the-badge)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/version-fox/vfox?display_name=tag&style=for-the-badge)](https://github.com/version-fox/vfox/releases)
[![Discord](https://img.shields.io/discord/1191981003204477019?style=for-the-badge&logo=discord)](https://discord.gg/85c8ptYgb7)

[[English]](./README.md)  [[中文文档]](./README_CN.md)

## Introduction

**`vfox` is a cross-platform tool for managing SDK versions, extendable via plugins**. It allows you to quickly install
and switch between different versions of SDKs using the command line.

## Why use VersionFox?

- **cross-platform support** (Windows, Linux, macOS)
- single CLI for multiple languages
- **consistent commands** to manage all your languages
- support **Global**、**Project**、**Session** scopes when switching versions
- simple **plugin system** to add support for your language of choice
- **automatically switches** runtime versions as you traverse your project
- shell completion available for common shells (Bash, Zsh, Powershell)
- **it's faster than `asdf-vm`, and offers more simple commands and genuine cross-platform unification.**
  see [Comparison with asdf](https://vfox.lhan.me/misc/vs-asdf.html)

## Demo

[![asciicast](https://asciinema.org/a/630778.svg)](https://asciinema.org/a/630778)

## Quickstart

> For detailed installation instructions, see [Quick Start](https://vfox.lhan.me/guides/quick-start.html)

#### 1. Choose an [installation](https://vfox.lhan.me/guides/quick-start.html#_1-installation) that works for you.

#### 2. ⚠️ **_Hook vfox into your shell_ (pick one that works for your shell)** ⚠️

```bash
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish

# For PowerShell, add the following line to your $PROFILE:
Invoke-Expression "$(vfox activate pwsh)"
```

#### 3. Add an SDK plugin
```bash 
$ vfox add nodejs/nodejs
```

#### 4. Install a runtime

```bash
$ vfox install nodejs@21.5.0
```

#### 5. Switch runtime

```bash
$ vfox use nodejs@21.5.0
$ node -v
21.5.0
```

## Full Documentation

See [vfox.lhan.me](https://vfox.lhan.me) for full documentation.

## Supported Plugins

> If you have installed `vfox`, you can view all available plugins with the `vfox available` command.

[![plugins](https://skillicons.dev/icons?i=java,kotlin,nodejs,flutter,dotnet,python,dart,golang,maven,zig,deno&theme=light)](https://github.com/version-fox/version-fox-plugins)

For more details, see the [version-fox-plugins](https://github.com/version-fox/version-fox-plugins)

## Contributors

> Thanks to following people who contributed to this project. 🎉🎉🙏🙏

<a href="https://github.com/version-fox/vfox/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=version-fox/vfox" />
</a>

## Contributing

Bug reports, contributions and forks are welcome. All bugs or other forms of discussion happen
on [issues](http://github.com/version-fox/vfox/issues).

See more at [CONTRIBUTING.md](./CONTRIBUTING.md).

Plugin Contributions, please go to [version-fox-plugins](https://github.com/version-fox/version-fox-plugins).

## Star History

![Star History Chart](https://api.star-history.com/svg?repos=version-fox/vfox&type=Date)

## COPYRIGHT

[Apache 2.0 license](./LICENSE) - Copyright (C) 2024 Han Li
and [contributors](https://github.com/version-fox/vfox/graphs/contributors)

