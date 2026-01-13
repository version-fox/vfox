# 配置

`vfox` 允许你修改一些配置, 所有配置信息都存放在`$HOME/.version-fox/config.yaml`文件中。

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

`vfox`默认将 SDK 缓存文件存储在`$HOME/.version-fox/cache`目录下。

::: danger !!!
在配置之前， 请确保`vfox`有文件夹的写权限。⚠⚠⚠
:::

```yaml
storage:
  sdkPath: /tmp
```

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
`$HOME/.version-fox/plugins/<plugin-name>/available.cache`
:::

## Config 命令 

设置，查看配置

**用法**

```shell
vfox config [<key>] [<value>]

vfox config proxy.enable true
vfox config proxy.url http://localhost:7890
vfox config storage.sdkPath /tmp
```

`key`：配置项，以`.`分割。
`value`：不传为查看配置项的值。

**选项**

- `-l, --list`：列出所有配置。
- `-un, --unset`：删除一个配置。
