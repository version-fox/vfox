<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>
<h1 style="margin-top: -40px">VersionFox</h1>

[![Go Report Card](https://goreportcard.com/badge/github.com/version-fox/vfox)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub](https://img.shields.io/github/license/version-fox/vfox)](https://wimg.shields.io/github/license/version-fox/vfox)
[![GitHub release](https://img.shields.io/github/v/release/version-fox/vfox)](https://github.com/version-fox/vfox/releases/latest)




[[English]](./README.md)  [[中文文档]](./README_CN.md)


## Introduction

VersionFox is a cross-platform tool for managing SDK versions. It allows you to quickly install
and switch between different versions of SDKs using the command line.
SDKs are provided as plugins in the form of Lua scripts. This means you can implement your own SDK sources or use
plugins shared by others to install SDKs. It all depends on your imagination. ;)

## Why use VersionFox?

- **cross-platform support** (Windows, Linux, macOS)
- single CLI for multiple languages
- **consistent commands** to manage all your languages
- simple **plugin system** to add support for your language of choice
- shell completion available for common shells (Bash, Zsh, Powershell)
- **it's similar to `asdf-vm`, but offers more simple commands and genuine cross-platform unification.**

## Installation

### macOS

On macOS, you can use Homebrew to quickly install `vfox`:

```bash
$ brew tap version-fox/tap
$ brew install vfox
```

If Homebrew is not installed, you can download the binary directly:

```bash
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

### Linux

- Install with APT

  <details><summary><code>sudo apt install vfox</code></summary>

  ```sh
   echo "deb [trusted=yes] https://apt.fury.io/versionfox/ /" | sudo tee /etc/apt/sources.list.d/versionfox.list
   sudo apt-get update
   sudo apt-get install vfox
  ```

  </details>
  
- Install with YUM

  <details><summary><code>sudo apt install vfox</code></summary>

   ```sh
    echo '[vfox]
   name=VersionFox Repo
   baseurl=https://yum.fury.io/versionfox/
   enabled=1
   gpgcheck=0' | sudo tee /etc/yum.repos.d/trzsz.repo

    sudo yum install vfox
    ```

  </details>

others, you can download the binary directly:
```bash
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

### Windows

For Windows users, please follow the steps below to install:

1. Navigate to the [Releases](https://github.com/version-fox/vfox/releases) page of this repository.
2. Download the latest `setup.exe` file.
3. Once the download is complete, double-click the `setup.exe` file to start the installation process.
4. Follow the prompts in the installation wizard to complete the installation.

Please ensure that you have the necessary permissions to install software on your machine.

## Usage

[![asciicast](https://asciinema.org/a/629106.svg)](https://asciinema.org/a/629106)

### 0. List Available Plugins
**Command** : `vfox available [<category>]`
```bash
$ vfox available
Name                     Version         Author          Description
flutter/flutter          0.0.1           Han Li          flutter plugin, support for getting stable, dev, beta version
java/adoptium-jdk        0.0.1           aooohan         Adoptium JDK
...

Please use vfox add <plugin name> to install plugin
```
VersionFox maintains a centralized official plugin repository, which houses all the supported plugins. 
You can use a specific command to check the currently supported plugins.

> NOTE: The naming convention for these plugins is divided into two parts: the category and the plugin name, separated by a slash (/).

### 1. Install Plugin (SDK)

In VersionFox, plugins are SDKs, and SDKs are plugins. So, before using them, you need to install the corresponding
plugin. 

VersionFox supports the installation of both official and custom plugins.

 - `vfox add [--alias <sdk-name>] <plugin-name>`: This command installs a plugin from the official repository. An alias for the plugin can also be set.

 - `vfox add [--source <url/path>] <sdk-name>`: This command installs a plugin from a specified path or URL and assigns it a name.

```bash
$ vfox add --alias node node/node
Adding plugin from https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/node/node.lua...
Checking plugin...
Plugin info:
Name    -> node
Author  -> Lihan
Version -> 0.0.1
Path    -> /${HOME}/.version-fox/plugins/node.lua
Add node plugin successfully! 
Please use `vfox install node@<version>` to install the version you need.
```

VersionFox has no restrictions on the installation sources of plug-ins that means you can add your custom plugins
or use plugins shared by others. 

```bash
$ vfox add --source https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/node/node.lua custom-node
Adding plugin from https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/node/node.lua...
Checking plugin...
Plugin info:
Name    -> node
Author  -> Lihan
Version -> 0.0.1
Path    -> /${HOME}/.version-fox/plugins/custom-node.lua
Add custom-node plugin successfully! 
Please use `vfox install custom-node@<version>` to install the version you need.
```

### 2. Get Available Versions of SDK

After installing the corresponding plugin, you can use the `vfox search <sdk-name>` command to get the available versions
of that SDK. For example:

```bash
$ vfox search node
Please select a version of node [type to search]: 
->  v21.4.0 [npm v10.2.4]
   ...
   v20.10.0 (LTS) [npm v10.2.3]
   v20.9.0 (LTS) [npm v10.1.0]
   ...
   v20.1.0 [npm v9.6.4] v20.0.0 [npm v9.6.4]
Press ↑/↓ to select and press ←/→ to page, and press Enter to confirm

```

Here, you can use the up and down arrow keys to select the version you want to install, and then press Enter to confirm
your choice. If you want to view more versions, you can use the left and right arrow keys to navigate.

### 3. Install SDK

VersionFox provides two ways to install SDKs:

1. Similar to the previous step, use the `vfox search <sdk-name>` command to get the available versions of the SDK. Use
   the
   arrow keys to select the target version and press Enter to confirm your choice.
2. Use the `vfox install <sdk-name>@<version>` or `vfox i <sdk-name>@<version>` command to directly install the specified
   version of the SDK. For example:

```bash
$ vfox install node@20.10.0
Installing node@20.10.0...
Downloading... 100% [===========] (6.7 MB/s)        
Unpacking ${HOME}/.version-fox/cache/node/node-v20.10.0-darwin-x64.tar.gz...
Install node@20.10.0 success! 
Please use vfox use node@20.10.0 to use it.
```

Regardless of the platform, VersionFox will install SDKs in a unified directory `${HOME}/.version-fox/cache`, divided
by `<
sdk-name>`. Here is the directory structure:

```bash
    ${HOME}/.version-fox/cache
    ├── node
    │   ├── v20.10.0
    │   ├── v18.10.0
    ├── java
    │   ├── v11.0.12
    │   ├── v8.0.302
    ....
```

### 4. Use or Switch SDK Version

1. Use the `vfox use <sdk-name>@<version>` command to use the specified version of the SDK. For example:
    ```bash
    $ vfox use node@20.10.0
    Now using node@20.10.0
    ```
2. Use the `vfox use <sdk-name>` command to list all installed versions. You can use the up and down arrow keys to select
   or directly enter the version number for fuzzy search. Then, press Enter to confirm your choice. For example:
    ```bash
    $ vfox use node
    Please select a version of node [type to search]: 
       8.16.2
    -> 20.10.0
    Now using node@20.10.0
    ```

This is one of the most frequently used commands! If you find the command too long, you can use `vfox u <sdk-name>` as a
shortcut. Congratulations, you have successfully installed and used the version you want! The commands are universal
across all platforms, and you don't need to remember different commands for different platforms!

## More Commands

Of course, VersionFox offers more features!

### Uninstall Specific SDK Version

Command: `vfox uninstall <sdk-name>@<version>` or `vfox un <sdk-name>@<version>`

```bash
$ vfox un node@20.10.0
Uninstall node@20.10.0 success!
```

### View Installed SDK Versions

#### List Versions of a Specific SDK

Command: `vfox list <sdk-name>` or `vfox ls <sdk-name>`

```bash
$ vfox ls node
-> 20.10.0 (current)
-> 18.10.0
...
```

#### List Versions of All SDKs

Command: `vfox list` or `vfox ls`

```bash
$ vfox ls
All installed sdk versions
└─┬node
  ├──v8.16.2
  └──v20.10.0
└─┬java
  ├──v8.0.302
  └──v11.0.12
...
```

### View Current SDK Version

#### Current Version of a Specific SDK

Command : `vfox current <sdk-name>` or `vfox c <sdk-name>`

```bash
$ vfox c node
-> v20.10.0
```

#### Current Versions of All SDKs

Command: `vfox current` or `vfox c`

```bash
$ vfox c
node -> v20.10.0
java -> v11.0.12
```

### View Plugin Information

Command: `vfox info <sdk-name>`

```bash
$ vfox info node
```

### Remove Plugin

Command: `vfox remove <sdk-name>`

```bash
$ vfox remove node
```

### Update Plugin

Command: `vfox update <sdk-name>`

```bash
$ vfox update node
```

## Plugin System

In VersionFox, a plugin is equivalent to an SDK, and an SDK is equivalent to a plugin. VersionFox plugins are provided
in the form of Lua scripts. The benefits of this approach are:

- Low development cost for plugins; only a basic understanding of Lua syntax is needed.
- Decoupled from platforms; plugins can run on any platform by placing the plugin file in the specified directory.
- Plugins can be shared across different platforms; write once, run anywhere.
- Customizable, shareable, and can use plugins shared by others.

### Plugin Development

#### Plugin Structure

```lua

--- Common libraries provided by VersionFox (optional)
local http = require("http")
local json = require("json")
local html = require("html")

--- The following two parameters are injected by VersionFox at runtime
--- Operating system type at runtime (Windows, Linux, Darwin)
OS_TYPE = ""
--- Operating system architecture at runtime (amd64, arm64, etc.)
ARCH_TYPE = ""

PLUGIN = {
    --- Plugin name, eg java, adoptium_jdk, etc.
    --- NOTE: Use only underscores as hyphens.
    name = "java",
    --- Plugin author
    author = "Lihan",
    --- Plugin version
    version = "0.0.1",
    description = "xxxxxx",
    -- Update URL
    updateUrl = "{URL}/sdk.lua",
}

--- Return information about the specified version based on ctx.version, including version, download URL, etc.
--- @param ctx table
--- @field ctx.version string User-input version
--- @return table Version information
function PLUGIN:PreInstall(ctx)
    return {
        --- Version number
        version = "xxx",
        --- Download URL, support tar.gz tar.xz zip three formats
        url = "xxx",
        --- You just choose one of the checksum algorithms.
        --- SHA256 checksum, first choice!
        sha256 = "xxx",
        --- sha1 checksum [optional]
        sha1= "xxx",
        --- sha512 checksum [optional]
        sha512= "xxx",
        --- md5 checksum [optional]
        md5= "xxx",
    }
end

--- Extension point, called after PreInstall, can perform additional operations, 
--- such as file operations for the SDK installation directory
--- Currently can be left unimplemented!
function PLUGIN:PostInstall(ctx)
    --- ctx.rootPath SDK installation directory
    local rootPath = ctx.rootPath
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end

--- Return all available versions provided by this plugin
--- @param ctx table Empty table used as context, for future extension
--- @return table Descriptions of available versions and accompanying tool descriptions
function PLUGIN:Available(ctx)
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

--- Each SDK may have different environment variable configurations. 
--- This allows plugins to define custom environment variables (including PATH settings)
--- Note: Be sure to distinguish between environment variable settings for different platforms!
--- @param ctx table Context information
--- @field ctx.path string SDK installation directory
function PLUGIN:EnvKeys(ctx)
    local mainPath = ctx.path
    return {
        {
            key = "JAVA_HOME",
            value = mainPath
        },
        {
            key = "PATH",
            value = mainPath .. "/bin"
        }
    }
end

```

#### How to Test Plugins

Currently, VersionFox plugin testing is straightforward. You only need to place the plugin file in the
`${HOME}/.version-fox/plugins` directory and verify that your features are working using different commands. You can use
`print` statements in Lua scripts for printing log.

- PLUGIN:PreInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:PostInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:Available -> `vfox search <sdk-name>`
- PLUGIN:EnvKeys -> `vfox use <sdk-name>@<version>`

#### Capabilities Provided by VersionFox

##### 1. HTTP Request Library

VersionFox provides a simple HTTP request library, currently supporting only GET requests. In the Lua script, you can
use `require("http")` to access it. For example:

```lua
local http = require("http")
assert(type(http) == "table")
assert(type(http.get) == "function")
local resp, err = http.get({
    url = "http://ip.jsontest.com/"
})
assert(err == nil)
assert(resp.status_code == 200)
assert(resp.headers['Content-Type'] == 'application/json')
assert(resp.body == '{"ip": "xxx.xxx.xxx.xxx"}')
```

##### 2. JSON Library

Based on [gopher-json](https://github.com/layeh/gopher-json/)

```lua
local json = require("json")

local obj = { "a", 1, "b", 2, "c", 3 }
local jsonStr = json.encode(obj)
local jsonObj = json.decode(jsonStr)
for i = 1, #obj do
    assert(obj[i] == jsonObj[i])
end
```

##### 3. HTML Library

The HTML library provided by VersionFox is based on [goquery](https://github.com/PuerkitoBio/goquery), with some
functionality encapsulated. You can use `require("html")` to access it, for example:

```lua
local html = require("html")
local doc = html.parse("<html><body><div id='test'>test</div><div id='t2'>456</div></body></html>")
local div = doc:find("body"):find("#t2")
print(div:text() == "456")
```

### Plugin Repository

VersionFox has no restrictions on the source of plugins; you can use any plugin as long as it complies with VersionFox
plugin specifications. To facilitate sharing and use, we also provide a plugin
repository [version-fox-plugin](https://github.com/version-fox/version-fox-plugins), where you
can find some commonly used plugins. Of course, you can also share your plugins in this repository.


## Command Overview

```bash
vfox - VersionFox, a tool for sdk version management
vfox available [<category>]     List all available plugins
vfox add [--alias <sdk-name> --source <url/path> ] <plugin-name>  Add a plugin from offical repository or custom source
vfox remove <sdk-name>          Remove a plugin
vfox update <sdk-name>          Update a plugin
vfox info <sdk-name>            Show plugin info
vfox search <sdk-name>          Search available versions of a SDK
vfox install <sdk-name>@<version> Install the specified version of SDK
vfox uninstall <sdk-name>@<version> Uninstall the specified version of SDK
vfox use <sdk-name>@<version>   Use the specified version of SDK
vfox use <sdk-name>             Select the version to use
vfox list <sdk-name>              List all installed versions of SDK
vfox list                      List all installed versions of all SDKs
vfox current <sdk-name>           Show the current version of SDK
vfox current                   Show the current version of all SDKs
vfox help                      Show this help message
```

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any
contributions you make are greatly appreciated.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a pull request

Plugin Contributions, please go to[version-fox-plugins](https://github.com/version-fox/version-fox-plugins).

## License

Distributed under the Apache 2.0 License. See [LICENSE](./LICENSE) for more information.

