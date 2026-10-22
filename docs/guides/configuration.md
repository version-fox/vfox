# Configuration

`vfox` allows you to change some configurations, all configuration is stored in the `$HOME/.vfox/config.yaml`
file.

::: tip
If you use `vfox` for the first time, an empty `config.yaml` file will be created automatically.
:::

## Legacy Version File <Badge type="tip" text=">= 0.4.0" vertical="middle" />

Plugins **with support** can read the versions files used by other version managers,
for example, `.nvmrc` in the case of Nodejs's `nvm`.

This capability is **turned on by default**. The related configuration options are as follows:

```yaml
legacyVersionFile:
  enable: true
  strategy: "specified" # Parsing strategy
```

- `enable`: Whether to enable legacy version file parsing functionality
- `strategy`: Parsing strategy, see strategy options below for details

### Strategy Options

`vfox` supports the following three parsing strategies:

- `latest_installed`: Use the latest installed version
- `latest_available`: Use the latest available version
- `specified`: Use the version specified in the legacy file (default)

::: warning

1. If both `.vfox.toml` and other version manager's configuration files (`.nvmrc`, `.sdkmanrc`, etc.) exist in the
   directory, `vfox` **priority read** the `.vfox.toml` file.
2. Enabling this feature may cause `vfox` to refresh environment variables slightly slower, **please enable it according
   to your needs**.

:::

If you want to disable this feature, you can use the command: `vfox config legacyVersionFile.enable false`

## Proxy Settings

::: tip
Currently only support http(s) proxy protocol
:::

**Format**: `http[s]://[username:password@]host:port`

```yaml
proxy:
  enable: false
  url: http://localhost:7890
```

## Storage Settings

By default, `vfox` stores SDK cache files in the `$HOME/.vfox/cache` directory.

::: danger !!!
Before configuring, please make sure that `vfox` has write permission to the folder.⚠⚠⚠
:::

```yaml
storage:
  sdkPath: /tmp
```

`storage.sdkPath` is the root directory used to store SDKs. For example, if `sdkPath` is `/tmp`, Node.js will be installed under `/tmp/nodejs/`.

::: warning Compatibility Note
`v1.0.2` to `v1.0.7` incorrectly treated `storage.sdkPath` as a shared root and installed SDKs under `sdkPath/cache/<sdk>`.

This behavior has been corrected to match `v0.10.x`. Existing SDKs under the incorrect `cache` path are still temporarily supported. New installs use the correct layout `sdkPath/<sdk>`.

If both `sdkPath/<sdk>` and `sdkPath/cache/<sdk>` exist, `vfox` will prefer `sdkPath/<sdk>`.
:::

## Plugin Registry Address

`vfox` will default to retrieve plugins from [plugins registry](https://version-fox.github.io/vfox-plugins).

If you want to use **your own registry or third-party mirror registry**, please configure it following:

```yaml
registry:
  address: "https://version-fox.github.io/vfox-plugins"
```

::: tip Available Mirrors

- https://cdn.jsdelivr.net/gh/version-fox/vfox-plugins/plugins
- https://rawcdn.githack.com/version-fox/vfox-plugins/plugins
  :::

## Cache Settings

`vfox` will cache the results of the `search` command (`available` hook) by default to reduce the number of network requests. The default
cache time is `12h`.

::: warning Special Value

- `-1`: Never expire
- `0`: Do not cache
  :::

```yaml
cache:
  availableHookDuration: 12h # s second, m minute, h hour
```

::: tip Cache File Path
`$HOME/.vfox/plugin/<plugin-name>/.available.cache`
:::

## Gitignore Settings <Badge type="tip" text=">= 1.0.12" vertical="middle" />

When you run `vfox use <sdk>@<version> -p` (project scope), `vfox` creates a `.vfox/` directory at the project root and, by default, appends `.vfox/` to the project's existing `.gitignore` (it will not create a new `.gitignore` if one is absent).

If you manage `.vfox/` through a [global gitignore](https://git-scm.com/docs/gitignore) (`core.excludesfile`) and prefer that `vfox` not touch the project's `.gitignore`, you can disable this behavior:

```yaml
gitignore:
  enable: true # set to false to disable auto-updating .gitignore
```

To disable via command:

```shell
vfox config gitignore.enable false
```

## Diagnostics Settings

After `vfox use`, `vfox` checks the version it just selected and prints what deserves attention, for example that the
`node` you get is not the one you selected because another `node` sits earlier in `PATH`. These are the fast checks of
[`vfox doctor`](../usage/core-commands.md#doctor); they never make `vfox use` fail.

```yaml
diagnostics:
  onUse: true # set to false to stop running the checks after `vfox use`
  ignore: # silence findings: <finding-id>[:<sdk>[@<version>]]
    - "sdk.drift:flutter@3.47.0"
    - "path.shadowed:nodejs"
```

To switch the checks off via command:

```shell
vfox config diagnostics.onUse false
```

## PATH Order Settings

`vfox` puts its own directories at the front of `PATH`, but it leaves in front of them what was put there before it
ran, so a Python virtualenv you activated after `vfox` keeps winning. Not everything in front of `vfox` is a
virtualenv, though: macOS `path_helper` moves the system directories to the front of a login shell, the Windows
machine `PATH` comes before the user `PATH` that `vfox use -g` writes to, and `pnpm setup` appends a line to your shell
startup file that puts `$PNPM_HOME` first. With those in front, `node` or `java` keeps being the one from the system
(or from pnpm) no matter what `vfox use` says. `vfox doctor` tells you when this happens.

```yaml
path:
  mode: legacy # legacy | baseline | aggressive
  preserve: [] # directories that always stay in front of vfox's, e.g. ["~/.cargo/bin"]
```

- `legacy` (the default for now): everything in front of the first `vfox` directory stays in front of it.
- `baseline` (recommended): only what was added to `PATH` **after** `vfox activate` ran stays in front. What was already
  in `PATH` when `vfox` was activated, such as the system directories above, goes behind the `vfox` directories.
  `vfox activate` records that `PATH` in `__VFOX_ORIG_PATH`. A directory you rely on being in front of `vfox`'s can be
  listed in `preserve`.
- `aggressive`: only the directories of an active virtualenv or conda environment and the ones in `preserve` stay in front.

The directories of an active virtualenv or conda environment (`VIRTUAL_ENV`, `CONDA_PREFIX`) always stay in front.

```shell
vfox config path.mode baseline
```

::: warning ⚠️ The default will change
`baseline` is going to become the default of `path.mode` in a future release, so that the system directories above stop
winning without any setting. [Quick Start](./quick-start.md#_2-hook-vfox-to-your-shell) already asks new installations
to set it. To keep today's behavior after the change, set `legacy` explicitly: `vfox config path.mode legacy`.
:::

::: tip
A setting change applies at the next prompt of a shell. `baseline` also needs the `PATH` that `vfox activate` recorded
when that shell started; a shell that was started before `vfox` was upgraded has none and behaves like `legacy` until
you open a new terminal.
:::

## Aliases

Names you give to installed versions with [`vfox alias`](../usage/core-commands.md#alias) are stored in the same file.
They are resolved for what you type on the command line only, never in `.vfox.toml`. `vfox alias` changes the
`aliases` key and nothing else: the rest of your file stays as it is, so a section it does not set keeps following the
shared configuration.

```yaml
aliases:
  java:
    j17: 17.0.9-tem
    j21: 21.0.1-graal
```

## Config Command

Setup, view config

**Usage**

```shell
vfox config [<key>] [<value>]

vfox config proxy.enable true
vfox config proxy.url http://localhost:7890
vfox config storage.sdkPath /tmp
vfox config path.mode baseline
```

`key`: Configuration item, separated by `. `.
`value`: If not passed, look at the value of the configuration item.

**Options**

- `-l, --list`: list all config.
- `-un, --unset`: remove a config.
