# 所有命令


## List

查看当前已安装的所有SDK版本。

**用法**

```shell
vfox list [<sdk-name>]

vfox ls[<sdk-name>]
```

`sdk-name`: SDK名称， 不传展示所有。

## Current

查看当前SDK的版本。


**用法**

```shell
vfox current [<sdk-name>]
vfox c
```



## Info

查看本地安装的SDK信息。

**用法**

```shell
vfox info <sdk-name>
```

## Remove

删除本地安装的插件。

**用法**

```shell
vfox remove <sdk-name>
```

::: danger 注意
删除插件，`vfox`会同步删除当前插件安装的所有版本运行时。
:::



## Update

更新插件版本。

**用法**

```shell
vfox update <sdk-name>
```



## 概览

```shell
vfox - VersionFox, a tool for sdk version management
vfox available [<category>]     List all available plugins
vfox add [--alias <sdk-name> --source <url/path> ] <plugin-name>  Add a plugin from offical repository or custom source
vfox remove <sdk-name>          Remove a plugin
vfox update <sdk-name>          Update a plugin
vfox info <sdk-name>            Show plugin info
vfox search <sdk-name>          Search available versions of a SDK
vfox install <sdk-name>@<version> Install the specified version of SDK
vfox uninstall <sdk-name>@<version> Uninstall the specified version of SDK
vfox use [--global --project --session] <sdk-name>[@<version>]   Use the specified version of SDK for different scope
vfox list [<sdk-name>]              List all installed versions of SDK
vfox current [<sdk-name>]           Show the current version of SDK
vfox help                      Show this help message
```