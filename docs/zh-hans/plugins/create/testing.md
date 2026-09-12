# 测试与调试插件

使用 `vfox plugin test` 对本地插件运行普通 Lua 断言，使用 `vfox plugin run` 查看单个 hook 的结果。两者复用 core 的插件加载器、内置模块和强类型 hook 转换，不会注册插件、安装 SDK 或初始化用户及项目的 vfox 配置。

## 编写测试

在插件仓库新增 `tests/available_test.lua`。下面的测试针对 vfox 仓库内的[示例插件](https://github.com/version-fox/vfox/tree/main/examples/plugins/sample)：

```lua
local test = require("vfox.test")
local plugin = test.load({
    os = "linux",
    arch = "amd64",
    env = { VFOX_SAMPLE_MIRROR = false },
    http = function(request)
        assert(request.method == "GET")
        assert(request.url == "https://example.com/releases")
        return {
            status_code = 200,
            body = '{"versions":["1.2.3","1.2.2"]}',
            headers = {},
        }
    end,
})

local versions = plugin:Available({})
assert(#versions == 2)
assert(versions[1].version == "1.2.3")
local archive = plugin:PreInstall({ version = "latest" })
assert(archive.version == "1.2.3")
assert(archive.url == "https://example.com/sample-1.2.3-linux-amd64.zip")
```

在插件仓库根目录运行：

```shell
vfox plugin test .
vfox plugin test . --file tests/available_test.lua
vfox plugin test . --online --timeout 45s
```

默认按路径顺序发现 `tests/**/*_test.lua`。每个文件拥有独立的 Lua 全局变量、模块缓存、环境覆盖和 HTTP 状态。每个文件调用一次 `test.load()`，返回的对象通过 core 正式的强类型方法调用 hook。未捕获的断言失败会停止当前文件，其他文件继续运行。命令输出文件级 `PASS`/`FAIL` 和错误位置；任何文件失败、没有测试文件或文件未调用 `test.load()`，都会非零退出。

`--file` 相对于插件目录，也可以使用绝对文件路径。命令保留当前工作目录。为了让 fixture 在任意工作目录都能读取，可用普通 `io.open` 打开 `RUNTIME.pluginDirPath .. "/tests/fixtures/releases.json"`，读取后关闭文件。示例目录包含完整的 fixture 和错误响应测试。

## 配置测试环境

`test.load({ ... })` 支持以下选项：

| 选项 | 行为 |
| --- | --- |
| `os`、`arch` | 覆盖 `RUNTIME.osType`/`archType` 和 `OS_TYPE`/`ARCH_TYPE`，默认使用宿主平台。这些值不会模拟另一个操作系统。 |
| `env` | 覆盖该 VM 的 `os.getenv`。键为字符串，值为字符串或表示不存在的 `false`；未指定的变量也不存在，不修改宿主环境。 |
| `http` | 函数接收 `{method, url, headers}`，返回 `{status_code, body, headers}` 或 `nil, error`；`body`、`headers` 可省略。 |

HTTP 函数负责全部内置 GET、HEAD 和下载请求，包括插件加载期间发出的请求。它收到构造后的请求头；插件初始化完成后的请求包含 User-Agent。请求头名称使用 HTTP 标准大小写，例如 `User-Agent`。响应继续经过正式 HTTP 转换以及真实 HTML/JSON 解析。函数内的断言失败会使调用失败；返回 `nil, error` 用于模拟传输错误。

未配置 HTTP 函数时，测试默认拒绝请求，显式使用 `--online` 才会联网。配置函数后，即使传入 `--online` 也不会回退到真实网络。请求计数、条件响应和读取 fixture 都使用普通 Lua。

支持 `main.lua` 和 `metadata.lua` 加 `hooks/` 两种布局。保留正式的模块搜索顺序，以及插件脚本加载后注入运行时变量的顺序；测试选项会在加载前安装，使顶层 `require` 获取到配置后的模块。

## 调试单个 hook

```shell
vfox plugin run . PreInstall --input '{"version":"latest"}' --json
vfox plugin run . EnvKeys --input-file tests/env-keys.json --offline --json
vfox plugin run . PreInstall --input '{"version":"1.2.3"}' --os windows --arch amd64
vfox plugin run . ParseLegacyFile --input '{"filename":".example-version","filepath":"tests/fixtures/version","strategy":"latest_installed","installedVersions":["1.2.3"]}' --json
```

如果 shell 的 JSON 引号规则不同，优先使用 `--input-file`。`--input` 与 `--input-file` 互斥，省略输入时使用 `{}`。

hook 名称区分大小写：`Available`、`PreInstall`、`EnvKeys`、`PostInstall`、`PreUse`、`ParseLegacyFile`、`PreUninstall`。输入沿用正式 hook 上下文，所需路径、已安装 SDK 等信息由调用方提供。`ParseLegacyFile` 接收 `installedVersions` 数组，由适配层生成正式的 `getInstalledVersions()` 回调。命令不会自动安装、切换版本或查询已安装 SDK；调用未实现的可选 hook 会报错。

`run` 默认联网并保留真实环境，`--offline` 拒绝 HTTP 请求。`--json` 的 stdout 只包含强类型结果；无返回值或未提供结果时输出 `null`，协议以外的字段由正式 codec 丢弃。Lua 打印、标准输出写入、下载进度和 `os.execute` 输出进入 stderr；错误也写 stderr 并非零退出。不使用 `--json` 时，结果带 hook 名称并缩进显示。

## 执行边界与 CI

`--timeout` 必须为正值。默认每个测试文件 30 秒、每次 `run` 60 秒，覆盖 Lua 和内置 HTTP，包括读取响应及下载；不保证终止任意外部进程或其他阻塞的原生操作。

开发入口将 `os.exit` 转为 Lua 错误。测试默认禁用 `os.execute`、`io.popen`；需要时可在加载前用普通 Lua 替换：

```lua
local commands = {}
os.execute = function(command)
    commands[#commands + 1] = command
    return 0
end
local plugin = require("vfox.test").load()
plugin:PostInstall({ rootPath = "fixture-dir", sdkInfo = {} })
assert(#commands == 1)
```

这段 stub 适用于 `PostInstall` 会调用一次外部命令的插件。`run` 允许执行外部命令，用于调试可信 hook。

这些命令不提供文件系统沙箱。插件代码、fixture IO、下载及 hook 仍可向提供的路径写文件。`os`/`arch` 模拟只改变运行时变量；安装脚本和真实 SDK 冒烟测试应在目标操作系统的一次性 CI runner 或 VM 中执行。仅设置 `VFOX_HOME` 不能隔离所有常规 vfox 操作。

在 vfox 仓库中，无需下载 SDK 即可运行示例：

```shell
go run . plugin test examples/plugins/sample
go run . plugin run examples/plugins/sample PreInstall --input '{"version":"1.2.3"}' --os linux --arch amd64 --offline --json
```

core 测试套件会运行示例与 CLI 回归测试，仓库现有 CI 在 Linux、macOS、Windows 执行该套件。插件仓库接入属于后续工作：包含新命令的 vfox 版本发布后，普通 CI 步骤执行 `vfox plugin test .` 即可。
