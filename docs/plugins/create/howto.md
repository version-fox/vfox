# Create a Plugin

Plugins are the core of `vfox`. The plugin is the SDK, and the SDK is the plugin.

Use `Lua` script to provide the `vfox` plugin. The advantage of this method is:

- Low cost of plugin development; only need to have a basic understanding of Lua syntax.
- Decoupled from the platform; plugins can run on any platform, just put the plugin file in the specified directory.
- Plugins can be customized, shared, and used by others.
- The plugin can be shared with others.

## What's in the plugin?

The directory structure is as follows:
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

- `hooks` directory is used to store the plugin's hook functions. **One hook function corresponds to one `.lua` file.**
- `lib` directory is used to store the plugin's dependent libraries. `vfox` will automatically load all `.lua` files in this directory. **If placed in other directories, it will not be loaded.**
- `metadata.lua` Plugin metadata information. Used to describe the basic information of the plugin, such as the plugin name, version, etc.
- `README.md` Plugin documentation.
- `LICENSE` Plugin license.
- 

::: warning Plugin template
To facilitate the development of plugins, we provide a plugin template that you can use directly [vfox-plugin-template](https://github.com/version-fox/vfox-plugin-template) to develop a plugin.
:::

## Required hook functions

### PreInstall

This hook function is called before the installation of the SDK. It is used to return the pre-installation information,
such as
the specific version, download source, and other information. `vfox` will help you download these files to a specific
directory
in advance. If it is a compressed package such as `tar`, `tar.gz`, `tar.xz`, `zip`, `vfox` will help you to decompress
it directly.

```lua
function PLUGIN:PreInstall(ctx)
    --- input parameters
    local version = ctx.version
    --- the current version of vfox running
    local runtimeVersion = ctx.runtimeVersion
    return {
        --- sdk version
        version = "xxx",
        --- remote URL or local file path [optional]
        url = "xxx",
        --- note information [optional]
        note = "xxx",
        --- SHA256 checksum [optional]
        sha256 = "xxx",
        --- md5 checksum [optional]
        md5 = "xxx",
        --- sha1 checksum [optional]
        sha1 = "xxx",
        --- sha512 checksum [optional]
        sha512 = "xx",
        --- additional files [optional]
        addition = {
            {
                --- additional file name !
                name = "xxx",
                --- other same as above
                ...
            }
        }
    }
end
```


### Available

This hook function is called when the `vfox search` command is executed. It is used to return the current available
version
list. If there is no version, return an empty array.

```lua
function PLUGIN:Available(ctx)
    --- input parameters, array
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

### EnvKeys

It is used to return the environment variables that need to be configured when using the SDK.

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
        --- NOTE: If you need to set multiple PATH paths, just pass multiple PATHs, vfox will automatically deduplicate and set them in the order of configuration
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


## Optional hook functions

### PostInstall

This hook function is called after the `PreInstall` function is executed. It is used to execute additional operations,
such
as compiling source code, etc. Implement as needed.

```lua
function PLUGIN:PostInstall(ctx)
    --- SDK installation root path
    local rootPath = ctx.rootPath
    local runtimeVersion = ctx.runtimeVersion
    ---  Get it from the name returned by PreInstall
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end
```

### PreUse

When the user uses `vfox use`, the plugin's `PreUse` function is called. The purpose of this function is to return the
version information entered by the user. If the `PreUse` function returns version information, `vfox` will use this new
version.

```lua
function PLUGIN:PreUse(ctx)
    local runtimeVersion = ctx.runtimeVersion
    --- user input version
    local version = ctx.version
    --- user current used version
    local previousVersion = ctx.previousVersion

    --- installed sdks
    local sdkInfo = ctx.installedSdks['version']
    local path = sdkInfo.path
    local name = sdkInfo.name
    local version = sdkInfo.version

    --- working directory
    local cwd = ctx.cwd

    --- user input scope
    --- could be one of global/project/session
    local scope = ctx.scope

    --- return the version information
    return {
        version = version,
    }
end
```

## Test Plugin

Currently, VersionFox plugin testing is straightforward. You only need to place the plugin file in the
`${HOME}/.version-fox/plugins` directory and verify that your features are working using different commands. You can use
`print`/`printTable` statements in Lua scripts for printing log.

- PLUGIN:PreInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:PostInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:Available -> `vfox search <sdk-name>`
- PLUGIN:EnvKeys -> `vfox use <sdk-name>@<version>`

## Publish to the public registry

`vfox` allows custom installation of plugins, such as `vfox add --source https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip`, --alias and --source not support for multiple plugins.

In order to make it easier for your users, you can add the plugin to the public registry to list your plugin and easily install it with shorter commands, such as `vfox add nodejs`.

For details, see [How to submit a plugin to the public registry](./howto_registry.md).
