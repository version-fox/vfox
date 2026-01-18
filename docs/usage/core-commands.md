# Core

## Search

View for all available versions of the specified SDK.

**Usage**

```shell
vfox search <sdk-name> [...optionArgs]
```

`sdk-name`: SDK name, such as `nodejs`, `custom-node`.
`optionArgs`: Additional arguments for the search command. NOTE: Whether it is supported or not depends on the plugin.

**Options**

- `--skip-cache`: Skip reading and writing the available cache for this search.

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
After using `unuse`, the SDK will no longer be active in the specified scope. If the SDK is configured in other scopes, those will take precedence according to vfox's scope hierarchy (Session > Project > Global).
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

## List

View all installed sdks.

**Usage**

```shell
vfox list [<sdk-name>]

vfox ls [<sdk-name>]
```

`sdk-name`: SDK name, if not passed, display all.

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

**Usage**

```shell
vfox upgrade
```

## Exec <Badge type="tip" text=">= 1.0.0" vertical="middle" />

Execute a command in a vfox managed environment.

**Usage**

```shell
vfox exec <sdk-name>[@<version>] -- <command> [args...]

vfox x <sdk-name>[@<version>] -- <command> [args...]
```

`sdk-name`: SDK name

`version`[optional]: Specify the version to use. If not provided, uses the version configured in the current scope.

`command`: The command to execute

`args`: Arguments to pass to the command

**Description**

The `exec` command allows you to temporarily execute commands in a specified SDK environment without modifying your current scope configuration. This is particularly useful for:

- **IDE Integration**: Use project-specific SDK versions in IDEs like VS Code
- **Script Execution**: Use specific SDK versions in CI/CD or build scripts
- **Temporary Testing**: Test code with different SDK versions

**Examples**

```shell
# Execute command with specified version
vfox exec nodejs@20.9.0 -- node -v

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
