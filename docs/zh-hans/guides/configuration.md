# 配置

`vfox` 允许你修改一些配置, 所有配置信息都存放在`$HOME/.version-fox/config.yaml`文件中。

::: tip 注意
如果你是首次运行`vfox`, 则会自动创建一个空的config.yaml文件。
:::

## 代理设置

::: tip 注意
当前仅支持http(s)代理协议
:::

**格式**: `http[s]://[username:password@]host:port`

```yaml
proxy:
  enable: false
  url: http://localhost:7890
```

## 存储路径

`vfox`默认将SDK缓存文件存储在`$HOME/.version-fox/cache`目录下。

::: danger !!!
在配置之前， 请确保`vfox`有文件夹的写权限。⚠⚠⚠
:::

```yaml
storage:
  sdkPath: /tmp
```

## 插件注册表地址

`vfox`默认从[vfox-plugins.lhan.me](https://vfox-plugins.lhan.me)检索插件。

如果你想要使用自己的索引仓库或第三方镜像仓库,可以按照以下方式配置:

```yaml
registry:
  address: 'https://vfox-plugins.lhan.me'
```

::: tip 可用镜像
- https://gitee.com/version-fox/vfox-plugins/raw/main/plugins
- https://cdn.jsdelivr.net/gh/version-fox/vfox-plugins/plugins
- https://rawcdn.githack.com/version-fox/vfox-plugins/plugins
:::
