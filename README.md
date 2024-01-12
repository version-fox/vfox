<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>
<h1 style="margin-top: -40px">VersionFox</h1>

[![Go Report Card](https://goreportcard.com/badge/github.com/version-fox/vfox)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub](https://img.shields.io/github/license/version-fox/vfox)](https://wimg.shields.io/github/license/version-fox/vfox)
[![GitHub release](https://img.shields.io/github/v/release/version-fox/vfox)](https://github.com/version-fox/vfox/releases/latest)




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
[![asciicast](https://asciinema.org/a/630769.svg)](https://asciinema.org/a/630769)

## Quickstart

Install VersionFox (For detailed installation see [Installation](https://github.com/version-fox/vfox/wiki/Getting-Started#installation))

```bash
$ brew tap version-fox/tap
$ brew install vfox
```

Hook VersionFox into your shell (pick one that works for your shell)
```bash
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish

# For PowerShell, add the following line to your $PROFILE:
Invoke-Expression "$(vfox activate pwsh)"
```

Add an SDK plugin (For detailed usage see [Getting Started](https://github.com/version-fox/vfox/wiki/Getting-Started))
```bash 
$ vfox add zig/zig
```

Install an SDK version
```bash
$ vfox install zig@0.11.0
```

Use the installed SDK version
```bash
$ vfox use zig@0.11.0
$ zig version
0.11.0
```


## Documentation

- [Getting Started](https://github.com/version-fox/vfox/wiki/Getting-Started)
- [Commands Overview](https://github.com/version-fox/vfox/wiki/All-Commands)
- [Plugins Repository](https://github.com/version-fox/version-fox-plugins)
- [How to write a custom plugin?](https://github.com/version-fox/vfox/wiki/How-to-write-a-custom-plugin%3F)
- [What is the difference with asdf-vm?](https://github.com/version-fox/vfox/wiki/What-is-the-difference-with-asdf%3F)

 For more information, read the [Wiki](https://github.com/version-fox/vfox/wiki).

## Contributors

> Thanks to following people who contributed to this project. 🎉🎉🙏🙏

<a href="https://github.com/version-fox/vfox/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=version-fox/vfox" />
</a>

## Contributing

Bug reports, contributions and forks are welcome. All bugs or other forms of discussion happen on [issues](http://github.com/version-fox/vfox/issues).

See more at [CONTRIBUTING.md](./CONTRIBUTING.md).

Plugin Contributions, please go to [version-fox-plugins](https://github.com/version-fox/version-fox-plugins).


## COPYRIGHT

[Apache 2.0 licence](./LICENSE) - Copyright (C) 2024 Han Li and [contributors](https://github.com/version-fox/vfox/graphs/contributors)

