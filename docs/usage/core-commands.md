# Core

`vfox` has a few core commands that are used frequently. The following are the most frequently used commands.

## Available

View all available plugins. If you specify a category, it will only show the plugins in that category. If you don't
specify a category, it will show all available plugins.

**Usage**

```shell
vfox available [<category>]
```

`category`[optional]: Category, such as java, nodejs, flutter, etc. If not passed, all available plugins will be
displayed.

::: warning
Note that the plugin name consists of two parts, `category/plugin-name`, such as `nodejs/nodejs`, `java/graalvm`
:::

## Add

Add a plugin from the official repository or a custom source. In `vfox`, a plugin is an SDK, and an SDK is a plugin.
Therefore, before using them, you need to install the corresponding plugin.

**Usage**

```shell
vfox add [options] <plugin-name>
```

`plugin-name`: Plugin name, such as `nodejs/nodejs`.

**Options**

- `-a, --alias`: Set the plugin alias.
- `-s, --source`: Install the plugin from the specified path (can be a remote file or a local file).

**Examples**

**Install plugin from the official repository**

```shell
$ vfox add --alias node nodejs/nodejs
```

**Install custom plugin**

```shell
$ vfox add --source https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/nodejs/nodejs.lua custom-node
```

## Search

View for all available versions of the specified SDK.

**Usage**

```shell
vfox search <sdk-name>
```

`sdk-name`: SDK name, such as `nodejs`, `custom-node`.

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
