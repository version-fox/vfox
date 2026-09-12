# Test and debug a plugin

Use `vfox plugin test` to run ordinary Lua assertions against a local plugin, and `vfox plugin run` to inspect a single hook. Both use the core's plugin loader, built-in modules, and typed hook conversion. They do not register the plugin, install an SDK, or initialize user/project vfox configuration.

## Write a test

Add `tests/available_test.lua` to your plugin repository. The following test targets the [sample plugin](https://github.com/version-fox/vfox/tree/main/examples/plugins/sample) shipped with vfox:

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

Run it from the plugin repository:

```shell
vfox plugin test .
vfox plugin test . --file tests/available_test.lua
vfox plugin test . --online --timeout 45s
```

The runner discovers `tests/**/*_test.lua` in path order. Each file has fresh Lua globals, loaded modules, environment overrides, and HTTP state. Call `test.load()` once per file; its returned object invokes the real typed hook methods. Assertions stop the current file on the first uncaught failure; other files still run. The command reports `PASS`/`FAIL` and error locations, and exits nonzero if any file fails or no test files exist. A file that never calls `test.load()` fails.

`--file` is relative to the plugin directory, or may be an absolute file path. Commands retain the current working directory. For fixtures that must work from any directory, read `RUNTIME.pluginDirPath .. "/tests/fixtures/releases.json"` with ordinary `io.open`, and close the file after reading it. See the sample's complete fixture and error-response tests.

## Configure the test VM

`test.load({ ... })` accepts:

| Option | Behavior |
| --- | --- |
| `os`, `arch` | Override `RUNTIME.osType`/`archType` and `OS_TYPE`/`ARCH_TYPE`; default to the host platform. This does not emulate another operating system. |
| `env` | Values returned by this VM's `os.getenv`. Keys are strings; values are strings or `false` for an absent variable. Unspecified variables are absent. The host environment is unchanged. |
| `http` | A function receiving `{method, url, headers}` and returning `{status_code, body, headers}` or `nil, error`. `body` and `headers` are optional. |

The HTTP function handles every built-in GET, HEAD and download request, including requests made during plugin loading. It receives the constructed request headers, including User-Agent once plugin initialization has finished. Header names use HTTP canonical casing (for example `User-Agent`). A response still goes through the normal HTTP response conversion and real HTML/JSON parsing. Assertions inside the handler fail the calling operation; returning `nil, error` simulates a transport failure.

Without a handler, tests reject HTTP unless `--online` is set. With a handler, `--online` does not add a fallback to real requests. Use ordinary Lua conditionals, counters, closures and fixture files to model different responses; no mock DSL is needed.

Both `main.lua` plugins and `metadata.lua` plus `hooks/` plugins are supported. Loading retains the production module search order and initializes runtime globals after the plugin scripts. Test options are installed before loading so top-level imports capture the configured modules.

## Invoke a hook

```shell
vfox plugin run . PreInstall --input '{"version":"latest"}' --json
vfox plugin run . EnvKeys --input-file tests/env-keys.json --offline --json
vfox plugin run . PreInstall --input '{"version":"1.2.3"}' --os windows --arch amd64
vfox plugin run . ParseLegacyFile --input '{"filename":".example-version","filepath":"tests/fixtures/version","strategy":"latest_installed","installedVersions":["1.2.3"]}' --json
```

On shells with different JSON quoting rules, prefer `--input-file`. `--input` and `--input-file` are mutually exclusive; omitted input is `{}`.

Hook names are case-sensitive: `Available`, `PreInstall`, `EnvKeys`, `PostInstall`, `PreUse`, `ParseLegacyFile`, `PreUninstall`. Supply the normal hook context, including paths and installed SDK information when needed. For `ParseLegacyFile`, supply an `installedVersions` array; the adapter provides the hook's `getInstalledVersions()` callback. No installation, scope switching or SDK lookup happens automatically. Invoking an unimplemented optional hook is an error.

`run` allows real HTTP and retains the real environment by default; `--offline` rejects HTTP. `--json` writes only the typed result to stdout. Void hooks and hooks that provide no result produce `null`. Unknown result fields are discarded by the normal codec. Lua prints, standard output writes, download progress and `os.execute` output go to stderr. Errors go to stderr and cause a nonzero exit. Without `--json`, the result is indented and prefixed with the hook name.

## Execution boundaries and CI

`--timeout` must be positive. It defaults to 30 seconds per test file and 60 seconds per `run`, covering Lua execution and built-in HTTP, including response bodies and downloads. It does not guarantee termination of arbitrary external processes or other blocking native operations.

`os.exit` becomes a Lua error. Tests disable `os.execute` and `io.popen` by default; replace them with ordinary Lua stubs before loading the plugin when needed:

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

Use that stub with a plugin whose `PostInstall` invokes one external command. `run` permits external commands for debugging trusted hooks.

These commands are not a filesystem sandbox. Plugin code, fixture IO, downloads and hooks may write to supplied paths. A simulated `os`/`arch` changes runtime values only. Run installation scripts and real SDK smoke tests in disposable CI runners or VMs, using the actual target OS. Setting `VFOX_HOME` alone does not isolate every normal vfox operation.

From a vfox checkout, the bundled sample can be tested without downloading an SDK:

```shell
go run . plugin test examples/plugins/sample
go run . plugin run examples/plugins/sample PreInstall --input '{"version":"1.2.3"}' --os linux --arch amd64 --offline --json
```

The core test suite runs the example and CLI regression tests. The repository CI runs that suite on Linux, macOS and Windows. Plugin workflow adoption is a separate change: once a vfox release contains these commands, an ordinary CI step can run `vfox plugin test .`.
