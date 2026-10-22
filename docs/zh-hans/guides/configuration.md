# 配置

`vfox` 允许你修改一些配置, 所有配置信息都存放在`$HOME/.vfox/config.yaml`文件中。

::: tip 注意
如果你是首次运行`vfox`, 则会自动创建一个空的 config.yaml 文件。
:::

## 兼容版本文件 

插件 **支持** 读取其他版本管理器的配置文件, 例如: Nodejs 的`nvm`的`.nvmrc`文件, Java 的`SDKMAN`的`.sdkmanrc`文件等。

此能力**默认是开启的**。相关配置选项如下:

```yaml
legacyVersionFile:
  enable: true
  strategy: "specified" # 解析策略
```

- `enable`: 是否启用 legacy version file 解析功能
- `strategy`: 解析策略，详见下方策略选项说明

### 策略选项

`vfox` 支持以下三种解析策略：

- `latest_installed`: 使用最新安装的版本
- `latest_available`: 使用最新可用的版本
- `specified`: 使用 legacy file 中指定的版本（默认）

::: warning

1. 如果目录里同时存在`.vfox.toml`和其他版本管理器的配置文件(`.nvmrc`, `.sdkmanrc`等),
   `vfox` **优先加载**`.vfox.toml`文件.
2. 开启此功能可能会导致`vfox`刷新环境变量时略微变慢, **请根据自己的需求开启**。

:::

如果你想禁用此功能，可以使用命令：`vfox config legacyVersionFile.enable false`

## 代理设置

::: tip 注意
当前仅支持 http(s)代理协议
:::

**格式**: `http[s]://[username:password@]host:port`

```yaml
proxy:
  enable: false
  url: http://localhost:7890
```

## 存储路径

`vfox`默认将 SDK 缓存文件存储在`$HOME/.vfox/cache`目录下。

::: danger !!!
在配置之前， 请确保`vfox`有文件夹的写权限。⚠⚠⚠
:::

```yaml
storage:
  sdkPath: /tmp
```

`storage.sdkPath` 表示 SDK 的根目录。例如当 `sdkPath` 为 `/tmp` 时，Node.js 会安装到 `/tmp/nodejs/`。

::: warning 兼容性说明
`v1.0.2` 到 `v1.0.7` 错误地将 `storage.sdkPath` 当成 shared root 使用，导致 SDK 被安装到 `sdkPath/cache/<sdk>`。

现在该行为已经修正，重新与 `v0.10.x` 保持一致。对于已经落在错误 `cache` 路径下的 SDK，`vfox` 仍会临时兼容；新安装会使用正确布局 `sdkPath/<sdk>`。

如果 `sdkPath/<sdk>` 和 `sdkPath/cache/<sdk>` 同时存在，`vfox` 会优先使用 `sdkPath/<sdk>`。
:::

## 插件注册表地址

`vfox`默认从[插件仓库](https://version-fox.github.io/vfox-plugins)检索插件。

如果你想要使用自己的索引仓库或第三方镜像仓库,可以按照以下方式配置:

```yaml
registry:
  address: "https://version-fox.github.io/vfox-plugins"
```

::: tip 可用镜像

- https://gitee.com/version-fox/vfox-plugins/raw/main/plugins
- https://cdn.jsdelivr.net/gh/version-fox/vfox-plugins/plugins
- https://rawcdn.githack.com/version-fox/vfox-plugins/plugins
  :::

## 缓存

`vfox` 默认会缓存`search`命令的结果, 以减少网络请求次数。默认缓存时间为`12h`。

::: warning 特殊值

- `-1`: 永不过期
- `0`: 不进行缓存
  :::

```yaml
cache:
  availableHookDuration: 12h # s 秒, m 分钟, h 小时
```

::: tip 缓存文件路径
`$HOME/.vfox/plugin/<plugin-name>/.available.cache`
:::

## Gitignore 设置 <Badge type="tip" text=">= 1.0.12" vertical="middle" />

当你执行 `vfox use <sdk>@<version> -p` (项目作用域) 时, `vfox` 会在项目根目录创建 `.vfox/` 目录, 并且默认会向项目已有的 `.gitignore` 中追加一行 `.vfox/` (如果项目没有 `.gitignore`, 则不会自动创建)。

如果你是通过 [全局 gitignore](https://git-scm.com/docs/gitignore) (`core.excludesfile`) 来管理 `.vfox/`, 不希望 `vfox` 修改项目的 `.gitignore`, 可以关闭此行为:

```yaml
gitignore:
  enable: true # 设置为 false 可关闭自动更新 .gitignore
```

通过命令关闭:

```shell
vfox config gitignore.enable false
```

## 诊断设置

执行 `vfox use` 之后，`vfox` 会检查刚选中的版本，并把值得注意的情况打印出来，例如你拿到的 `node` 并不是刚选中的那个，因为 `PATH` 里更靠前的位置还有另一个 `node`。这些就是 [`vfox doctor`](../usage/core-commands.md#doctor) 的快速检查，它们不会让 `vfox use` 失败。

```yaml
diagnostics:
  onUse: true # 设置为 false 可关闭 `vfox use` 之后的自动检查
  ignore: # 静音某些结果：<finding-id>[:<sdk>[@<version>]]
    - "sdk.drift:flutter@3.47.0"
    - "path.shadowed:nodejs"
```

通过命令关闭检查：

```shell
vfox config diagnostics.onUse false
```

## PATH 顺序设置

`vfox` 会把自己的目录放到 `PATH` 的最前面，但会保留在它之前就已经排在前面的内容，所以你在 `vfox` 之后才激活的 Python 虚拟环境仍然优先。不过排在 `vfox` 前面的并不都是虚拟环境：macOS 的 `path_helper` 会把系统目录挪到登录 shell 的最前面；Windows 的系统 `PATH` 排在 `vfox use -g` 写入的用户 `PATH` 之前；`pnpm setup` 会往 shell 启动文件末尾追加一行，把 `$PNPM_HOME` 放到最前面。有了这些在前面，不管 `vfox use` 怎么说，`node` 或 `java` 依然是系统（或 pnpm）的那一个。出现这种情况时 `vfox doctor` 会告诉你。

```yaml
path:
  mode: legacy # legacy | baseline | aggressive
  preserve: [] # 始终排在 vfox 目录之前的目录，例如 ["~/.cargo/bin"]
```

- `legacy`（目前的默认值）：第一个 `vfox` 目录之前的所有内容都继续排在它前面。
- `baseline`（推荐）：只有在 `vfox activate` 运行**之后**才加入 `PATH` 的内容继续排在前面。`vfox` 激活时 `PATH` 里已经有的内容（比如上面说的系统目录）会排到 `vfox` 目录之后。`vfox activate` 会把当时的 `PATH` 记录在 `__VFOX_ORIG_PATH` 中。如果你依赖某个目录排在 `vfox` 之前，可以把它列入 `preserve`。
- `aggressive`：只有处于激活状态的虚拟环境 / conda 环境的目录，以及 `preserve` 中的目录继续排在前面。

处于激活状态的虚拟环境或 conda 环境的目录（`VIRTUAL_ENV`、`CONDA_PREFIX`）始终排在前面。

```shell
vfox config path.mode baseline
```

::: warning ⚠️ 默认值将会改变
未来的版本会把 `path.mode` 的默认值从 `legacy` 改为 `baseline`，这样上面说的系统目录不用任何设置就不再压过 `vfox`。[快速入门](./quick-start.md#_2-挂载vfox到你的shell)已经要求新安装的用户设置它。想在改动之后继续保持现在的行为，请显式设置：`vfox config path.mode legacy`。
:::

::: tip
修改设置后，shell 在下一次提示符出现时生效。`baseline` 还需要 `vfox activate` 在该 shell 启动时记录下来的 `PATH`；在升级 `vfox` 之前启动的 shell 里没有这份记录，会按 `legacy` 处理，重新打开终端即可。
:::

## 别名

用 [`vfox alias`](../usage/core-commands.md#alias) 给已安装版本起的名字也保存在同一个文件里。它们只在命令行输入时解析，不会出现在 `.vfox.toml` 中。`vfox alias` 只修改 `aliases` 这一项，文件的其余部分保持原样，所以你没有设置的配置段会继续沿用共享配置。

```yaml
aliases:
  java:
    j17: 17.0.9-tem
    j21: 21.0.1-graal
```

## Config 命令 

设置，查看配置

**用法**

```shell
vfox config [<key>] [<value>]

vfox config proxy.enable true
vfox config proxy.url http://localhost:7890
vfox config storage.sdkPath /tmp
vfox config path.mode baseline
```

`key`：配置项，以`.`分割。
`value`：不传为查看配置项的值。

**选项**

- `-l, --list`：列出所有配置。
- `-un, --unset`：删除一个配置。
