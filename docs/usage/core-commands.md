# Core

## Search

View for all available versions of the specified SDK.

**Usage**

```shell
vfox search <sdk-name> [...optionArgs]
```

`sdk-name`: SDK name, such as `nodejs`, `custom-node`.
`optionArgs`: Additional arguments for the search command. NOTE: Whether it is supported or not depends on the plugin.

::: warning Cache

`vfox` will cache the results of the `search` command to reduce the number of network requests. The default cache time is `12h`.

You can disable it through the following command.
```shell
vfox config cache.availableHookDuration 0
```
For details, see [Cache Settings](../guides/configuration.md#cache-settings).
:::

::: tip Quick install
Select the target version, and press Enter to install quickly.
:::

::: tip
`search` command will retrieve the plugin from the remote repository and add it locally if it is not installed locally.
:::

## Install

Install the specified SDK version to your computer and cache it for future use.

**Usage**

```shell
vfox install <sdk-name>@<version>

vfox i <sdk-name>@<version>
```

`sdk-name`: SDK name

`version`: The version to install

**Options**

- `-a, --all`: Install all SDK versions recorded in .vfox.toml
- `-y, --yes`: Quick installation, skip interactive prompts​
- `-u, --use`: Use the installed version in the current shell session, as `vfox use` would, so it needs an activated shell (`vfox activate`) the same way. Cannot be combined with `--all`, takes a version (no picker) and one version per SDK.
- `--reinstall`: Replace an installed version with a fresh copy, for example after the SDK modified itself in place (see [Doctor](#doctor)). Give the version exactly as it is installed. A version registered with [`vfox link`](#link) cannot be reinstalled.

::: tip
You can install multiple SDKs at the same time by separating them with space.

```shell
vfox install nodejs@20 golang ...
```

:::

::: tip
Quick installation, skip interactive prompts​​

```shell
vfox install --yes nodejs@20
vfox install --yes --all
```

:::

## Use

Set the runtime version.

**Usage**

```shell
vfox use [options] <sdk-name>[@<version>]

vfox u [options] <sdk-name>[@<version>]
```

`sdk-name`: SDK name

`version`[optional]: Use the specified version of the runtime. If not passed, you can select it from the list.

**Options**

- `-g, --global`: Effective globally
- `-p, --project`: Effective in the current directory(`$PWD`)
- `-s, --session`: Effective within the current Shell session

::: tip Default scope

`Windows`: `Global` scope

`Unix-like`: `Session` scope
:::

## Unuse

Unset the runtime version from a specific scope.

**Usage**

```shell
vfox unuse [options] <sdk-name>
```

`sdk-name`: SDK name

**Options**

- `-g, --global`: Remove from global scope
- `-p, --project`: Remove from project scope (current directory)
- `-s, --session`: Remove from session scope (current Shell session)

::: tip Default scope

`Windows`: `Global` scope

`Unix-like`: `Session` scope
:::

::: warning Effect
After using `unuse`, the SDK will no longer be active in the specified scope. If the SDK is configured in other scopes, those will take precedence according to vfox's scope hierarchy (Project > Session > Global).
:::

## Uninstall

Uninstall the specified version of the SDK.

**Usage**

```shell
vfox uninstall <sdk-name>@<version>
vfox un <sdk-name>@<version>
```

`sdk-name`: SDK name

`version`: The specific version number

::: tip Linked versions
Uninstalling a version registered with [`vfox link`](#link) removes the link only. The SDK it points to is left untouched.
:::

## List

View all installed sdks.

**Usage**

```shell
vfox list [<sdk-name>]

vfox ls [<sdk-name>]
```

`sdk-name`: SDK name, if not passed, display all.

Versions registered with [`vfox link`](#link) are marked `(linked → <path>)`, and versions that have an [alias](#alias) show it:

```shell
$ vfox list java
-> 17-system (linked → /usr/lib/jvm/java-17-openjdk-amd64)
-> 17.0.9-tem [alias: j17] <— current
```

## Link

Use an SDK that is already on your machine as a version of `vfox`. That can be a JDK from your system package manager, one an IDE downloaded, or one a CI runner ships in its tool cache.

Nothing is downloaded or copied. `vfox` keeps a link to the directory, and the SDK stays where it is and stays its owner's: `vfox uninstall` removes the link only.

**Usage**

```shell
vfox link [options] <sdk-name>@<version> <path>

vfox ln <sdk-name>@<version> <path>

vfox link --detect [-y] <sdk-name>
```

`sdk-name`: SDK name. Its plugin must be added already.

`version`: The name to give the SDK. It is used exactly as written, for example `17.0.9`, `17-system` or `brew`. Use letters, digits, `.`, `_`, `+` and `-`; `latest` is not allowed.

`path`: The directory of the SDK, the one that holds `bin`.

**Options**

- `-f, --force`: Point an already linked version to a different directory. A version that `vfox` installed itself is never replaced.
- `--detect`: Ask the plugin to look for SDKs on this machine and link what it finds. The plugin must implement the [`Detect`](../plugins/create/howto.md#detect) hook.
- `-y, --yes`: With `--detect`, link everything that was found without asking.

**Examples**

```shell
vfox link java@17-system /usr/lib/jvm/java-17-openjdk-amd64
vfox use java@17-system

# Nothing is downloaded for a version that is linked
vfox install java@17-system

# Where is it?
vfox info --format "{{.Home}}" java@17-system

# Removes the link, not the JDK
vfox uninstall java@17-system
```

::: tip The plugin has a say
`vfox link` asks the plugin whether the directory looks like one of its SDKs: the directories the plugin puts on `PATH` must exist and hold commands. Otherwise the link is refused and says why.
:::

## Alias

Give an installed version a short name of your own.

**Usage**

```shell
vfox alias set <sdk-name> <alias> <version>
vfox alias list [<sdk-name>]
vfox alias unset <sdk-name> <alias>
```

**Examples**

```shell
vfox alias set java j17 17.0.9-tem
vfox use java@j17
```

::: tip Aliases stay on your machine
Aliases are stored in your `config.yaml` and are resolved only for what you type on the command line: `use`, `uninstall`, `info`, `exec` and `doctor`. `vfox use -p java@j17` writes the real version into `.vfox.toml`, so the file keeps working for teammates who do not have the alias.
:::

A version that is installed under the same name always wins over an alias, so `alias set` refuses such a name. Uninstalling a version removes the aliases that name it.

A leading `v` in a version you type (`v20.1.0` for `20.1.0`) is dropped only when no installed version and no alias is called what you typed. So an alias, like a [linked](#link) version, may start with a `v`.

## Doctor

Check `vfox` and the installed SDKs for problems.

**Usage**

```shell
vfox doctor [options] [<sdk-name>[@<version>]]
```

**Options**

- `--json`: Output the findings as JSON.
- `--adopt`: Write a receipt for versions installed before `vfox` kept them, taking their current state as the one to compare against later.

`vfox` writes a small receipt (`.vfox-receipt.json`) into the directory of every version it installs or links. It records where the version came from and, if the plugin implements [`Fingerprint`](../plugins/create/howto.md#fingerprint), the facts that identify it.

| Finding                   | Level   | Meaning                                                                                                                  |
| :------------------------ | :------ | :----------------------------------------------------------------------------------------------------------------------- |
| `path.shadowed`           | warn    | A command of the SDK is not the one that runs, because another executable sits earlier in `PATH` (a pnpm-installed `node`, a system JDK …) |
| `path.not-activated`      | error   | `vfox activate` is not set up in this shell                                                                              |
| `sdk.drift`               | warn    | The SDK was modified in place since it was installed (for example `flutter upgrade`). Fix: `vfox install --reinstall <sdk>@<version>` |
| `sdk.link-dangling`       | error   | The directory a linked version points to is gone                                                                         |
| `sdk.payload-empty`       | error   | The directories the plugin puts on `PATH` do not exist or hold no command                                                |
| `sdk.envkeys-failed`      | error   | The plugin could not say what to put in the environment                                                                  |
| `sdk.install-interrupted` | warn    | An installation was interrupted                                                                                          |
| `sdk.no-receipt`          | info    | Installed before receipts existed, so changes to it cannot be detected (`vfox doctor --adopt`)                            |
| `sdk.plugin-missing`      | warn    | Versions are installed but the plugin is not added                                                                       |
| `plugin.incompatible`     | warn    | The plugin needs a newer `vfox`                                                                                          |
| `alias.dangling`          | warn    | An alias names a version that is not installed                                                                           |

`path.shadowed` is checked for the version your shell uses, which is the one `vfox env` puts on `PATH`: the first installed version among those your project, session and global settings ask for. Name a version, as in `vfox doctor java@17`, to check that one instead. A command that is the very file the SDK provides, such as a symlink to a [linked](#link) installation, is not a shadow.

The exit status is `1` when an error was found.

After `vfox use`, `vfox` runs the fast checks for the version it just selected by itself and prints the warnings. See [Diagnostics Settings](../guides/configuration.md#diagnostics-settings) to switch that off or to silence single findings.

## Current

View the current version of the SDK.

**Usage**

```shell
vfox current [<sdk-name>]
vfox c
```

## Cd 

Launch a shell in the `VFOX_HOME` or SDK directory.

**Usage**

```shell
vfox cd [options] [<sdk-name>]
```

`sdk-name`: SDK name, if not passed, default `VFOX_HOME`.

**Options**

- `-p, --plugin`: Launch a shell in the plugin directory.

## Upgrade

Upgrade `vfox` to the latest version.

On Linux and macOS, the upgrade replaces the `vfox` binary in place, so it needs write access to the directory that holds it. If `vfox` is installed in a system directory such as `/usr/local/bin`, run it with `sudo`, for example `sudo /usr/local/bin/vfox upgrade`.

**Usage**

```shell
vfox upgrade
```

## Exec <Badge type="tip" text=">= 1.0.0" vertical="middle" />

Execute a command in a vfox managed environment.

**Usage**

```shell
vfox exec <sdk-name>[@<version>]... -- <command> [args...]

vfox x <sdk-name>[@<version>]... -- <command> [args...]
```

`sdk-name`: SDK name

`version`[optional]: Specify the version to use. If not provided, uses the version configured in the current scope.

You can specify multiple SDKs before `--`. They are merged from left to right, and the leftmost SDK has higher priority when PATH or environment variables overlap.

`command`: The command to execute

`args`: Arguments to pass to the command

**Description**

The `exec` command allows you to temporarily execute commands in a specified SDK environment without modifying your current scope configuration. This is particularly useful for:

- **IDE Integration**: Use project-specific SDK versions in IDEs like VS Code
- **Script Execution**: Use specific SDK versions in CI/CD or build scripts
- **Temporary Testing**: Test code with different SDK versions

::: tip Non-Interactive Environments

For Docker, CI/CD, and other non-interactive shells, prefer `vfox exec` instead of relying on `vfox activate`.

`activate` integrates vfox through shell hooks, while `exec` prepares the SDK environment explicitly for the subprocess you are running.

```shell
vfox exec nodejs@24.14.0 -- npm install -g pnpm
vfox exec nodejs@24.14.0 -- bash -lc 'node -v && npm -v'
```

:::

**Examples**

```shell
# Execute command with specified version
vfox exec nodejs@20.9.0 -- node -v

# Execute command with multiple SDKs
vfox exec nodejs@24.14.0 golang@1.25.6 -- npm install -g @qwen-code/qwen-code@latest

# Run build in maven environment
vfox exec maven@3.9.1 -- mvn clean install

# Use alias x (short for exec)
vfox x maven@3.9.1 -- mvn clean

```

::: tip IDE Integration

In VS Code, you can use the `exec` command to ensure your project uses the correct SDK version. For example, configure in `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Run with Node.js",
      "type": "shell",
      "command": "vfox",
      "args": ["x", "nodejs@20", "--", "node", "${file}"]
    }
  ]
}
```

:::

::: tip Auto Install

If the specified version is not installed, `exec` will automatically install it.

:::

::: warning Environment Variables

The `exec` command sets the correct environment variables (such as PATH, JAVA_HOME, etc.) in a subprocess, but does not affect your current Shell session.

:::
