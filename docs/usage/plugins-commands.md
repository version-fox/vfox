# Plugins

Plugins are how `vfox` knows to handle different tools like `Node.js`, `Java`, `Elixir` etc.

See [Creating Plugins](../plugins/create/howto.md) for the plugin API used to support more tools.

## Available

View all available plugins.

**Usage**

```shell
vfox available 
```

## Add
Add a plugin from the official repository or a custom source. 

**Usage**

```shell
vfox add [options] <plugin-name> [<plugin-name2>...]
```

`plugin-name`: Plugin name, such as `nodejs`. You can install multiple plugins at once, separated by spaces.

**Options**

- `-a, --alias`: Set the plugin alias.
- `-s, --source`: Install the plugin from the specified path (can be a remote file or a local file).


::: warning
`--alias` and `--source` are not supported when adding multiple plugins.
:::

**Examples**

**Install plugin from the official repository**

```shell
$ vfox add --alias node nodejs

$ vfox add golang java nodejs
```

**Install custom plugin**

```shell
$ vfox add --source https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip custom-node
```

## Info

View the SDK information installed locally.

**Usage**

```shell
vfox info <plugin-name>
vfox info <plugin-name>@<version>
vfox info [options] <plugin-name>
```

`plugin-name`: Plugin name, such as `nodejs`.
`version`: Specific version of the plugin.

**Options**

- `-f, --format`: Format the output using the given Go template. Available fields:
  - For plugin info: `Name`, `Version`, `Homepage`, `InstallPath`, `Description`
  - For version info: `Name`, `Version`, `Path`

**Examples**

**View plugin information**
```shell
vfox info nodejs
```

**View specific version path**
```shell
vfox info nodejs@20.0.0
```

**Format output with template**
```shell
vfox info --format "{{.Homepage}}" nodejs
vfox info --format "{{.InstallPath}}" nodejs
vfox info --format "{{.Path}}" nodejs@20.0.0
```

## Remove

Remove the installed plugin.

**Usage**

```shell
vfox remove <plugin-name>
```

::: danger
`vfox` will remove all versions of the runtime installed by the current plugin.
:::



## Update

Update a specified or all plugin(s)

**Usage**

```shell
vfox update <plugin-name>
vfox update --all # update all installed plugins
```

