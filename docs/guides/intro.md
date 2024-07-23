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

## Why use vfox?

- **cross-platform support** (**Windows**, Linux, macOS)
- **consistent commands** to manage all your languages
- supports **different versions for different projects, different shells, and globally**.
- simple **plugin system** to add support for your runtime of choice
- **automatically switches** runtime versions as you traverse your project
- support for existing config files `.node-version`, `.nvmrc`, `.sdkmanrc` for easy migration
- shell completion available for common shells (Bash, ZSH, Powershell, Clink)
- **Faster than `asdf-vm`**, and provides simpler commands and true cross-platform unification.
  See [Comparison to asdf](../misc/vs-asdf.md)。

## Supported Shell

| Shell      | Support | Note                                                                             |
|------------|---------|----------------------------------------------------------------------------------|
| Powershell | ✅       |                                                                                  |
| GitBash    | ✅       | [Issue](./faq.md#why-can-t-i-select-when-use-use-and-search-commands-in-gitbash) |
| Bash       | ✅       |                                                                                  |
| ZSH        | ✅       |                                                                                  |
| Fish       | ✅       |                                                                                  |
| CMD        | ✅       | Only Support `Global` Scope. Not Recommend!!!                                    |
| Clink      | ✅       |                                                                                  |
| Cmder      | ✅       |                                                                                  |



## Contributors


> [!TIP]
> Thanks to the following contributors for their contributions.🎉🎉🙏🙏

#### [vfox](https://github.com/version-fox/vfox)

![plugins](https://contrib.rocks/image?repo=version-fox/vfox)

#### [Public Registry](https://github.com/version-fox/vfox-plugins)

![plugins](https://contrib.rocks/image?repo=version-fox/vfox-plugins))
