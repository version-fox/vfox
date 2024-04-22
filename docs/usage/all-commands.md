# All Commands

```shell
vfox - VersionFox, a tool for sdk version management
vfox available List all available plugins
vfox add [--alias <sdk-name> --source <url/path> ] <plugin-name>  Add a plugin or plugins from offical repository or custom source, --alias` and `--source` are not supported when adding multiple plugins.
vfox remove <sdk-name>          Remove a plugin
vfox update [<sdk-name> | --all] Update a specified or all plugin(s)
vfox info <sdk-name>            Show plugin info
vfox search <sdk-name>          Search available versions of a SDK
vfox install <sdk-name>@<version> Install the specified version of SDK
vfox uninstall <sdk-name>@<version> Uninstall the specified version of SDK
vfox use [--global --project --session] <sdk-name>[@<version>]   Use the specified version of SDK for different scope
vfox list [<sdk-name>]              List all installed versions of SDK
vfox current [<sdk-name>]           Show the current version of SDK
vfox config [<key>] [<value>]       Setup, view config
vfox cd [--plugin] [<sdk-name>]     Launch a shell in the VFOX_HOME, SDK directory, or plugin directory
vfox help                      Show this help message
```
