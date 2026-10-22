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

| 钩子                                            | **必须** | 描述                                    |
| :---------------------------------------------- | :------- | :-------------------------------------- |
| [hooks/available.lua](#available)               | ✅       | 列举所有可用版本                        |
| [hooks/pre_install.lua](#preinstall)            | ✅       | 解析版本号并返回预安装信息,如下载地址等 |
| [hooks/env_keys.lua](#envkeys)                  | ✅       | 配置环境变量                            |
| [hooks/post_install.lua](#postinstall)          | ❌       | 执行额外的操作, 如编译源码等            |
| [hooks/pre_use.lua](#preuse)                    | ❌       | 在切换版本之前, 提供修改版本的机会      |
| [hooks/parse_legacy_file.lua](#parselegacyfile) | ❌       | 自定义解析遗留文件                      |
| [hooks/pre_uninstall.lua](#preuninstall)        | ❌       | 删除之前进行额外操作                    |
| [hooks/fingerprint.lua](#fingerprint)           | ❌       | 报告能标识一次安装的事实，让对它的改动能被发现 |
| [hooks/detect.lua](#detect)                     | ❌       | 查找本机上 `vfox` 没有安装的 SDK，供 `vfox link --detect` 使用 |

## 必须实现的钩子函数

### PreInstall

返回预安装信息， 例如具体版本号、下载源等信息。 `vfox`会帮你提前将这些文件下载到特定目录下。如果是压缩包如`tar`、`tar.gz`、`tar.xz`、`zip`这四种压缩包， `vfox`会直接帮你解压处理。

`vfox`默认从下载链接中获取文件名，如果下载链接最后一项不是有效的文件名，可以通过在链接末尾附加 Fragment 来指定文件名，以便于`vfox`识别文件格式并解压。 如：`https://example.com/1234567890#/filename.zip`

如果版本的返回值为空，表示未找到版本，`vfox`会询问用户是否进行搜索操作。

**位置**: `hooks/pre_install.lua`

```lua
function PLUGIN:PreInstall(ctx)
    --- 用户输入
    local version = ctx.version
    return {
        --- 版本号
        version = "xxx",
        --- 文件地址, 可以是远程地址或者本地文件路径 [可选]
        url = "xxx",
        --- 下载链接的请求头 [可选]
        headers = {
            ["xxx"] = "xxx",
        },
        --- 备注信息 [可选]
        note = "xxx",
        --- SHA256 checksum [optional]
        sha256 = "xxx",
        --- md5 checksum [optional]
        md5= "xxx",
        --- sha1 checksum [optional]
        sha1 = "xxx",
        --- sha512 checksum [optional]
        sha512 = "xxx",
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

::: warning 缓存

`vfox`会缓存`Available`返回的结果, 默认缓存时间为`12h`。请查看[配置#缓存](../../guides/configuration.md#%E7%BC%93%E5%AD%98)。

:::

### EnvKeys

告诉`vfox`当前 SDK 需要配置的环境变量有哪些。

**位置**: `hooks/env_keys.lua`

```lua
function PLUGIN:EnvKeys(ctx)
    local mainSdkInfo = ctx.main
    local mainPath = mainSdkInfo.path
    local mversion = mainSdkInfo.version
    local mname = mainSdkInfo.name
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

::: warning
如果不需要这些钩子函数，请**删除对应的`.lua`文件**!
:::

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

    --- 已安装的 SDK 信息（使用版本字符串作为键）
    --- 例如，如果你安装了版本 "1.19.2"，使用：
    --- local sdkInfo = ctx.installedSdks['1.19.2']
    local sdkInfo = ctx.installedSdks[version]
    local path = sdkInfo.path
    local name = sdkInfo.name
    local sdkVersion = sdkInfo.version

    --- 当前工作目录
    local cwd = ctx.cwd

    --- 用户输入的 scope 信息，值为 global/project/session 其中之一
    local scope = ctx.scope

    --- 返回版本信息
    return {
        version = sdkVersion,
    }
end
```

### ParseLegacyFile

解析其他配置文件，以确定工具的版本。例如，`nvm` 的 `.nvmrc` 文件、`SDKMAN` 的 `.sdkmanrc` 文件等。

::: danger
该钩子函数必须配合 `legacyFilenames` 配置项使用, 告诉`vfox` 你的插件支持解析哪些文件。
:::

::: tip 策略配置
在实现此钩子函数时，你可以通过 `ctx.strategy` 获取用户配置的解析策略。详细的策略配置请参考[配置文档](../../guides/configuration.md#兼容版本文件)。
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
    --- 解析策略 (latest_installed, latest_available, specified)
    local strategy = ctx.strategy
    --- 获取当前插件已安装的版本列表
    local versions = ctx:getInstalledVersions()

    return {
        --- 返回具体版本号
        version = "x.y.z"
    }
end
```

### PreUninstall

在卸载 SDK 之前执行的钩子函数。如果插件需要在卸载之前执行一些操作，可以实现这个钩子函数。例如清理缓存、删除配置文件等。

**位置**: `hooks/pre_uninstall.lua`

```lua
function PLUGIN:PreUninstall(ctx)
    local mainSdkInfo = ctx.main
    local mainPath = mainSdkInfo.path
    local mversion = mainSdkInfo.version
    local mname = mainSdkInfo.name
    --- 其他 SDK 信息, PreInstall中返回的`addition`字段, 通过name获取
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end
```

### Fingerprint

有些 SDK 在安装之后会自己改动自己：`flutter upgrade` 会把它的 git 检出移到另一个提交，`npm install -g npm` 会替换 npm。`vfox` 按目录管理版本，光靠自己没办法发现 `flutter@3.47.4` 已经不再是 3.47.4。

这个钩子返回能标识一次安装的事实。`vfox` 会在安装完成之后（以及 `vfox link` 之后）调用它，把结果保存在该版本的回执里。之后 `vfox use` 和 `vfox doctor` 会再次调用它，事实不一致时给出警告，所有插件的提示内容和处理办法（`vfox install --reinstall`）都是一样的。

只返回能标识 SDK 本身的内容。用户本来就该做的改动，比如全局包，不能算在里面。没有什么可检查的时候返回 `nil`。

::: warning 注意
每次 `vfox use` 都会运行这个钩子。它只能读取，不能访问网络，而且必须很快（远低于 50 毫秒）。读几个文件是正确的做法，启动进程则不是。
:::

**位置**: `hooks/fingerprint.lua`

```lua
function PLUGIN:Fingerprint(ctx)
    --- 主运行时目录
    local path = ctx.path
    local version = ctx.version
    --- 其他 SDK 信息，与 PostInstall 中相同
    local sdkInfo = ctx.sdkInfo['sdk-name']

    --- 一个扁平的事实表，值按文本比较。
    return {
        facts = {
            head = read_git_head(path),
        },
    }
end
```

::: tip 链接的 SDK
用 `vfox link` 登记的 SDK 同样会记录事实，但它们发生变化时只提示一下：那个 SDK 不归 `vfox` 所有。
:::

### Detect

这个钩子用来找出本机上已经存在、但不是 `vfox` 安装的 SDK，例如系统包管理器、IDE 或 CI runner 装的 JDK。`vfox link --detect <sdk>` 会调用它，并把返回的内容链接进来。

返回 SDK 的目录（也就是 `EnvKeys` 里 `ctx.path` 期望的那个目录）和链接时使用的名字。名字会原样使用，所以请取用户会输入的名字，比如 `21.0.3-tem`。它可以包含字母、数字、`.`、`_`、`+` 和 `-`，不能是 `latest`。已经链接或已经安装的条目，`vfox` 会自动跳过。

**位置**: `hooks/detect.lua`

```lua
function PLUGIN:Detect(ctx)
    local found = {}
    for _, home in ipairs(candidate_homes()) do
        local release = read_release_file(home)
        if release then
            table.insert(found, {
                version = release.version .. "-" .. release.vendor,
                path = home,
                note = "found in " .. home, -- 可选
            })
        end
    end
    return found
end
```

### shadowHints

这是元数据，不是钩子。`vfox use` 之后，如果 SDK 的某个命令没有被运行，因为 `PATH` 里更靠前的位置还有别的可执行文件，`vfox` 会告诉用户。插件了解常见的“罪魁祸首”，所以可以说明该怎么处理。每一项匹配遮蔽它的可执行文件路径的一部分（不区分大小写），并附上一段说明，还可以附一个链接。

**位置**: `metadata.lua`

```lua
PLUGIN.shadowHints = {
    {
        match = "pnpm",
        message = "pnpm installed its own node. Run `pnpm remove -g node`.",
        url = "https://github.com/version-fox/vfox-nodejs/blob/main/docs/pnpm.md",
    },
}
```

检测“被遮蔽”本身由 `vfox` 对所有插件统一完成，不需要再在 `PreUse` 里扫描 `PATH`。

## 测试插件

直接对本地插件运行 Lua 断言，无需安装插件：

```shell
vfox plugin test .
vfox plugin run . PreInstall --input '{"version":"latest"}' --json
```

HTTP fixture、环境覆盖、hook 输入及 CI 边界请参阅[测试与调试插件](./testing.md)。真实安装测试应在一次性 CI runner 或 VM 中执行。

## 插件示例

https://github.com/version-fox/vfox-nodejs

你可以参考这个插件来开发自己的插件。

## 向官方插件存储库提交插件

`vfox`可以允许自定义安装插件，比如:

```shell
vfox add --source https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip
```

为了使你的用户更轻松，你可以将插件添加到官方插件存储库中，以列出你的插件并使用较短的命令轻松安装，比如 `vfox add nodejs`。

具体步骤请查看[如何将插件提交到索引仓库](./howto_registry.md)。
