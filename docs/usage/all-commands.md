# All Commands

```shell
vfox - vfox is a tool for runtime version management.
vfox available List all available plugins
vfox add [--alias <sdk-name> --source <url/path> ] <plugin-name>  Add a plugin or plugins from official repository or custom source, --alias` and `--source` are not supported when adding multiple plugins.
vfox remove <sdk-name>          Remove a plugin
vfox update [<sdk-name> | --all] Update a specified or all plugin(s)
vfox info <sdk-name>[@<version>] [options]  Show plugin info or SDK path with optional formatting
vfox search <sdk-name>          Search available versions of a SDK
vfox install <sdk-name>@<version> Install the specified version of SDK
vfox uninstall <sdk-name>@<version> Uninstall the specified version of SDK
vfox link <sdk-name>@<version> <path>  Use an SDK that is already on this machine as a version (vfox link --detect <sdk-name> lets the plugin find them)
vfox alias set|list|unset           Give installed versions short names of your own
vfox doctor [<sdk-name>[@<version>]] Check vfox and the installed SDKs for problems
vfox use [--global --project --session] <sdk-name>[@<version>]   Use the specified version of SDK for different scope
vfox unuse [--global --project --session] <sdk-name>   Unset the version of SDK from specified scope
vfox exec <sdk-name>[@<version>]... -- <command> [args...]   Execute a command in vfox managed environment
vfox list [<sdk-name>]              List all installed versions of SDK
vfox current [<sdk-name>]           Show the current version of SDK
vfox config [<key>] [<value>]       Setup, view config
vfox cd [--plugin] [<sdk-name>]     Launch a shell in the VFOX_HOME, SDK directory, or plugin directory
vfox upgrade                    Upgrade vfox to the latest version
vfox --version                  Show the installed vfox version
vfox -v                         Alias for vfox --version
vfox -V                         Alias for vfox --version
vfox help                      Show this help message
```
