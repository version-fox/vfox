# vfox project guidance

vfox is a cross-platform SDK version manager with Lua plugins and Global, Project, and Session scopes. Use the Go version declared in [go.mod](go.mod).

This is the single project-wide guide. Keep behavior descriptions aligned with the implementation; use file paths and symbol names instead of fixed command counts, test counts, or source line numbers.

## Build and verification

Choose verification for the affected behavior. Run from the repository root:

```bash
# Build
go build .

# Focused tests; choose the affected package
go test ./internal/sdk -v

# Full suite for changes that affect multiple packages
go test ./...

# Coverage when needed
go test ./... -coverprofile=coverage.out -covermode=atomic
```

- For behavior changes, add or update tests that exercise the relevant behavior. Reproduce bugs with a failing regression test before fixing them when practical.
- Documentation and formatting changes do not require new tests. Check their diff and any affected examples or references.
- Do not remove or weaken assertions merely to make a failure pass. Update or remove obsolete tests when an intentional behavior change justifies it.
- Format changed Go files with `gofmt -w <files>`. Use `go fmt ./...` when formatting the whole module is intended; preserve unrelated working-tree changes.
- Use `go mod tidy` for dependency maintenance and `go get <module>@<version>` for intentional dependency changes. These commands can modify `go.mod` and `go.sum`.
- Consult [.github/workflows/ci.yml](.github/workflows/ci.yml) for CI checks and `go.mod` for the required Go version.

## E2E and release workflows

- [scripts/e2e-test.sh](scripts/e2e-test.sh) and [scripts/e2e-test.ps1](scripts/e2e-test.ps1) install SDKs, change vfox state, and clean installation or user directories. Run them only in a disposable environment such as a CI runner or VM. Setting `VFOX_HOME` alone does not isolate the Unix script from the real user's vfox directory; Windows global use also writes user environment settings to the registry.
- `./scripts/bump.sh <version>` changes the runtime version, stages it, commits, and creates a local tag. Use it for requested versioning work with the staging state reviewed; it does not push.
- `goreleaser release` runs the release workflow, including a GitHub draft release and configured distribution updates. Use it for requested release work. See [.goreleaser.yaml](.goreleaser.yaml) and [.github/workflows/go-releaser.yml](.github/workflows/go-releaser.yml).

## Responsibilities and imports

- Commands obtain SDKs through `Manager.LookupSdk` or `LookupSdkWithInstall`, then call SDK methods such as `Install`, `UseWithConfig`, and `Uninstall`.
- `Manager` lives in the `internal` package. It owns SDK lookup and lifetime, plugin discovery/add/update/remove, and orchestration. Plugin loading and validation belong to that management work.
- The SDK layer implements runtime installation, version resolution, scope links, and lifecycle hook calls. Keep these implementations out of Manager and commands.
- The plugin layer loads Lua code and implements hook invocation. Commands and Manager delegate SDK lifecycle hooks through SDK methods.
- Check `NewSdkManager` errors before deferring `manager.Close()`. Manager owns cached SDK lifetimes, SDK owns its plugin, and callers creating temporary plugins own their cleanup.

Allowed project-internal dependencies are listed below. `shared` includes the `internal/shared` package and its subpackages.

| Package or subtree | Allowed internal dependencies |
|---|---|
| `cmd/` | `internal` Manager, `sdk`, `shell`, `env`, `pathmeta`, `config`, `shared` |
| `internal` Manager | `sdk`, `plugin`, `env`, `pathmeta`, `config`, `shared` |
| `internal/sdk` | `plugin`, `shell`, `env`, `pathmeta`, `shared` |
| `internal/plugin/` | Packages within `plugin/`, `env`, `config`, `shared` |
| `internal/shell` | `env`, `shared` |
| `internal/env` | `pathmeta`, `config`, `shared` |
| `internal/pathmeta`, `internal/config` | `shared` |
| `internal/shared/` | Other packages within `shared/` |

Keep imports acyclic. Shared utilities must not depend on SDK, plugin, environment, configuration, or orchestration packages. Resolve dependency questions within the task's scope; keep unrelated refactoring separate.

## Code conventions

- Use the existing Apache 2.0 header style for new Go source files, without requiring a fixed line count. Preserve valid build-constraint placement.
- Group imports as standard library, third-party packages, then project packages in code being changed.
- Return errors from library code; do not introduce `log.Fatal` there. Preserve error chains with `fmt.Errorf("context: %w", err)`.
- Handle errors or explain intentional best-effort behavior, such as optional clipboard operations or cleanup.
- Keep `unsafe` limited to necessary platform API interop, such as Windows elevation and environment-change broadcasts.

## CLI behavior

- Register command definitions in [cmd/cmd.go](cmd/cmd.go). Helpers and platform-specific files may share a command implementation.
- Use `CategorySDK` or `CategoryPlugin` for commands belonging to those groups. Utility commands may be uncategorized.
- Keep `activate` hidden unless CLI visibility changes are part of the task. Users still invoke `vfox activate <shell>` during shell setup.
- Activation and environment output is evaluated by the shell; keep unrelated output out of generated shell code.
- SDK argument parsing is command-specific. Check the relevant parser before changing version prefixes, `@latest`, or `exec` arguments and its `--` separator.

## SDK and plugin behavior

- Installation calls `PreInstall`, prepares the main runtime and additions, then calls optional `PostInstall`. Preserve cleanup of partial directories created by a failed attempt. Uninstallation and its optional `PreUninstall` hook belong to SDK.
- Version resolution calls optional `PreUse` first. If no version is returned, use the existing exact-installed and prefix-matching logic. `IsNoResultProvided(err)` permits fallback; actual hook errors propagate. `UseWithConfig` checks that the resolved version is installed before applying scope state.
- `Current` checks the highest-priority configured version. Activation/export uses the separate installed-version fallback in [tool_resolution.go](cmd/commands/tool_resolution.go); preserve the caller-specific behavior.
- `EnvKeysForScope` passes scope link paths to the plugin and does not create links. Use the platform helpers in `env` to create/remove directory links for the main runtime and additions, preserving links that already point to the correct target.
- `Available(args)` caches by arguments in the plugin's `.available.cache`. Preserve `AvailableHookDuration` semantics: `0` disables caching and `-1` means no expiry.
- Required Lua hooks are `Available`, `PreInstall`, and `EnvKeys`; optional hooks are `PostInstall`, `PreUse`, `ParseLegacyFile`, and `PreUninstall`. Hooks receive context tables, such as `PLUGIN:PreInstall(ctx)` using `ctx.version`. Use [model.go](internal/plugin/model.go) for fields/results and the codec for structured conversion.
- Plugin loading prefers `main.lua`, with local modules in the plugin directory. Otherwise it loads `metadata.lua` and hook files, with `hooks/?.lua` searched before `lib/?.lua`. Preserve package-path restrictions and plugin-local module support.
- Preserve runtime-global initialization after plugin scripts load. Consult [module registration](internal/plugin/luai/module/module.go) for available built-ins, and both `HookFuncMap` and invocation code for hook names, case, and filenames.

## Configuration and environment behavior

Tool selections support simple versions and attributes:

```toml
[tools]
nodejs = "21.5.1"
java = { version = "21", vendor = "openjdk" }
```

- Within a directory, [pathmeta.LoadConfig](internal/pathmeta/config_loader.go) tries `.vfox.toml`, then `vfox.toml`, then `.tool-versions`. Reading `.tool-versions` attempts to write a migrated `.vfox.toml` while retaining the original.
- Other legacy files are handled through enabled plugins and their declared `LegacyFilenames` order. There is no core-wide ordering of `.nvmrc`, `.node-version`, and `.sdkmanrc`. Activation/export uses legacy results only for tools absent from the loaded project configuration.
- vfox settings live in `config.yaml`; their shared/user merge behavior is implemented by [LoadConfigWithFallback and Merge](internal/config/config.go).
- Config chains append from lowest to highest priority. Use `GetToolConfig`/`GetToolVersion` for the winning configuration, or `GetToolConfigsByPriority` for fallback candidates; check the found flag before using a returned scope.
- Within vfox-managed paths, priority is Project > Session > Global. Append paths in that order; merge ordinary variables in reverse order so Project wins.
- Final PATH assembly preserves user paths before the first existing vfox path, such as an activated virtualenv: preserved prefix > vfox-managed paths > remaining system paths. See [SplitSystemPaths](internal/env/context.go) and [environment export](cmd/commands/env.go).
- Preserve state-cache invalidation on configuration, project, and PATH changes. Use named scope constants; their numeric values do not define priority.

## Paths and platforms

- The user root uses an existing `~/.version-fox` directory, otherwise `~/.vfox`. It holds user configuration, temporary session state, and global SDK links.
- The shared root is `VFOX_HOME`, defaulting to the user root. Its default directories are `plugin/` and `cache/`, with shared settings in `config.yaml`.
- `storage.sdkPath` overrides the SDK installation root only. Preserve the legacy installation lookup in `sdk.NewSdk` when changing storage behavior.
- Global links use `<user-root>/sdks`; project links use `<project>/.vfox/sdks`; session links use `PathMeta.Working.SessionSdkDir`. Use [PathMeta](internal/pathmeta/path_meta.go) and [RuntimeEnvContext](internal/env/context.go) to resolve them.
- SDK directory links use Unix symlinks and Windows junctions. The separate `shared/shim` utility is not the SDK directory-link implementation.
- Supported release OS/architecture combinations are defined in [.goreleaser.yaml](.goreleaser.yaml). Shell implementations live in [internal/shell/](internal/shell/).
- Shell integration variables are defined in [env/flag.go](internal/env/flag.go) and [pathmeta/path_meta.go](internal/pathmeta/path_meta.go); preserve their contracts when changing hooks.

## Shared utilities

- Keep filesystem, network, clipboard, and process effects explicit in utility contracts; keep SDK lifecycle and scope policy in their owning packages.
- Review synchronization when changing shared state. A mutex in one cache instance does not establish safety between instances or processes using the same file. Logger level changes are not synchronized.
- `internal/shared/checksum.go` belongs to the existing `shared` package. File moves and package redesign require a task that calls for that refactoring.

## Where to look

| Area | Entry points |
|---|---|
| CLI registration and commands | [cmd/cmd.go](cmd/cmd.go), [commands/](cmd/commands/) |
| SDK lifecycle and runtime types | [sdk.go](internal/sdk/sdk.go), [runtime.go](internal/sdk/runtime.go) |
| Plugin loading and hooks | [plugin.go](internal/plugin/plugin.go), [lua_plugin.go](internal/plugin/lua_plugin.go), [model.go](internal/plugin/model.go) |
| Manager and registry | [manager.go](internal/manager.go), [manager_registry.go](internal/manager_registry.go) |
| Paths and tool configuration | [pathmeta/](internal/pathmeta/) |
| Environment and scope handling | [context.go](internal/env/context.go), [env.go](internal/env/env.go), [vfox_toml_chain.go](internal/env/vfox_toml_chain.go) |
| Shell initialization and exports | [shell/](internal/shell/) |
| vfox settings | [config/](internal/config/) |
| Shared utilities | [shared/](internal/shared/) |
| Windows MSIX packaging | [packaging/msix/](packaging/msix/) (`AppxManifest.xml`, `make-msix.ps1`, `gen-assets.ps1`; docs at [msix.md](docs/guides/msix.md); release via [compile-msix.yml](.github/workflows/compile-msix.yml), e2e via [e2e-msix-test.ps1](scripts/e2e-msix-test.ps1)) |
