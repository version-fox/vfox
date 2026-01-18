# 核心命令

## Search

获取指定 SDK 所有可用的运行时版本。

**用法**

```shell
vfox search <sdk-name> [...optionArgs]
```

`sdk-name`: 运行时名称， 如`nodejs`、`custom-node`。
`optionArgs`: 搜索命令的附加参数。注意：是否支持取决于插件。

**选项**

- `--skip-cache`: 跳过本次搜索的可用缓存读写。

::: warning 缓存

`vfox`会缓存`search`的结果, 默认缓存时间为`12h`。

如果你想禁用，可以通过`vfox config`命令进行配置。
```shell
vfox config cache.availableHookDuration 0
```

其余操作, 请查看[配置#缓存](../guides/configuration.md#%E7%BC%93%E5%AD%98)。

:::

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

- `-a, --all`: 安装 .vfox.toml 中记录的所有 SDK 版本
- `-y, --yes`: 直接安装，跳过确认提示

::: tip 自动安装
你可以一次性安装多个 SDK，通过空格分隔。

```shell
vfox install nodejs@20 golang ...
```

:::

::: tip
直接安装，跳过确认提示

```shell
vfox install --yes nodejs@20
vfox install --yes --all
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

## Unuse

从指定作用域取消设置运行时版本

**用法**

```shell
vfox unuse [options] <sdk-name>
```

`sdk-name`: SDK 名称

**选项**

- `-g, --global`: 从全局作用域移除
- `-p, --project`: 从项目作用域移除（当前目录）
- `-s, --session`: 从会话作用域移除（当前 Shell 会话）

::: tip 默认作用域
`Windows`: 默认`Global`作用域

`Unix-like`: 默认`Session`作用域
:::

::: warning 效果
使用 `unuse` 后，SDK 将不再在指定作用域中处于活动状态。如果 SDK 在其他作用域中配置，那些将根据 vfox 的作用域层次结构优先生效（Session > Project > Global）。
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


## Upgrade

升级 `vfox` 到最新版本。

**用法**

```shell
vfox upgrade
```

## Exec <Badge type="tip" text=">= 1.0.0" vertical="middle" />

在 vfox 管理的环境中执行命令。

**用法**

```shell
vfox exec <sdk-name>[@<version>] -- <command> [args...]

vfox x <sdk-name>[@<version>] -- <command> [args...]
```

`sdk-name`: SDK 名称

`version`[可选]: 指定使用的版本。如不传，则使用当前作用域配置的版本。

`command`: 要执行的命令

`args`: 传递给命令的参数

**说明**

`exec` 命令允许您在指定的 SDK 环境中临时执行命令，而无需修改当前作用域的配置。这对于以下场景特别有用：

- **IDE 集成**: 在 IDE（如 VS Code）中使用项目特定版本的 SDK
- **脚本执行**: 在 CI/CD 或构建脚本中使用特定 SDK 版本
- **临时测试**: 临时使用不同版本的 SDK 测试代码

**示例**

```shell

# 使用指定版本执行命令
vfox exec nodejs@20.9.0 -- node -v

# 在 maven 环境中执行构建
vfox exec maven@3.9.1 -- mvn clean install

# 使用别名 x（exec 的简写）
vfox x maven@3.9.1 -- mvn clean

```

::: tip IDE 集成

在 VS Code 中，您可以使用 `exec` 命令来确保项目使用正确版本的 SDK。例如，在 `.vscode/tasks.json` 中配置：

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

::: tip 版本自动安装

如果指定的版本尚未安装，`exec` 命令会自动安装它。

:::

::: warning 环境变量

`exec` 命令会在子进程中设置正确的环境变量（如 PATH、JAVA_HOME 等），但不会影响当前 Shell 会话。

:::
