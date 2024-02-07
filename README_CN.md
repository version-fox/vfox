<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>
<h1 style="margin-top: -40px">VersionFox</h1>

[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/version-fox/vfox)
[![GitHub License](https://img.shields.io/github/license/version-fox/vfox?style=for-the-badge)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/version-fox/vfox?display_name=tag&style=for-the-badge)](https://github.com/version-fox/vfox/releases)
[![Discord](https://img.shields.io/discord/1191981003204477019?style=for-the-badge&logo=discord)](https://discord.gg/MUfedBm9)



[[English]](./README.md)  [[中文文档]](./README_CN.md)

## 介绍

VersionFox 是一个跨平台的用于管理SDK版本的工具，它允许你通过命令行快速安装和切换不同版本的SDK, 并且SDK是以Lua脚本形式作为插件进行提供,
也就是说这允许你实现自己的SDK来源, 或者使用别人分享的插件来安装SDK. 这都取决于你的想象力. ;)

## 安装

### macOS

在macOS上,你可以使用Homebrew来快速安装`vfox`:

```bash
$ brew tap version-fox/tap
$ brew install vfox
```

或者如果没有安装Homebrew, 你可以直接下载二进制文件:

```bash
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

### Linux

- 使用APT安装

  <details><summary><code>sudo apt install vfox</code></summary>

  ```sh
   echo "deb [trusted=yes] https://apt.fury.io/versionfox/ /" | sudo tee /etc/apt/sources.list.d/versionfox.list
   sudo apt-get update
   sudo apt-get install vfox
  ```

  </details>

- 使用YUM安装

  <details><summary><code>sudo yum install vfox</code></summary>

   ```sh
    echo '[vfox]
   name=VersionFox Repo
   baseurl=https://yum.fury.io/versionfox/
   enabled=1
   gpgcheck=0' | sudo tee /etc/yum.repos.d/trzsz.repo

    sudo yum install vfox
    ```

  </details>

当然,你也可以直接下载二进制文件:
```bash
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

### Windows
对于Windows用户, 有两种方式进行安装:
1. 下载setup.ext安装器进行安装
2. 直接下载二进制文件, 将它**配置到PATH**中


### ⚠️⚠️⚠️将 vfox 挂到你的 shell 中（从下面条目中,选择适合你 shell 的版本）：
```bash
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish

# 对于Powershell, 将以下行添加到你的 $PROFILE中:
Invoke-Expression "$(vfox activate pwsh)"
```

### 演示(Nodejs)
[![asciicast](https://asciinema.org/a/630778.svg)](https://asciinema.org/a/630778)



### 0. 获取官方可用插件列表
**Command** : `vfox available [<category>]`
```bash
$ vfox available
Name                      Version         Author          Description
flutter/flutter          0.0.1           Han Li          flutter plugin, support for getting stable, dev, beta version
java/adoptium-jdk        0.0.1           aooohan         Adoptium JDK
...

Please use vfox add <plugin name> to install plugin
```
VersionFox 维护一个集中的官方插件库，其中包含所有各种插件。

> 注意: 插件名由两部分组成: 分类/插件名

### 1. 安装插件

在VersionFox的理念里, 插件就是SDK, SDK就是插件. 所以, 在使用之前, 你需要安装对应的插件.

VersionFox支持安装官方插件和自定义插件, 安装插件的命令如下:

- `vfox add [--alias <sdk-name>] <plugin-name>`: 此命令从官方软件仓库安装插件。还可以为插件设置别名。
- `vfox add [--source <url/path>] <sdk-name>`: 此命令从指定路径或 URL 安装插件并为其命名。

```bash
$ vfox add --alias node nodejs/nodejs
Adding plugin from https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/nodejs/nodejs.lua...
Checking plugin...
Plugin info:
Name    -> nodejs
Author  -> Lihan
Version -> 0.0.1
Path    -> /${HOME}/.version-fox/plugins/node.lua
Add node plugin successfully! 
Please use `vfox install node@<version>` to install the version you need.
```

VersionFox 对插件的安装来源没有限制，这意味着您可以添加自定义插件或使用他人共享的插件。

```bash
$ vfox add --source https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/nodejs/nodejs.lua custom-node
Adding plugin from https://raw.githubusercontent.com/version-fox/version-fox-plugins/main/nodejs/nodejs.lua...
Checking plugin...
Plugin info:
Name    -> nodejs
Author  -> Lihan
Version -> 0.0.1
Path    -> /${HOME}/.version-fox/plugins/custom-node.lua
Add custom-node plugin successfully! 
Please use `vfox install custom-node@<version>` to install the version you need.
```


### 2. 获取SDK的可用版本

在安装好对应的插件之后, 你可以通过`vfox search <sdk-name>`命令来获取该SDK的可用版本,例如:

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

在这里,你可以通过上下键来选择你想要安装的版本,然后按下回车键来确认你的选择,如果你想要查看更多的版本,你可以按下左右键来翻页.

### 3. 安装SDK

VersionFox安装SDK的方式有两种:
1. 即上一步,通过`vfox search <sdk-name>`命令来获取该SDK的可用版本,通过方向键选择目标版本,最后按下回车键来确认你的选择.
2. 通过`vfox install <sdk-name>@<version>`或`vfox i <sdk-name>@<version>`命令来直接安装指定版本的SDK,例如:

```bash
$ vfox install node@20.10.0
Installing node@20.10.0...
Downloading... 100% [===========] (6.7 MB/s)        
Unpacking ${HOME}/.version-fox/cache/node/node-v20.10.0-darwin-x64.tar.gz...
Install node@20.10.0 success! 
Please use vfox use node@20.10.0 to use it.
```

不论在那种平台下, VersionFox的都会将SDK统一安装到`${HOME}/.version-fox/cache`
目录下并以`<sdk-name>`进行划分,以下是目录结构:

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

### 4. 使用或切换SDK版本

1. 通过`vfox use <sdk-name>@<version>`命令来使用指定版本的SDK,例如:

```bash
$ vfox use node@20.10.0
Now using node@20.10.0
```

2.通过`vfox use <sdk-name>`列举当前已安装的所有版本,你可以通过上下键来选择或者直接输入版本号进行模糊搜索,然后按下回车键来确认你的选择,例如:

```bash
$ vfox use node
Please select a version of node [type to search]: 
   8.16.2
-> 20.10.0
Now using node@20.10.0
```

这是最频繁使用的命令之一! 如果嫌弃命令过长,你可以使用`vfox u <sdk-name>`来代替.
至此,你已经成功安装并使用了你想要的版本了! 是不是非常简单! 命令是所有平台通用!
你不需要因为平台的不同而记许多不同的命令了!

## 更多命令

当然了, VersionFox的功能远不止于此!!!

### 卸载指定版本SDK

命令: `vfox uninstall <sdk-name>@<version>`或`vfox un <sdk-name>@<version>`

```bash
$ vfox un node@20.10.0
Uninstall node@20.10.0 success!
```

### 查看已安装的SDK版本

#### 特定SDK的版本列表

命令: `vfox list <sdk-name>`或`vfox ls <sdk-name>`

```bash
$ vfox ls node
-> 20.10.0 (current)
-> 18.10.0
...
```

#### 所有SDK的版本列表

命令: `vfox list`或`vfox ls`

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

### 查看当前使用的SDK

#### 指定SDK的当前版本

命令: `vfox current <sdk-name>`或`vfox c <sdk-name>`

```bash
$ vfox c node
-> v20.10.0
```

#### 所有SDK的当前版本

命令: `vfox current`或`vfox c`

```bash
$ vfox c
node -> v20.10.0
java -> v11.0.12
```

### 查看插件信息

命令: `vfox info <sdk-name>`

```bash
$ vfox info node
```

### 卸载插件

命令: `vfox remove <sdk-name>`

```bash
$ vfox remove node
```

### 更新插件

命令: `vfox update <sdk-name>`

```bash
$ vfox update node
```

## 插件系统

在VersionFox中,插件即SDK,SDK即插件. VersionFox的插件是以Lua脚本的形式进行提供的,这样做的好处是:

- 插件的开发成本低,只需要了解一点点Lua语法即可.
- 与平台解耦,插件可以在任何平台下运行,只需要将插件文件放到指定目录即可.
- 插件可以在不同平台下共享, 一次编写,到处运行.
- 可以自定义、可与他人分享、可使用他人分享的插件.

### 插件开发

#### 插件结构

```lua

--- VersionFox 提供的常用库(选用)
local http = require("http")
local json = require("json")
local html = require("html")

--- 以下两个参数由VersionFox在运行时注入
--- 运行时OS类型 (Windows, Linux, Darwin)
OS_TYPE = ""
--- 运行时OS架构 (amd64, arm64等)
ARCH_TYPE = ""

PLUGIN = {
    --- 插件名称, 即sdk名称
    name = "java",
    --- 插件作者
    author = "Lihan",
    --- 插件版本
    version = "0.0.1",
    --- 插件描述
    description = "xxx",
    -- 升级地址
    updateUrl = "{URL}/sdk.lua",
    -- 最先运行时的版本 >=
    minRuntimeVersion = "0.2.2",
}

--- 根据ctx.version来返回对应版本的信息,包括版本、下载地址等
--- @param ctx table
--- @field ctx.version string 用户输入的版本号
--- @return table 版本信息
function PLUGIN:PreInstall(ctx)
    local runtimeVersion = ctx.runtimeVersion
    local version = ctx.version
    return {
        --- 版本号 必填
        version = "xxx",
        --- 下载地址 [选填]
        url = "xxx",
    }
end

--- 拓展点,会在PreInstall执行之后调用,可以在这里进行一些额外的操作, 针对SDK安装目录的文件操作等
--- 目前可不实现!
function PLUGIN:PostInstall(ctx)
    --- ctx.rootPath SDK安装目录
    local rootPath = ctx.rootPath
    local runtimeVersion = ctx.runtimeVersion
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end

--- 返回当前插件所提供的所有可用版本
--- @param ctx table 没有任何字段属性, 仅作为一个空table传入, 方便后续拓展
--- @return table 可用版本的描述以及附带工具的描述, additional可不传, 例如
--- {
---   {
---     version = "20.10.0",
---     note = "LTS",
---     addition = {
---       {
---         name = "npm",
---         version = "8.8.8",
---       }
---     }
---   }
--- }
function PLUGIN:Available(ctx)
    local runtimeVersion = ctx.runtimeVersion
    return {
        {
            version = "xxxx",
            note = "LTS",
            addition = {
                {
                    name = "npm",
                    version = "xxx",
                }
            }
        }
    }
end

--- 每种SDK配置的环境变量会有所不同, 这里允许插件自定义环境变量(包括PATH的设置)
--- 注意: 记得区分不同平台的环境变量设置哦!
--- @param ctx table 上下文信息
--- @field ctx.version_path string SDK安装目录
function PLUGIN:EnvKeys(ctx)
    local mainPath = ctx.path
    local runtimeVersion = ctx.runtimeVersion
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

#### 如何测试插件

目前VersionFox插件测试很暴力也很简单, 你只需要将插件文件放到`${HOME}/.version-fox/plugins`目录下, 通过不同的命令来验证你的功能是否正常即可.
可以在lua脚本中使用`print`来输出调试信息.

- PLUGIN:PreInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:PostInstall -> `vfox install <sdk-name>@<version>`
- PLUGIN:Available -> `vfox search <sdk-name>`
- PLUGIN:EnvKeys -> `vfox use <sdk-name>@<version>`

#### 插件提供的能力

##### 1. HTTP请求库

VersionFox提供了一个简单的HTTP请求库,目前仅支持GET请求, 后续可能会拓展其他类型, 你可以通过`require("http")`来使用它,
例如:

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

##### 2. JSON库

```lua
local json = require("json")

local obj = { "a", 1, "b", 2, "c", 3 }
local jsonStr = json.encode(obj)
local jsonObj = json.decode(jsonStr)
for i = 1, #obj do
    assert(obj[i] == jsonObj[i])
end
```

##### 3. HTML库

VersionFox所提供的HTML库是基于[goquery](https://github.com/PuerkitoBio/goquery)进行的部分功能封装,
你可以通过`require("html")`
来使用它,例如:

```lua
local html = require("html")
local doc = html.parse("<html><body><div id='test'>test</div><div id='t2'>456</div></body></html>")
local div = doc:find("body"):find("#t2")
print(div:text() == "456")
```

### 插件仓库

VersionFox对于插件的来源是没有任何限制的,你可以使用任何你想要的插件,只要它符合VersionFox的插件规范即可.
为了方便共享和使用,我们还是提供了一个插件仓库[version-fox-plugin](https://github.com/version-fox/version-fox-plugins)
,你可以在这里找到一些常用的插件,当然了,你也可以将你的插件分享到这个仓库中来.

## 命令一览表

```bash
vfox - VersionFox, a tool for sdk version management
vfox add <sdk-name> <url/path>  Add a plugin from url or path
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

## 进行贡献

有了您的贡献，开放源码社区才能成为学习、启发和创造的绝佳场所。我们非常感谢您的任何贡献!

1. Fork当前项目
2. 创建你的开发分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的变更 (`git commit -m 'Add some AmazingFeature'`)
4. 推送你的开发分支 (`git push origin feature/AmazingFeature`)
5. 创建一个新的Pull Request

插件贡献, 请移步[version-fox-plugins](https://github.com/version-fox/version-fox-plugins).

## 许可证

根据 Apache 2.0 许可证分发, 请参阅 [LICENSE](./LICENSE) 了解更多信息。

