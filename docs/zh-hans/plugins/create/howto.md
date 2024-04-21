
# 创建插件

## 插件里有什么

目录结构如下:
```shell
    .
    ├── README.md
    ├── LICENSE
    └── hooks
        └── available.lua
        └── env_keys.lua
        └── post_install.lua
        └── pre_install.lua
        ....
    └── lib
        └── xxx.lua
        └── xxx.lua
        └── xxx.lua
    └── metadata.lua

```

- `hooks` 目录用于存放插件的钩子函数。**一个钩子函数对应一个`.lua`文件。**
- `lib` 目录用于存放插件的依赖库。`vfox`会自动加载这个目录下的所有`.lua`文件。**放在其他目录下,则无法加载。**
- `metadata.lua` 插件的元数据信息。用于描述插件的基本信息，如插件名称、版本等。
- `README.md` 插件的说明文档。
- `LICENSE` 插件的许可证。

::: warning 插件模板
为了方便插件的开发，我们提供了一个插件模板，你可以直接使用[vfox-plugin-template](https://github.com/version-fox/vfox-plugin-template)创建一个插件。
:::

## 钩子概览

`vfox`支持的所有钩子函数列表。

| 钩子                                              | **必须** | 描述                   |
|:------------------------------------------------|:-------|:---------------------|
| [hooks/available.lua](#available)               | ✅      | 列举所有可用版本             |
| [hooks/pre_install.lua](#preinstall)            | ✅      | 解析版本号并返回预安装信息,如下载地址等 |
| [hooks/env_keys.lua](#envkeys)                  | ✅      | 配置环境变量               |
| [hooks/post_install.lua](#postinstall)          | ❌      | 执行额外的操作, 如编译源码等      |
| [hooks/pre_use.lua](#preuse)                    | ❌      | 在切换版本之前, 提供修改版本的机会   |
| [hooks/parse_legacy_file.lua](#parselegacyfile) | ❌      | 自定义解析遗留文件            |

## 必须实现的钩子函数

### PreInstall

返回预安装信息， 例如具体版本号、下载源等信息。 `vfox`会帮你提前将这些文件下载到特定目录下。如果是压缩包如`tar`、`tar.gz`、`tar.xz`、`zip`这四种压缩包， `vfox`会直接帮你解压处理。

**位置**: `hooks/pre_install.lua`
```lua
function PLUGIN:PreInstall(ctx)
    --- 用户输入
    local version = ctx.version
    return {
        --- 版本号
        version = "xxx",
        --- 文件地址, 可以是远程地址或者本地文件路径[可选]
        url = "xxx",
        --- 备注信息[可选]
        note = "xxx",
        --- SHA256 checksum [optional]
        sha256 = "xxx",
        --- md5 checksum [optional]
        md5= "xxx",
        --- sha1 checksum [optional]
        sha1 = "xxx",
        --- sha512 checksum [optional]
        sha512 = "xx",
        --- 额外需要的文件 [optional]
        addition = {
            {
                --- additional file name !
                name = "xxx",
                --- 其余同上
            }
        }
    }
end
```

### Available

返回当前可用版本列表。如没有则返回空数组。

**位置**: `hooks/available.lua`
```lua
function PLUGIN:Available(ctx)
    --- 用户输入附带的参数, 数组
    local args = ctx.args
    --- 插件的名称（别名）
    local name = ctx.name
    return {
        {
            version = "xxxx",
            note = "LTS",
            addition = {
                {
                    name = "npm",
                    version = "8.8.8",
                }
            }
        }
    }
end
```

### EnvKeys

告诉`vfox`当前SDK需要配置的环境变量有哪些。

**位置**: `hooks/env_keys.lua`
```lua
function PLUGIN:EnvKeys(ctx)
    --- this variable is same as ctx.sdkInfo['plugin-name'].path
    local mainPath = ctx.path
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
    return {
        {
            key = "JAVA_HOME",
            value = mainPath
        },
        --- 注意, 如果需要设置多个PATH路径, 只需要传递多个PATH即可,vfox将会自动去重并按配置顺序设置
        {
            key = "PATH",
            value = mainPath .. "/bin"
        },
        {
            key = "PATH",
            value = mainPath .. "/bin2"
        }
    }
end
```



## 可选钩子函数

### PostInstall

拓展点，在`PreInstall`执行之后调用，用于执行额外的操作， 如编译源码等。根据需要实现。

**位置**: `hooks/post_install.lua`
```lua
function PLUGIN:PostInstall(ctx)
    --- ctx.rootPath SDK 安装目录
    local rootPath = ctx.rootPath
    --- 根据PreInstall返回的name获取
    local sdkInfo = ctx.sdkInfo['sdk-name']
    --- 文件存放路径
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end
```

### PreUse

当用户使用 `vfox use` 的时候，会调用插件的 `PreUse` 函数。这个函数的作用是返回用户输入的版本信息。
如果 `PreUse` 函数返回了版本信息， `vfox` 将会使用这个新的版本信息。

**位置**: `hooks/pre_use.lua`
```lua
function PLUGIN:PreUse(ctx)
    --- 用户输入的版本
    local version = ctx.version
    --- 用户之前环境中设置的版本
    local previousVersion = ctx.previousVersion

    --- 已安装的 SDK 信息
    local sdkInfo = ctx.installedSdks['version']
    local path = sdkInfo.path
    local name = sdkInfo.name
    local version = sdkInfo.version

    --- 当前工作目录
    local cwd = ctx.cwd

    --- 用户输入的 scope 信息，值为 global/project/session 其中之一
    local scope = ctx.scope

    --- 返回版本信息
    return {
        version = version,
    }
end
```
### ParseLegacyFile <Badge type="tip" text=">= 0.4.0" vertical="middle" />

解析其他配置文件，以确定工具的版本。例如，`nvm` 的 `.nvmrc` 文件、`SDKMAN` 的 `.sdkmanrc` 文件等。

::: danger
该钩子函数必须配合 `legacyFilenames` 配置项使用, 告诉`vfox` 你的插件支持解析哪些文件。
:::

**位置**: `metadata.lua`
```lua 
--- 当前插件支持解析的遗留文件名列表, 例如: .nvmrc, .node-version, .sdkmanrc
PLUGIN.legacyFilenames = {
    '.nvmrc',
    '.node-version',
}
```

**位置**: `hooks/parse_legacy_file.lua`
```lua 
function PLUGIN:ParseLegacyFile(ctx)
    --- 文件名
    local filename = ctx.filename
    --- 文件路径
    local filepath = ctx.filepath
    --- 获取当前插件已安装的版本列表
    local versions = ctx:getInstalledVersions()

    return {
        --- 返回具体版本号
        version = "x.y.z"
    }
end
```


## 测试插件

目前，`vfox` 插件测试方法很简陋。您需要将插件放在 `${HOME}/.version-fox/plugins` 目录中，并使用不同的命令验证您的功能是否正常工作。您可以在c插件中使用 `print`/`printTable` 函数来打印日志进行调试。

- PLUGIN:PreInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:PostInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:Available -> `vfox search <sdk-name>`
- PLUGIN:EnvKeys -> `vfox use <sdk-name>@<version>`



## 向官方插件存储库提交插件

`vfox`可以允许自定义安装插件，比如:

```shell
vfox add --source https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip 
```

为了使你的用户更轻松，你可以将插件添加到官方插件存储库中，以列出你的插件并使用较短的命令轻松安装，比如 `vfox add nodejs`。

具体步骤请查看[如何将插件提交到索引仓库](./howto_registry.md)。
