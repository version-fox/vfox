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

- `-a, --all`: Install all SDK versions recorded in .tool-versions
- `-y, --yes`: Quick installation, skip interactive prompts​

::: tip
You can install multiple SDKs at the same time by separating them with space.

```shell
vfox install nodejs@20 golang ...
```

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

## Upgrade <Badge type="tip" text=">= 0.4.2" vertical="middle" />

Upgrade `vfox` to the latest version.

**Usage**

```shell
vfox upgrade
```