# Configuration

`vfox` allows you to change some configurations, all configuration is stored in the `$HOME/.version-fox/config.yaml`
file.

::: tip
If you use `vfox` for the first time, an empty `config.yaml` file will be created automatically.
:::

## Legacy Version File <Badge type="tip" text=">= 0.4.0" vertical="middle" />

Plugins **with support** can read the versions files used by other version managers,
for example, `.nvmrc` in the case of Nodejs's `nvm`.

This capability is **turned off by default**, you can enable it as follows.

```yaml
legacyVersionFile:
  enable: true
```

::: warning

1. If both `.tool-versions` and other version manager's configuration files (`.nvmrc`, `.sdkmanrc`, etc.) exist in the
   directory, `vfox` **priority read** the `.tool-versions` file.
2. Enabling this feature may cause `vfox` to refresh environment variables slightly slower, **please enable it according
   to your needs**.
   :::

## Proxy Settings

::: tip
Currently only support http(s) proxy protocol
:::

**Format**: `http[s]://[username:password@]host:port`

```yaml
proxy:
  enable: false
  url: http://localhost:7890
```

## Storage Settings

By default, `vfox` stores SDK cache files in the `$HOME/.version-fox/cache` directory.

::: danger !!!
Before configuring, please make sure that `vfox` has write permission to the folder.⚠⚠⚠
:::

```yaml
storage:
  sdkPath: /tmp
```

## Plugin Registry Address

`vfox` will default to retrieve plugins from [vfox-plugins.lhan.me](https://vfox-plugins.lhan.me).

If you want to use **your own registry or third-party mirror registry**, please configure it following:

```yaml
registry:
  address: "https://vfox-plugins.lhan.me"
```

::: tip Available Mirrors

- https://cdn.jsdelivr.net/gh/version-fox/vfox-plugins/plugins
- https://rawcdn.githack.com/version-fox/vfox-plugins/plugins
  :::

## Cache Settings <Badge type="tip" text=">= 0.4.1" vertical="middle" />

`vfox` will cache the results of the `search` command (`available` hook) by default to reduce the number of network requests. The default
cache time is `12h`.

::: warning Special Value
- `-1`: Never expire
- `0`: Do not cache
:::

```yaml
cache:
  availableHookDuration: 12h # s second, m minute, h hour
```


::: tip Cache File Path
`$HOME/.version-fox/plugins/<plugin-name>/available.cache`
:::

## Config Command <Badge type="tip" text=">= 0.4.0" vertical="middle" />

Setup, view config

**Usage**

```shell
vfox config [<key>] [<value>]

vfox config proxy.enable true
vfox config proxy.url http://localhost:7890
vfox config storage.sdkPath /tmp
```

`key`: Configuration item, separated by `. `.
`value`: If not passed, look at the value of the configuration item.

**Options**

- `-l, --list`: list all config.
- `-un, --unset`: remove a config.
