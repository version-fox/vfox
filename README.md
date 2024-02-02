<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>

# VersionFox

[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub License](https://img.shields.io/github/license/version-fox/vfox?style=for-the-badge)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/version-fox/vfox?display_name=tag&style=for-the-badge)](https://github.com/version-fox/vfox/releases)
[![Discord](https://img.shields.io/discord/1191981003204477019?style=for-the-badge&logo=discord)](https://discord.gg/PdEvGXHp)




[[English]](./README.md)  [[中文文档]](./README_CN.md)


## Introduction

VersionFox is a cross-platform tool for managing SDK versions. It allows you to quickly install
and switch between different versions of SDKs using the command line.
SDKs are provided as plugins in the form of Lua scripts. This means you can implement your own SDK sources or use
plugins shared by others to install SDKs. It all depends on your imagination. ;)

## Why use VersionFox?

- **cross-platform support** (Windows, Linux, macOS)
- single CLI for multiple languages
- **consistent commands** to manage all your languages
- support **Global**、**Project**、**Session** scopes when switching versions
- simple **plugin system** to add support for your language of choice
- **automatically switches** runtime versions as you traverse your project
- shell completion available for common shells (Bash, Zsh, Powershell)
- **it's faster than `asdf-vm`, and offers more simple commands and genuine cross-platform unification.** see [What-is-the-difference-with-asdf?](https://github.com/version-fox/vfox/wiki/What-is-the-difference-with-asdf%3F)

## Demo
[![asciicast](https://asciinema.org/a/630778.svg)](https://asciinema.org/a/630778)

## Quickstart

Install VersionFox (For detailed installation see [Installation](https://github.com/version-fox/vfox/wiki/Getting-Started#installation))

```bash
$ brew tap version-fox/tap
$ brew install vfox
```

⚠️ **_Hook VersionFox into your shell_ (pick one that works for your shell)** ⚠️

```bash
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish

# For PowerShell, add the following line to your $PROFILE:
Invoke-Expression "$(vfox activate pwsh)"
```

Add an SDK plugin (For detailed usage see [Getting Started](https://github.com/version-fox/vfox/wiki/Getting-Started))
> NOTE: if you don’t know which plugins to add, you can use the `vfox available` command to check all available plugins
```bash 
$ vfox add nodejs/nodejs
```

Install an SDK version
```bash
$ vfox install nodejs@21.5.0
```

Use the installed SDK version
```bash
$ vfox use nodejs@21.5.0
$ node -v
21.5.0
```


## Documentation

- [Getting Started](https://github.com/version-fox/vfox/wiki/Getting-Started)
- [Commands Overview](https://github.com/version-fox/vfox/wiki/All-Commands)
- [How to write a custom plugin?](https://github.com/version-fox/vfox/wiki/How-to-write-a-custom-plugin%3F)
- [What is the difference with asdf-vm?](https://github.com/version-fox/vfox/wiki/What-is-the-difference-with-asdf%3F)

 For more information, read the [Wiki](https://github.com/version-fox/vfox/wiki).


## Supported Plugins

If you have installed `vfox`, you can view all available plugins with the `vfox available` command.

Or please see the [version-fox-plugins](https://github.com/version-fox/version-fox-plugins) repository.

## FAQ

### 1.**Switch xxx not work or the `vfox use ` command does not work ?**

If your shell prompt `Warning: The current shell lacks hook support or configuration. It has switched to global scope automatically` that
means you do not hook `vfox` into your shell, please hook it manually first.

See [issue#35](https://github.com/version-fox/vfox/issues/35)



## Contributors

> Thanks to following people who contributed to this project. 🎉🎉🙏🙏

<a href="https://github.com/version-fox/vfox/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=version-fox/vfox" />
</a>

## Contributing

Bug reports, contributions and forks are welcome. All bugs or other forms of discussion happen on [issues](http://github.com/version-fox/vfox/issues).

See more at [CONTRIBUTING.md](./CONTRIBUTING.md).

Plugin Contributions, please go to [version-fox-plugins](https://github.com/version-fox/version-fox-plugins).

## Star History

![Star History Chart](https://api.star-history.com/svg?repos=version-fox/vfox&type=Date)

## COPYRIGHT

[Apache 2.0 license](./LICENSE) - Copyright (C) 2024 Han Li and [contributors](https://github.com/version-fox/vfox/graphs/contributors)

