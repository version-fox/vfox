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

## Info

View the SDK information installed locally.

**Usage**

```shell
vfox info <plugin-name>
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

