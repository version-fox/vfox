# 核心命令

## Search

获取指定 SDK 所有可用的运行时版本。

**用法**

```shell
vfox search <sdk-name> [...optionArgs]
```

`sdk-name`: 运行时名称， 如`nodejs`、`custom-node`。
`optionArgs`: 搜索命令的附加参数。注意：是否支持取决于插件。

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
- `-u, --use`: 安装完成后在当前 shell 会话中使用该版本，等同于 `vfox use`，因此同样需要已激活的 shell（`vfox activate`）。不能与 `--all` 同时使用，必须指定版本（不会进入选择器），每个 SDK 只能指定一个版本。
- `--reinstall`: 用全新的副本替换已安装的版本，例如 SDK 在原地改动过自己之后（见 [Doctor](#doctor)）。版本号需与已安装的完全一致。通过 [`vfox link`](#link) 登记的版本不能重装。

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
使用 `unuse` 后，SDK 将不再在指定作用域中处于活动状态。如果 SDK 在其他作用域中配置，那些将根据 vfox 的作用域层次结构优先生效（Project > Session > Global）。
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

::: tip 通过 link 登记的版本
卸载用 [`vfox link`](#link) 登记的版本，只会删除链接，它指向的 SDK 不会被动。
:::

## List

查看当前已安装的所有 SDK 版本。

**用法**

```shell
vfox list [<sdk-name>]

vfox ls[<sdk-name>]
```

`sdk-name`: SDK 名称， 不传展示所有。

用 [`vfox link`](#link) 登记的版本会标注 `(linked → <路径>)`，有 [别名](#alias) 的版本会显示别名：

```shell
$ vfox list java
-> 17-system (linked → /usr/lib/jvm/java-17-openjdk-amd64)
-> 17.0.9-tem [alias: j17] <— current
```

## Link

把本机上已经存在的 SDK 当作 `vfox` 的一个版本来用，比如系统包管理器装的 JDK、IDE 下载的 JDK，或者 CI runner 工具缓存里自带的 JDK。

不会下载，也不会复制。`vfox` 只保存一个指向该目录的链接，SDK 仍然留在原处、仍然归它原来的主人所有：`vfox uninstall` 只会删除链接。

**用法**

```shell
vfox link [options] <sdk-name>@<version> <path>

vfox ln <sdk-name>@<version> <path>

vfox link --detect [-y] <sdk-name>
```

`sdk-name`: SDK 名称。需要已经添加了它的插件。

`version`: 给这个 SDK 起的名字，原样使用，例如 `17.0.9`、`17-system`、`brew`。可以使用字母、数字、`.`、`_`、`+` 和 `-`，不能是 `latest`。

`path`: SDK 所在的目录，即包含 `bin` 的那一层。

**选项**

- `-f, --force`: 把已经链接的版本指向另一个目录。`vfox` 自己安装的版本不会被替换。
- `--detect`: 让插件在本机上查找 SDK，并把找到的链接进来。插件需要实现 [`Detect`](../plugins/create/howto.md#detect) hook。
- `-y, --yes`: 配合 `--detect` 使用，不再询问，直接链接找到的全部。

**示例**

```shell
vfox link java@17-system /usr/lib/jvm/java-17-openjdk-amd64
vfox use java@17-system

# 已经链接的版本不会再下载
vfox install java@17-system

# 它在哪里？
vfox info --format "{{.Home}}" java@17-system

# 只删除链接，不删除 JDK
vfox uninstall java@17-system
```

::: tip 插件有发言权
`vfox link` 会让插件判断这个目录像不像它的 SDK：插件放进 `PATH` 的目录必须存在且里面有命令。否则会拒绝链接并说明原因。
:::

## Alias

给已安装的版本起一个自己的短名字。

**用法**

```shell
vfox alias set <sdk-name> <alias> <version>
vfox alias list [<sdk-name>]
vfox alias unset <sdk-name> <alias>
```

**示例**

```shell
vfox alias set java j17 17.0.9-tem
vfox use java@j17
```

::: tip 别名只留在你的机器上
别名保存在你的 `config.yaml` 中，只在命令行输入时解析：`use`、`uninstall`、`info`、`exec` 和 `doctor`。`vfox use -p java@j17` 写进 `.vfox.toml` 的是真实版本，所以没有这个别名的同事也能正常使用这个文件。
:::

已安装的同名版本永远优先于别名，因此 `alias set` 会拒绝这样的名字。卸载一个版本时，指向它的别名会一并删除。

你输入的版本号开头的 `v`（`v20.1.0` 即 `20.1.0`）只有在没有任何已安装版本、也没有任何别名叫这个名字时才会被去掉。所以别名和用 [`vfox link`](#link) 登记的版本都可以以 `v` 开头。

## Doctor

检查 `vfox` 和已安装的 SDK 有没有问题。

**用法**

```shell
vfox doctor [options] [<sdk-name>[@<version>]]
```

**选项**

- `--json`: 以 JSON 输出检查结果。
- `--adopt`: 为 `vfox` 开始保存回执之前安装的版本补写回执，以它们当前的状态作为之后比较的基准。

`vfox` 会在每个它安装或链接的版本目录里写一个很小的回执（`.vfox-receipt.json`），记录这个版本从哪里来；如果插件实现了 [`Fingerprint`](../plugins/create/howto.md#fingerprint)，还会记录能标识这个版本的事实。

| 检查项                    | 级别  | 含义                                                                                                              |
| :------------------------ | :---- | :---------------------------------------------------------------------------------------------------------------- |
| `path.shadowed`           | warn  | SDK 的某个命令没有被运行，因为 `PATH` 中更靠前的位置有同名的可执行文件（pnpm 装的 `node`、系统 JDK……）                |
| `path.not-activated`      | error | 当前 shell 没有设置 `vfox activate`                                                                               |
| `sdk.drift`               | warn  | SDK 在安装之后被就地改动过（例如 `flutter upgrade`）。处理：`vfox install --reinstall <sdk>@<version>`               |
| `sdk.link-dangling`       | error | 链接版本指向的目录不见了                                                                                          |
| `sdk.payload-empty`       | error | 插件放进 `PATH` 的目录不存在，或里面没有命令                                                                      |
| `sdk.envkeys-failed`      | error | 插件说不出该往环境里放什么                                                                                        |
| `sdk.install-interrupted` | warn  | 有一次安装被中断了                                                                                                |
| `sdk.no-receipt`          | info  | 安装于回执出现之前，无法发现它被改动过（`vfox doctor --adopt`）                                                   |
| `sdk.plugin-missing`      | warn  | 有已安装的版本，但没有添加对应的插件                                                                              |
| `plugin.incompatible`     | warn  | 插件需要更新版本的 `vfox`                                                                                         |
| `alias.dangling`          | warn  | 别名指向一个没有安装的版本                                                                                        |

`path.shadowed` 检查的是你的 shell 正在使用的版本，也就是 `vfox env` 放进 `PATH` 的那个：项目、会话和全局设置所要求的版本里，第一个已安装的。指定版本（例如 `vfox doctor java@17`）则改为检查那个版本。如果那个命令其实就是 SDK 自己提供的同一个文件（例如指向 [`vfox link`](#link) 登记的安装的符号链接），就不算遮蔽。

发现错误时退出码为 `1`。

执行 `vfox use` 之后，`vfox` 会自动对刚选中的版本运行快速检查并打印警告。如何关闭它，或只静音个别结果，见 [诊断设置](../guides/configuration.md#诊断设置)。

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

在 Linux 和 macOS 上，升级会直接替换 `vfox` 可执行文件，因此需要对其所在目录有写权限。如果 `vfox` 安装在 `/usr/local/bin` 等系统目录，请使用 `sudo` 运行，例如 `sudo /usr/local/bin/vfox upgrade`。

**用法**

```shell
vfox upgrade
```

## Exec <Badge type="tip" text=">= 1.0.0" vertical="middle" />

在 vfox 管理的环境中执行命令。

**用法**

```shell
vfox exec <sdk-name>[@<version>]... -- <command> [args...]

vfox x <sdk-name>[@<version>]... -- <command> [args...]
```

`sdk-name`: SDK 名称

`version`[可选]: 指定使用的版本。如不传，则使用当前作用域配置的版本。

可以在 `--` 之前连续指定多个 SDK。它们会按从左到右的顺序合并；如果 PATH 或环境变量发生重叠，左边的 SDK 优先级更高。

`command`: 要执行的命令

`args`: 传递给命令的参数

**说明**

`exec` 命令允许您在指定的 SDK 环境中临时执行命令，而无需修改当前作用域的配置。这对于以下场景特别有用：

- **IDE 集成**: 在 IDE（如 VS Code）中使用项目特定版本的 SDK
- **脚本执行**: 在 CI/CD 或构建脚本中使用特定 SDK 版本
- **临时测试**: 临时使用不同版本的 SDK 测试代码

::: tip 非交互环境

在 Docker、CI/CD 和其他非交互 Shell 中，推荐优先使用 `vfox exec`，而不是依赖 `vfox activate`。

`activate` 通过 Shell Hook 集成 vfox，而 `exec` 会为当前子进程显式准备好 SDK 环境。

```shell
vfox exec nodejs@24.14.0 -- npm install -g pnpm
vfox exec nodejs@24.14.0 -- bash -lc 'node -v && npm -v'
```

:::

**示例**

```shell

# 使用指定版本执行命令
vfox exec nodejs@20.9.0 -- node -v

# 同时指定多个 SDK 执行命令
vfox exec nodejs@24.14.0 golang@1.25.6 -- npm install -g @qwen-code/qwen-code@latest

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
