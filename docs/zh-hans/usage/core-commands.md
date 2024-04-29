# 核心命令

## Search

获取指定 SDK 所有可用的运行时版本。

**用法**

```shell
vfox search <sdk-name> [...optionArgs]
```

`sdk-name`: 运行时名称， 如`nodejs`、`custom-node`。
`optionArgs`: 搜索命令的附加参数。注意：是否支持取决于插件。

::: tip 快捷安装
选择目标版本， 回车即可快速安装。
:::

::: tip 自动安装
如果本地没有安装 SDK，`search`命令会从远端仓库检索插件并安装到本地。
:::

## Install

将指定的 SDK 版本安装到您的计算机并缓存以供将来使用。

**用法**

```shell
vfox install <sdk-name>@<version>

vfox i <sdk-name>@<version>
```

`sdk-name`: SDK 名称

`version`: 需要安装的版本号

**选项**

- `-a, --all`: 安装 .tool-versions 中记录的所有 SDK 版本

::: tip 自动安装
你可以一次性安装多个 SDK，通过空格分隔。

```shell
vfox install nodejs@20 golang ...
```
:::

## Use

设置运行时版本

**用法**

```shell
vfox use [options] <sdk-name>[@<version>]

vfox u [options] <sdk-name>[@<version>]
```

`sdk-name`: SDK 名称

`version`[可选]: 使用 指定版本运行时。如不传， 则下拉选择。

**选项**

- `-g, --global`: 全局生效
- `-p, --project`: 当前目录下生效
- `-s, --session`: 当前 Shell 会话内生效

::: tip 默认作用域
`Windows`: 默认`Global`作用域

`Unix-like`: 默认`Session`作用域

:::

## Uninstall

卸载指定版本的 SDK。

**用法**

```shell
vfox uninstall <sdk-name>@<version>
vfox un <sdk-name>@<version>
```

`sdk-name`: SDK 名

`version`: 具体版本号

## List

查看当前已安装的所有 SDK 版本。

**用法**

```shell
vfox list [<sdk-name>]

vfox ls[<sdk-name>]
```

`sdk-name`: SDK 名称， 不传展示所有。

## Current

查看当前 SDK 的版本。

**用法**

```shell
vfox current [<sdk-name>]
vfox c
```

## Cd 

在 `VFOX_HOME` 或 SDK 目录下启动 shell。

**用法**

```shell
vfox cd [options] [<sdk-name>]
```

`sdk-name`: SDK 名称, 不传默认为 `VFOX_HOME`。

**选项**

- `-p, --plugin`: 在插件目录下启动 shell。


## Upgrade <Badge type="tip" text=">= 0.4.2" vertical="middle" />

升级 `vfox` 到最新版本。

**用法**

```shell
vfox upgrade
```
