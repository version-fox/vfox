# Core

`vfox` has a few core commands that are used frequently. The following are the most frequently used commands.

## Available

View all available plugins. 

**Usage**

```shell
vfox available 
```

## Add

Add a plugin from the official repository or a custom source. In `vfox`, a plugin is an SDK, and an SDK is a plugin.
Therefore, before using them, you need to install the corresponding plugin.

**Usage**

```shell
vfox add [options] <plugin-name>
```

`plugin-name`: Plugin name, such as `nodejs`.

**Options**

- `-a, --alias`: Set the plugin alias.
- `-s, --source`: Install the plugin from the specified path (can be a remote file or a local file).

**Examples**

**Install plugin from the official repository**

```shell
$ vfox add --alias node nodejs
```

**Install custom plugin**

```shell
$ vfox add --source https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip custom-node
```

## Search

View for all available versions of the specified SDK.

**Usage**

```shell
vfox search <sdk-name> [...optionArgs]
```

`sdk-name`: SDK name, such as `nodejs`, `custom-node`.
`optionArgs`: Additional arguments for the search command. NOTE: Whether it is supported or not depends on the plugin.

::: tip Quick install
Select the target version, and press Enter to install quickly.
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
