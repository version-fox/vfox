# PROJECT KNOWLEDGE BASE

**Generated:** 2026-01-18
**Project:** vfox (Version Fox) - Cross-platform SDK version manager

## OVERVIEW
vfox is a cross-platform SDK version manager (similar to nvm, fvm, sdkman, asdf-vm) that uses Lua-based plugins to manage runtime versions across Global, Project, and Session scopes. Built with Go 1.24.0.

## COMMANDS
```bash
# Build
go build .

# Test (all)
go test ./...

# Test (single package)
go test ./internal/sdk -v

# Test with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic

# E2E tests
./scripts/e2e-test.sh  # Unix
pwsh ./scripts/e2e-test.ps1  # Windows

# Dependencies
go mod tidy
go get .

# Version bump
./scripts/bump.sh <version>

# Release (via goreleaser)
goreleaser release
```

## CONVENTIONS

### Code Style
- **License headers:** All Go files start with Apache 2.0 copyright block (17 lines)
- **Formatting:** Standard `gofmt` - use `go fmt ./` before committing
- **No linter config:** No golangci-lint, go vet, or golint configuration found
- **Import grouping:** Third-party packages grouped, internal packages last
- **Error wrapping:** Use `fmt.Errorf("context: %w", err)` for error chains

### Package Dependency Hierarchy (STRICT)
```
manager (orchestration)
  ↓
sdk (SDK abstraction)
  ↓
plugin (Lua plugin system)
  ↓
env (environment variables & shell integration)
  ↓
pathmeta, config (metadata & configuration)
  ↓
shared/* (utilities: logger, util, cache, shim)
```
- ✅ Downward dependencies only
- ❌ No upward dependencies
- ❌ No peer dependencies
- ✅ `shared/*` can import nothing from `internal/`

### SDK Operations (CRITICAL)
**FORBIDDEN:** Manager layer directly handling SDK operations
```go
// WRONG - Manager handling SDK ops directly
manager.Install(sdk, version)
manager.CreateSymlinks(...)

// CORRECT - Delegate to SDK layer
sdk, err := manager.LookupSdk("nodejs")
sdk.Install(version)
sdk.CreateSymlinksForScope(version, env.Project)
sdk.EnvKeysForScope(version, env.Project)
```

### Environment Variable Merging
- **Paths:** Appended in order (first = HIGHEST priority): Project → Session → Global
- **Vars:** Overwritten (last = HIGHEST priority): Global → Session → Project

### Configuration Format
**Simple:** `[tools] nodejs = "21.5.1"`
**Complex:** `[tools] java = { version = "21", vendor = "openjdk" }`
**Priority:** `.vfox.toml` > `vfox.toml` > `.tool-versions` > `.nvmrc` > `.node-version` > `.sdkmanrc`

### Path Architecture
- **UserPaths** (`~/.vfox` or `~/.version-fox`): tmp/, config.yaml, sdks/ (symlinks)
- **SharedPaths** (`VFOX_HOME` or UserPaths): cache/ (installs), plugins/, config.yaml
- **Environment variable:** `VFOX_HOME` sets shared root (default: same as UserPaths)
- **Internal vars:** `__VFOX_SHELL`, `__VFOX_PID`, `__VFOX_CURTMPPATH` (for shell hooks)

### Plugin System (Lua-based)
**Required hooks:** `Available()`, `PreInstall(version)`, `EnvKeys(version)`
**Optional hooks:** `PostInstall(version)`, `PreUse(version)`, `ParseLegacyFile()`, `PreUninstall(version)`
**Formats:** Single file (`main.lua`) or multi-file (`metadata.lua` + `hooks/*.lua`)

## ANTI-PATTERNS

### Architectural Violations
1. **Upward dependencies:** Lower layers importing higher layers
2. **Peer dependencies:** Same-level packages importing each other
3. **Manager SDK operations:** Direct SDK handling in manager (must delegate to SDK layer)
4. **Shared packages with business logic:** `internal/shared/*` should only contain utilities

### Code Quality
1. **No log.Fatal:** Use proper error handling
2. **Minimize unsafe:** Only for Windows-specific UAC elevation
3. **No circular imports:** Enforce strict layered hierarchy
4. **No empty catch blocks:** Handle errors properly

### Testing
1. **Write tests first:** TDD approach
2. **No deleting failing tests:** Fix code, not tests
3. **33 test files** cover: concurrent SDK lookup, paths, config, shells, plugins, cross-platform

## WHERE TO LOOK
| Task | Location |
|------|----------|
| **CLI commands** | `cmd/commands/` (22 commands) |
| **SDK operations** | `internal/sdk/sdk.go` |
| **Plugin system** | `internal/plugin/` |
| **Manager** | `internal/manager.go` (lookup, plugin registry only) |
| **Paths** | `internal/pathmeta/path_meta.go` |
| **Environment** | `internal/env/env.go` (scope-aware merging) |
| **Config** | `internal/config/config.go` |

## KEY DEPENDENCIES
- `github.com/urfave/cli/v3` - CLI framework
- `github.com/yuin/gopher-lua` - Lua VM
- `github.com/BurntSushi/toml` - TOML parsing
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/pterm/pterm` - Terminal UI
- `github.com/shirou/gopsutil/v4` - Process/system utilities

## CROSS-PLATFORM
- **Targets:** linux, darwin, windows on 386, amd64, arm, arm64, loong64
- **Shells:** Bash, Zsh, Fish, PowerShell, Clink, Nushell
- **Symlinks:** Unix (symbolic links), Windows (shim executables)
