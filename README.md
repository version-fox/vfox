<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>

# vfox

[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub License](https://img.shields.io/github/license/version-fox/vfox?style=for-the-badge)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/version-fox/vfox?display_name=tag&style=for-the-badge)](https://github.com/version-fox/vfox/releases)
[![Discord](https://img.shields.io/discord/1191981003204477019?style=for-the-badge&logo=discord)](https://discord.gg/85c8ptYgb7)

[[English]](./README.md)  [[中文文档]](./README_CN.md)

If you **switch between development projects which expect different environments**, specifically different runtime versions or ambient libraries,
or **you are tired of all kinds of cumbersome environment configurations**, `vfox` is the ideal choice for you.
## Introduction

**`vfox` is a cross-platform version manager(similar to `nvm`, `fvm`, `sdkman`, `asdf-vm`, etc.), extendable via plugins**. It allows you to quickly install
and switch between different environment you need via the command line.

## Why use vfox?

- **cross-platform support** (Windows, Linux, macOS)
- single CLI for multiple languages
- **consistent commands** to manage all your languages
- support **Global**、**Project**、**Session** scopes when switching versions
- simple **plugin system** to add support for your language of choice
- **automatically switches** runtime versions as you traverse your project
- shell completion available for common shells (Bash, ZSH, Powershell)
- **it's faster than `asdf-vm`, and offers more simple commands and genuine cross-platform unification.**
  see [Comparison with asdf](https://vfox.lhan.me/misc/vs-asdf.html)

## Demo

[![asciicast](https://asciinema.org/a/650100.svg)](https://asciinema.org/a/650100)

## Quickstart

> For detailed installation instructions, see [Quick Start](https://vfox.lhan.me/guides/quick-start.html)

#### 1. Choose an [installation](https://vfox.lhan.me/guides/quick-start.html#_1-installation) that works for you.

#### 2. ⚠️ **_Hook vfox into your shell_ (pick one that works for your shell)** ⚠️

```bash
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish

# For PowerShell:
# 1. Open PowerShell Profile:
New-Item -Type File -Path $PROFILE # Just ignore the 'file already exists' error.
Invoke-Item $PROFILE
# 2. Add the following line to the end of your $PROFILE and save:
Invoke-Expression "$(vfox activate pwsh)"

# For Clink:
# 1. Install clink: https://github.com/chrisant996/clink/releases
#    Or Install cmder: https://github.com/cmderdev/cmder/releases
# 2. Find script path: clink info | findstr scripts
# 3. copy internal/shell/clink_vfox.lua to script path
```

> Remember to restart your shell to apply the changes.

#### 3. Add an SDK plugin
```bash 
$ vfox add nodejs
```
or add more SDK plugins
```bash
$ vfox add nodejs golang
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

## Roadmap

Our future plans and high priority features and enhancements are:

- [x] Refactoring the plugin mechanism:
  - Introducing plugin templates to facilitate multi-file plugin development.
  - Establishing a global registry (similar to `NPM Registry` or `Scoop Main Bucket`) to provide a unified entry point for plugin distribution.
  - Decomposing the existing plugin repository into individual repositories, one for each plugin.
- [X] Allowing the switching of registry addresses.
- [ ] Plugin capabilities: Parsing legacy configuration files, such as `.nvmrc`, `.node-version`, `.sdkmanrc`, etc.
- [ ] Plugin capabilities: Allowing plugins to load installed runtimes and provide information about the runtime.

## Available Plugins

> If you have installed `vfox`, you can view all available plugins with the `vfox available` command.

For more details, see the [Available Plugins](https://vfox.lhan.me/plugins/available.html).

## Contributors

> Thanks to following people who contributed to this project. 🎉🎉🙏🙏

<a href="https://github.com/version-fox/vfox/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=version-fox/vfox" />
</a>

## Contributing

Bug reports, contributions and forks are welcome. All bugs or other forms of discussion happen
on [issues](http://github.com/version-fox/vfox/issues).

See more at [CONTRIBUTING.md](./CONTRIBUTING.md).

Plugin Contributions, please go to [Public Registry](https://github.com/version-fox/vfox-plugins)

## Star History

![Star History Chart](https://api.star-history.com/svg?repos=version-fox/vfox&type=Date)

## Thanks

<a href="https://hellogithub.com/repository/a32a1f2ad04a4b8aa4dd3e1b76c880b2" target="_blank"><img src="https://api.hellogithub.com/v1/widgets/recommend.svg?rid=a32a1f2ad04a4b8aa4dd3e1b76c880b2" alt="Featured｜HelloGitHub" style="width: 250px; height: 54px;" width="250" height="54" /></a>

## COPYRIGHT

[Apache 2.0 license](./LICENSE) - Copyright (C) 2024 Han Li
and [contributors](https://github.com/version-fox/vfox/graphs/contributors)

