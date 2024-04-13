# 核心命令

核心`vfox`命令很少， 下面列举的是最频繁使用的命令。

## Available

列举[索引仓库](https://github.com/version-fox/vfox-plugins)中所有可用的插件。

**用法**
```shell
vfox available
```

## Add

在 `vfox` 中，插件就是 SDK，而 SDK 就是插件。因此，在使用它们之前，您需要安装相应的插件。


**用法**

```shell
vfox add [options] <plugin-name>
```
`plugin-name`: 插件名称， 如`nodejs`。

**选项**
- `-a, --alias`: 设置插件别名。
- `-s, --source`: 安装指定路径下的插件（可以是远程文件也可以是本地文件）


**例子**

**安装仓库插件**
```shell
$ vfox add --alias node nodejs
```
或者安装多个仓库插件
```bash
$ vfox add nodejs golang
```

**安装自定义插件**
```shell
$ vfox add --source  https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip custom-node
```


## Search

获取指定SDK所有可用的运行时版本。

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
如果本地没有安装SDK，`search`命令会从远端仓库检索插件并安装到本地。
:::

## Install

将指定的 SDK 版本安装到您​​的计算机并缓存以供将来使用。

**用法**

```shell
vfox install <sdk-name>@<version>

vfox i <sdk-name>@<version>
```


`sdk-name`: SDK名称

`version`: 需要安装的版本号


::: tip 自动安装
如果本地没有安装SDK，`install`命令会从远端仓库检索插件并安装到本地。
:::

## Use

设置运行时版本


**用法**

```shell
vfox use [options] <sdk-name>[@<version>]

vfox u [options] <sdk-name>[@<version>]
```

`sdk-name`: SDK名称

`version`[可选]: 使用 指定版本运行时。如不传， 则下拉选择。 

**选项**
- `-g, --global`: 全局生效
- `-p, --project`: 当前目录下生效
- `-s, --session`: 当前Shell会话内生效


::: tip 默认作用域
`Windows`: 默认`Global`作用域

`Unix-like`: 默认`Session`作用域

:::


## Uninstall

卸载指定版本的SDK。


**用法**

```shell
vfox uninstall <sdk-name>@<version>
vfox un <sdk-name>@<version>
```

`sdk-name`: SDK名

`version`: 具体版本号
