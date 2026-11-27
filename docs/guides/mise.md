# Using vfox Plugins with mise

[mise](https://mise.jdx.dev/) can install and run vfox plugins through its
[`vfox` backend](https://mise.jdx.dev/dev-tools/backends/vfox.html). mise does
not call the `vfox` binary for this: it has its own Rust implementation of the
plugin runtime, so a plugin can behave a little differently under mise than
under `vfox`. This page explains how the two fit together.

::: info
This page describes mise as of commit
[`8e9f47c`](https://github.com/jdx/mise/tree/8e9f47c9c7166f3f6d92df0b7724ab2dbf16381b).
For anything mise-specific, the [mise documentation](https://mise.jdx.dev/) is
the reference.
:::

## Setup

Install a plugin and select a version with `mise use`, naming the plugin's
GitHub repository after the `vfox:` prefix:

```shell
# Current project: writes to ./mise.toml
mise use vfox:version-fox/vfox-erlang@26.2.3

# Global: writes to ~/.config/mise/config.toml
mise use -g vfox:version-fox/vfox-erlang@26.2.3

# Run one command without changing any config
mise exec vfox:version-fox/vfox-erlang@26.2.3 -- erl -version
```

mise installs the plugin from GitHub the first time it needs it. `mise use`
writes the tool to the `[tools]` table of the config file:

```toml
[tools]
"vfox:version-fox/vfox-erlang" = "26.2.3"
```

The `vfox:` prefix always selects that plugin. A plain name such as `erlang`
is looked up in mise's own registry instead, which can list several backends
for one tool, and mise uses the first one available on your platform. Run
`mise registry erlang` to see which backends a name maps to. To be sure you get
a vfox plugin, write its `vfox:` identifier.

## Comparison with vfox

| | vfox | mise with a vfox plugin |
|---|---|---|
| Add a plugin | `vfox add <name>` | automatic on first use, or `mise plugins install` |
| Install a version | `vfox install <sdk>@<version>` | `mise install vfox:<owner>/<repo>@<version>` |
| Select a version | `vfox use -g/-p/-s` | `mise use` (project) or `mise use -g` (global) |
| Run a command | `vfox exec` | `mise exec` |
| List installed versions | `vfox list` | `mise ls` |
| List available versions | `vfox search` | `mise ls-remote` |
| Remove a version | `vfox uninstall` | `mise uninstall` |
| Scopes | global, project, session | global, project, plus `MISE_ENV` environments; no session scope |
| Version files | `.vfox.toml`, `vfox.toml`; without either, `.tool-versions`, which vfox then saves as `.vfox.toml` | `mise.toml`, `.tool-versions` |
| Installs | `$HOME/.vfox/cache/<sdk>/`, or under `storage.sdkPath` | `~/.local/share/mise/installs/` (`$MISE_DATA_DIR/installs`) |
| Plugins | `$HOME/.vfox/plugin/<name>/` | `~/.local/share/mise/plugins/` (`$MISE_DATA_DIR/plugins`) |
| `PATH` setup | `vfox activate <shell>` | `mise activate <shell>`, shims, or `mise exec` |
| Plugin sources | the vfox registry, a URL or a local file | `vfox:<owner>/<repo>`, a Git URL, a zip or a local directory |
| Platforms | Linux, macOS, Windows | Linux, macOS, Windows |

Some features exist in only one of them:

- Only in vfox: `vfox link` and `vfox link --detect`, which use an SDK already
  on your machine; the plugin diagnostics of `vfox doctor` and of
  `diagnostics.onUse`; and the session scope (`vfox use -s`).
- Only in mise: lockfiles (`mise.lock`), tasks, `[env]` in `mise.toml`, and
  plugins that manage several tools (backend plugins).

## Environment Variables

mise reads none of vfox's own variables. `VFOX_HOME` does not move mise's
installs, and the `__VFOX_*` variables that `vfox activate` sets
(`__VFOX_SHELL`, `__VFOX_PID`, `__VFOX_ORIG_PATH`, `__VFOX_CURTMPPATH`, and
`__VFOX_INITIALIZED` in PowerShell) have no
effect on mise. A plugin that reads them with `os.getenv` sees them only if
`vfox activate` also runs in the same shell.

Plugins get the same Lua globals under mise as under vfox: `RUNTIME.osType`,
`RUNTIME.archType`, `RUNTIME.version`, `RUNTIME.pluginDirPath`, `OS_TYPE`,
`ARCH_TYPE` and `PLUGIN`. `RUNTIME.version` is the version of the runtime that
runs the plugin, so it is not a vfox version under mise. Prefer these globals
over environment variables to find out where the plugin runs.

Variables that matter under mise:

- `MISE_DATA_DIR`: the root of mise's plugins and installs. A hook that mise
  runs, such as `PreInstall` or `EnvKeys`, can read it to find mise's
  directories.
- `MISE_GITHUB_TOKEN`, `GITHUB_API_TOKEN` or `GITHUB_TOKEN`: mise adds the
  first one that is set to the plugin's `http` requests to the GitHub API
  (`api.github.com`), which raises the rate limit. Release downloads are sent
  without it.
- `MISE_TOOL_OPTS__<KEY>`: each tool option from `mise.toml`, set for the
  commands that the plugin runs.

## Configuration

mise does not read `$HOME/.vfox/config.yaml`, `.vfox.toml` or `vfox.toml`. Set
each `config.yaml` section in mise instead:

| `config.yaml` | Under mise |
|---|---|
| `legacyVersionFile` | Off by default. Turn it on per tool with mise's [`idiomatic_version_file_enable_tools`](https://mise.jdx.dev/configuration/settings.html#idiomatic_version_file_enable_tools); mise then runs the plugin's `ParseLegacyFile` hook. There is no `strategy` setting. |
| `proxy` | Not read. Set `HTTPS_PROXY` / `HTTP_PROXY` in the environment that runs mise. |
| `plugin.http.timeout`, `plugin.http.downloadTimeout` | Not read, and mise has no setting that replaces them: its vfox runtime sets no timeout on the plugin's `http` requests. mise limits the whole `Available` hook with [`fetch_remote_versions_timeout`](https://mise.jdx.dev/configuration/settings.html#fetch_remote_versions_timeout). |
| `storage.sdkPath` | Not read. Installs go under `$MISE_DATA_DIR/installs`. |
| `registry.address`, `github.baseUrl` | Not read: mise installs plugins from the repository you name, not from the vfox registry. To use a mirror, set mise's [`url_replacements`](https://mise.jdx.dev/url-replacements.html). |
| `cache.availableHookDuration` | Not read. mise's [`fetch_remote_versions_cache`](https://mise.jdx.dev/configuration/settings.html#fetch_remote_versions_cache) sets how long the version list from `Available` is kept. |
| `gitignore` | Does not apply: mise does not create a `.vfox/` directory in the project. |
| `diagnostics` | Does not apply. `mise doctor` checks mise's own setup. |
| `path.mode`, `path.preserve` | Do not apply to `mise activate`. Its closest setting is [`activate_aggressive`](https://mise.jdx.dev/configuration/settings.html#activate_aggressive). |
| `aliases` | Not read. Use mise's [`[tool_alias]`](https://mise.jdx.dev/dev-tools/aliases.html). |

The versions you selected with `vfox use` in `.vfox.toml` and `vfox.toml` are
not read either. Select them again with `mise use`, which writes `mise.toml`.

## Plugin Hooks

For a plugin written with the [plugin guide](../plugins/create/howto.md),
mise runs these hooks:

| Hook | Under mise |
|---|---|
| `Available` | Runs, to list versions. |
| `PreInstall` | Runs, and mise downloads the URL it returns. |
| `PostInstall` | Runs after the download. |
| `EnvKeys` | Runs. mise puts the `PATH` entries on `PATH` and sets the other variables. |
| `ParseLegacyFile` | Runs only for a tool listed in `idiomatic_version_file_enable_tools`. |
| `PreUninstall` | Runs before mise removes a version. |
| `PreUse` | **Does not run.** |
| `Fingerprint` | **Does not run**; mise has no such hook. |
| `Detect` | **Does not run**; mise has no `vfox link --detect`. |

For plugin authors:

- Keep the result of a plugin correct without `PreUse`. A plugin that uses
  `PreUse` to turn a partial version such as `21` into a full one should also
  return full versions from `Available`, so mise can resolve `21` itself.
- `Fingerprint` and `Detect` only add to what the plugin does under vfox, so a
  plugin that has them still installs under mise.
- mise also runs hooks of its own that vfox does not know about: `Backend*`
  hooks for one plugin that manages several tools; `PackageInstalled`,
  `PackageInstall`, `PackageUpgrade` and `PackageUninstall` for package
  manager plugins; `SettingsRead` and `SettingsWrite` for settings plugins; and
  `MiseEnv`, `MisePath` and `MiseInstallSatisfied`. They are documented in
  mise's [tool plugin](https://mise.jdx.dev/tool-plugin-development.html),
  [backend plugin](https://mise.jdx.dev/backend-plugin-development.html),
  [package plugin](https://mise.jdx.dev/package-plugin-development.html) and
  [settings plugin](https://mise.jdx.dev/settings-plugin-development.html)
  guides. vfox ignores them.
- To check a plugin under mise, install it from a local directory with
  `mise plugins link <name> <path>` and run `mise install <name>@<version>`.

## Using Both in One Shell

You can have `vfox activate` and `mise activate` in the same shell startup
file, but each puts its own directories at the front of `PATH`. The one whose
hook ran last for the current prompt wins for a tool that both manage, so the
result can change from one prompt to the next. Manage each tool with one of
the two.

Plugins that set the same variables have the same problem. For example, the
Go plugin sets `GOROOT`, and with Go selected in both vfox and mise the shell
can end up with `go` from one install and `GOROOT` from the other. The
[vfox-golang troubleshooting page](https://github.com/version-fox/vfox-golang/blob/main/docs/troubleshooting.md)
explains this case.

### Where installs are and how to remove them

mise and vfox keep their installs in separate directories, and mise does not
see the versions that vfox installed. vfox can use a version that mise
installed, though: `vfox link` points at its directory, and
`vfox link --detect` finds it when the plugin looks in mise's directories, as
[vfox-deno](https://github.com/version-fox/vfox-deno/blob/main/docs/link.md)
does. If you then remove that version with `mise uninstall`, the vfox link
points at a directory that no longer exists; remove it with `vfox uninstall`.

- Remove one version from mise with
  `mise uninstall vfox:<owner>/<repo>@<version>`.
- Remove the plugin from mise with `mise plugins uninstall <name>`; run
  `mise plugins ls` to see its name.
- `mise prune` removes the versions that no config file selects any more.

To remove what vfox installed, see [Uninstallation](./uninstallation.md).
