
# 创建插件


在`vfox`中，插件即SDK，SDK即插件。 

`vfox` 插件以 `Lua` 脚本的形式提供。这种方法的好处是：

- 插件开发成本低；只需要对 Lua 语法有基本的了解。
- 与平台解耦；插件可以在任何平台上运行，只需将插件文件放在指定目录中即可。
- 插件可以跨平台共享；编写一次，随处运行。
- 可定制、可共享，并且可以使用其他人共享的插件。


## 插件里有什么

插件就是一个`lua`脚本。 里面提供了四个Hook函数， 分别是`PLUGIN:PreInstall`、`PLUGIN:PostInstall`、`PLUGIN:EnvKeys`、`PLUGIN:Available`。你需要做的就是实现这四个函数即可。


### PreInstall

返回预安装信息， 例如具体版本号、下载源等信息。 `vfox`会帮你提前将这些文件下载到特定目录下。如果是压缩包如`tar`、`tar.gz`、`tar.xz`、`zip`这四种压缩包， `vfox`会直接帮你解压处理。

```lua
function PLUGIN:PreInstall(ctx)
    --- 用户输入
    local version = ctx.version
    --- 当前vfox运行时版本
    local runtimeVersion = ctx.runtimeVersion
    return {
        --- 版本号
        version = "xxx",
        --- remote URL or local file path [optional]
        url = "xxx",
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
                --- remote URL or local file path [optional]
                url = "xxx",
                --- SHA256 checksum [optional]
                sha256 = "xxx",
                --- md5 checksum [optional]
                md5= "xxx",
                --- sha1 checksum [optional]
                sha1 = "xxx",
                --- sha512 checksum [optional]
                sha512 = "xx",
            }
        }
    }
end
```

### PostInstall

拓展点，在`PreInstall`执行之后调用，用于执行额外的操作， 如编译源码等。根据需要实现。

```lua
function PLUGIN:PostInstall(ctx)
    --- ctx.rootPath SDK 安装目录
    local rootPath = ctx.rootPath
    local runtimeVersion = ctx.runtimeVersion
    --- 根据PreInstall返回的name获取
    local sdkInfo = ctx.sdkInfo['sdk-name']
    --- 文件存放路径
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end
```

### Available

返回当前可用版本列表。如没有则返回空数组。

```lua
function PLUGIN:Available(ctx)
    local runtimeVersion = ctx.runtimeVersion
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

```lua
function PLUGIN:EnvKeys(ctx)
    --- this variable is same as ctx.sdkInfo['plugin-name'].path
    local mainPath = ctx.path
    local runtimeVersion = ctx.runtimeVersion
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

## PreUse

当用户使用 `vfox use` 的时候，会调用插件的 `PreUse` 函数。这个函数的作用是返回用户输入的版本信息。
如果 `PreUse` 函数返回了版本信息， `vfox` 将会使用这个新的版本信息。

```lua
function PLUGIN:PreUse(ctx)
    local runtimeVersion = ctx.runtimeVersion
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

## 测试插件

目前，`vfox` 插件测试方法很简陋。您需要将插件放在 `${HOME}/.version-fox/plugins` 目录中，并使用不同的命令验证您的功能是否正常工作。您可以在c插件中使用 `print`/`printTable` 函数来打印日志进行调试。

- PLUGIN:PreInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:PostInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:Available -> `vfox search <sdk-name>`
- PLUGIN:EnvKeys -> `vfox use <sdk-name>@<version>`



## 发布插件

当你完成插件并测试无误之后， 就可以直接[发起PR](https://github.com/version-fox/version-fox-plugins/pulls)啦~
