# 在 mise 中使用 vfox 插件

[mise](https://mise.jdx.dev/) 可以通过它的
[`vfox` 后端](https://mise.jdx.dev/dev-tools/backends/vfox.html)安装和运行 vfox 插件。
mise 不会调用 `vfox` 可执行文件，而是用 Rust 自己实现了一套插件运行时，所以同一个插件在
mise 下的表现可能和在 `vfox` 下略有不同。本页说明两者如何配合使用。

::: info
本页描述的是 mise 提交
[`8e9f47c`](https://github.com/jdx/mise/tree/8e9f47c9c7166f3f6d92df0b7724ab2dbf16381b)
时的行为。mise 自身的行为以 [mise 文档](https://mise.jdx.dev/)为准。
:::

## 安装与使用

用 `mise use` 安装插件并选择版本，在 `vfox:` 前缀后写插件的 GitHub 仓库：

```shell
# 当前项目：写入 ./mise.toml
mise use vfox:version-fox/vfox-erlang@26.2.3

# 全局：写入 ~/.config/mise/config.toml
mise use -g vfox:version-fox/vfox-erlang@26.2.3

# 只运行一条命令，不修改任何配置
mise exec vfox:version-fox/vfox-erlang@26.2.3 -- erl -version
```

mise 第一次需要插件时会从 GitHub 安装它。`mise use` 会把工具写入配置文件的 `[tools]` 表：

```toml
[tools]
"vfox:version-fox/vfox-erlang" = "26.2.3"
```

`vfox:` 前缀总是选择这个插件。不带前缀的名称（例如 `erlang`）会在 mise 自己的注册表中查找，
一个工具可能对应多个后端，mise 会使用当前平台上第一个可用的后端。运行
`mise registry erlang` 可以查看一个名称对应哪些后端。要确保使用 vfox 插件，请写出它的
`vfox:` 标识。

## 与 vfox 的对比

| | vfox | mise + vfox 插件 |
|---|---|---|
| 添加插件 | `vfox add <name>` | 首次使用时自动安装，或 `mise plugins install` |
| 安装版本 | `vfox install <sdk>@<version>` | `mise install vfox:<owner>/<repo>@<version>` |
| 选择版本 | `vfox use -g/-p/-s` | `mise use`（项目）或 `mise use -g`（全局） |
| 运行命令 | `vfox exec` | `mise exec` |
| 列出已安装版本 | `vfox list` | `mise ls` |
| 列出可用版本 | `vfox search` | `mise ls-remote` |
| 删除版本 | `vfox uninstall` | `mise uninstall` |
| 作用域 | 全局、项目、会话 | 全局、项目，以及 `MISE_ENV` 环境；没有会话作用域 |
| 版本文件 | `.vfox.toml`、`vfox.toml`；两者都没有时读取 `.tool-versions`，并将其保存为 `.vfox.toml` | `mise.toml`、`.tool-versions` |
| 安装位置 | `$VFOX_HOME/cache/<sdk>/`，或 `storage.sdkPath` 下 | `~/.local/share/mise/installs/`；Windows 上为 `%LOCALAPPDATA%\mise\installs`（`$MISE_DATA_DIR/installs`） |
| 插件位置 | `$VFOX_HOME/plugin/<name>/` | `~/.local/share/mise/plugins/`；Windows 上为 `%LOCALAPPDATA%\mise\plugins`（`$MISE_DATA_DIR/plugins`） |
| `PATH` 设置 | `vfox activate <shell>` | `mise activate <shell>`、shims 或 `mise exec` |
| 插件来源 | vfox 注册表、URL 或本地文件 | `vfox:<owner>/<repo>`、Git URL、zip 或本地目录 |
| 平台 | Linux、macOS、Windows | Linux、macOS、Windows |

`VFOX_HOME` 默认为 `~/.vfox`；如果 `~/.version-fox` 目录存在，则为 `~/.version-fox`。

有些功能只有其中一个有：

- 只有 vfox 有：`vfox link` 和 `vfox link --detect`（使用机器上已有的 SDK）；
  `vfox doctor` 和 `diagnostics.onUse` 的插件检查；会话作用域（`vfox use -s`）。
- 只有 mise 有：锁文件（`mise.lock`）、任务、`mise.toml` 中的 `[env]`，以及管理多个工具的插件（后端插件）。

## 环境变量

mise 不读取 vfox 自己的任何变量。`VFOX_HOME` 不会改变 mise 的安装位置，`vfox activate`
设置的 `__VFOX_*` 变量（`__VFOX_SHELL`、`__VFOX_PID`、`__VFOX_ORIG_PATH`、`__VFOX_CURTMPPATH`，以及 PowerShell 中的 `__VFOX_INITIALIZED`）
对 mise 没有作用。插件用 `os.getenv` 读取它们时，只有在同一个 shell 中也运行了
`vfox activate` 才能读到。

插件在 mise 下得到的 Lua 全局变量与在 vfox 下相同：`RUNTIME.osType`、`RUNTIME.archType`、
`RUNTIME.version`、`RUNTIME.pluginDirPath`、`OS_TYPE`、`ARCH_TYPE` 和 `PLUGIN`。
在 mise 下，`RUNTIME.version` 是固定字符串 `0.6.0`，检查它的插件看到的是这个 vfox 版本号。
判断插件运行环境时，请优先使用这些全局变量，而不是环境变量。

在 mise 下有作用的变量：

- `MISE_DATA_DIR`：mise 插件和安装的根目录。mise 不会设置它，所以只有用户导出了它时，
  `PreInstall` 或 `EnvKeys` 等钩子才能读到。否则请回退到 `~/.local/share/mise`（Linux 和 macOS）
  或 `%LOCALAPPDATA%\mise`（Windows）。
- `MISE_GITHUB_TOKEN`、`GITHUB_API_TOKEN` 或 `GITHUB_TOKEN`：mise 会把第一个已设置的值加到插件发往
  GitHub API（`api.github.com`）的 `http` 请求中，以提高速率限制。下载 Release 文件时不会带上它。
- `MISE_TOOL_OPTS__<KEY>`：`mise.toml` 中的每个工具选项，插件运行的命令可以读取。

## 配置

mise 不读取 `$HOME/.vfox/config.yaml`、`.vfox.toml` 或 `vfox.toml`。请在 mise 中设置
`config.yaml` 各部分对应的内容：

| `config.yaml` | 在 mise 下 |
|---|---|
| `legacyVersionFile` | 默认关闭。用 mise 的 [`idiomatic_version_file_enable_tools`](https://mise.jdx.dev/configuration/settings.html#idiomatic_version_file_enable_tools) 按工具开启，之后 mise 会运行插件的 `ParseLegacyFile` 钩子。没有 `strategy` 设置。 |
| `proxy` | 不读取。请在运行 mise 的环境中设置 `HTTPS_PROXY` / `HTTP_PROXY`。 |
| `plugin.http.timeout`、`plugin.http.downloadTimeout` | 不读取，mise 也没有对应的设置：它的 vfox 运行时不会给插件的 `http` 请求设置超时。mise 用 [`fetch_remote_versions_timeout`](https://mise.jdx.dev/configuration/settings.html#fetch_remote_versions_timeout) 限制整个 `Available` 钩子的运行时间。 |
| `storage.sdkPath` | 不读取。安装位于 `$MISE_DATA_DIR/installs` 下。 |
| `registry.address`、`github.baseUrl` | 不读取：mise 从你写出的仓库安装插件，而不是从 vfox 注册表。要使用镜像，请设置 mise 的 [`url_replacements`](https://mise.jdx.dev/url-replacements.html)。 |
| `cache.availableHookDuration` | 不读取。mise 的 [`fetch_remote_versions_cache`](https://mise.jdx.dev/configuration/settings.html#fetch_remote_versions_cache) 决定 `Available` 返回的版本列表保留多久。 |
| `gitignore` | 不适用：mise 不会在项目中创建 `.vfox/` 目录。 |
| `diagnostics` | 不适用。`mise doctor` 检查的是 mise 自身的设置。 |
| `path.mode`、`path.preserve` | 对 `mise activate` 不适用。最接近的设置是 [`activate_aggressive`](https://mise.jdx.dev/configuration/settings.html#activate_aggressive)。 |
| `aliases` | 不读取。请使用 mise 的 [`[tool_alias]`](https://mise.jdx.dev/dev-tools/aliases.html)。 |

你用 `vfox use` 在 `.vfox.toml` 和 `vfox.toml` 中选择的版本同样不会被读取。请用 `mise use`
重新选择，它会写入 `mise.toml`。

## 插件钩子

对于按[插件开发指南](../plugins/create/howto.md)编写的插件，mise 会运行以下钩子：

| 钩子 | 在 mise 下 |
|---|---|
| `Available` | 运行，用于列出版本。 |
| `PreInstall` | 运行，mise 会下载它返回的 URL。 |
| `PostInstall` | 下载后运行。 |
| `EnvKeys` | 运行。mise 把其中的 `PATH` 条目加入 `PATH`，并设置其他变量。 |
| `ParseLegacyFile` | 只对 `idiomatic_version_file_enable_tools` 中列出的工具运行。 |
| `PreUninstall` | 在 mise 删除版本之前运行。 |
| `PreUse` | **不运行。** |
| `Fingerprint` | **不运行**；mise 没有这个钩子。 |
| `Detect` | **不运行**；mise 没有 `vfox link --detect`。 |

给插件作者的建议：

- 不依赖 `PreUse` 也要让插件的结果正确。如果插件用 `PreUse` 把 `21` 这样的部分版本补全为完整版本，
  也应在 `Available` 中返回完整版本，这样 mise 能自己解析 `21`。
- `Fingerprint` 和 `Detect` 只是在 vfox 下增加功能，所以实现了它们的插件在 mise 下照样可以安装。
- mise 还会运行 vfox 不认识的钩子：用于一个插件管理多个工具的 `Backend*` 钩子；包管理器插件的
  `PackageInstalled`、`PackageInstall`、`PackageUpgrade` 和 `PackageUninstall`；设置插件的
  `SettingsRead` 和 `SettingsWrite`；以及 `MiseEnv`、`MisePath` 和 `MiseInstallSatisfied`。它们在 mise 的
  [工具插件](https://mise.jdx.dev/tool-plugin-development.html)、
  [后端插件](https://mise.jdx.dev/backend-plugin-development.html)、
  [包插件](https://mise.jdx.dev/package-plugin-development.html)和
  [设置插件](https://mise.jdx.dev/settings-plugin-development.html)文档中有说明。vfox 会忽略它们。
- 要在 mise 下检查插件，可以用 `mise plugins link <name> <path>` 从本地目录安装，再运行
  `mise install <name>@<version>`。

## 在同一个 shell 中同时使用

可以在同一个 shell 启动文件中同时写 `vfox activate` 和 `mise activate`，但两者都会把自己的目录放到
`PATH` 前面。对于两者都管理的工具，当前提示符下最后运行钩子的那一个生效，所以结果可能每次提示符都不同。
请让每个工具只由其中一个管理。

设置相同变量的插件也有同样的问题。例如 Go 插件会设置 `GOROOT`，如果在 vfox 和 mise 中都选择了 Go，
shell 中的 `go` 可能来自一个安装，而 `GOROOT` 来自另一个。
[vfox-golang 故障排除页面](https://github.com/version-fox/vfox-golang/blob/main/docs/troubleshooting.md)
说明了这种情况。

### 安装位置与删除方法

mise 和 vfox 把安装放在不同的目录中，mise 看不到 vfox 安装的版本。但 vfox 可以使用 mise 安装的版本：
`vfox link` 可以指向它的目录，插件在 mise 的目录中查找时，`vfox link --detect` 也能找到它，
[vfox-deno](https://github.com/version-fox/vfox-deno/blob/main/docs/link.md) 就是这样做的。
如果之后用 `mise uninstall` 删除了这个版本，vfox 的链接会指向一个不存在的目录；请用 `vfox uninstall` 删除该链接。

- 用 `mise uninstall vfox:<owner>/<repo>@<version>` 从 mise 删除一个版本。
- 用 `mise plugins uninstall <name>` 从 mise 删除插件；运行 `mise plugins ls` 查看插件名称。
- `mise prune` 会删除不再被任何配置文件选择的版本。

要删除 vfox 安装的内容，请参阅[卸载](./uninstallation.md)。
