# 所有命令

```shell
vfox - vfox is a tool for runtime version management.
vfox available      List all available plugins
vfox add [--alias <sdk-name> --source <url/path> ] <plugin-name>  Add a plugin or plugins from official repository or custom source, `--alias` and `--source` are not supported when adding multiple plugins.
vfox remove <sdk-name>          Remove a plugin
vfox update [<sdk-name> | --all] Update a specified or all plugin(s)
vfox info <sdk-name>[@<version>] [options]  显示插件信息或 SDK 路径，支持格式化选项
vfox search <sdk-name>          Search available versions of a SDK
vfox install <sdk-name>@<version> Install the specified version of SDK
vfox uninstall <sdk-name>@<version> Uninstall the specified version of SDK
vfox link <sdk-name>@<version> <path>  把本机上已有的 SDK 当作一个版本来用（vfox link --detect <sdk-name> 让插件自己去找）
vfox alias set|list|unset           给已安装的版本起自己的短名字
vfox doctor [<sdk-name>[@<version>]] 检查 vfox 和已安装的 SDK 有没有问题
vfox use [--global --project --session] <sdk-name>[@<version>]   Use the specified version of SDK for different scope
vfox unuse [--global --project --session] <sdk-name>   Unset the SDK version from the specified scope
vfox exec <sdk-name>[@<version>]... -- <command> [args...]   Execute a command in vfox managed environment
vfox list [<sdk-name>]              List all installed versions of SDK
vfox current [<sdk-name>]           Show the current version of SDK
vfox config [<key>] [<value>]       Setup, view config
vfox cd [--plugin] [<sdk-name>]     Launch a shell in the VFOX_HOME, SDK directory, or plugin directory
vfox upgrade                    Upgrade vfox to the latest version
vfox --version                  显示当前安装的 vfox 版本
vfox -v                         vfox --version 的别名
vfox -V                         vfox --version 的别名
vfox help                      Show this help message
```
