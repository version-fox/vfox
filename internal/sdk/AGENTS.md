# PROJECT KNOWLEDGE BASE - internal/sdk/

**Generated:** 2026-01-17
**Package:** internal/sdk - SDK lifecycle management layer

## OVERVIEW
Autonomous SDK lifecycle manager handling installation, version management, symlinks, and environment variables across all scopes.

## STRUCTURE
```
internal/sdk/
├── sdk.go           (1054 lines) - Main SDK implementation with all lifecycle operations
├── version.go       (109 lines)  - Version, Runtime, RuntimePackage types
├── env.go           (69 lines)   - SdkEnv, SdkEnvs for environment export
├── errors.go        (24 lines)   - ErrRuntimeNotFound error
├── metadata.go      (32 lines)   - Metadata struct (plugin + SDK paths)
├── attribute.go     (32 lines)   - UnLinkAttrFlag, IsUseUnLink helper
└── runtime.go       (20 lines)   - Empty
```

## WHERE TO LOOK
| Operation | Method | Location | Notes |
|-----------|--------|----------|-------|
| **Install SDK** | `Install(version)` | sdk.go:151 | Calls PreInstall hook, downloads, PostInstall hook |
| **Uninstall SDK** | `Uninstall(version)` | sdk.go:361 | Calls PreUninstall hook, removes symlinks, cleans dir |
| **Use version** | `Use(version, scope)` | sdk.go:525 | Validates hook env, resolves version, creates symlinks |
| **Create symlinks** | `CreateSymlinksForScope(version, scope)` | sdk.go:738 | Creates symlinks in scope-specific directory |
| **Get env keys** | `EnvKeysForScope(version, scope)` | sdk.go:748 | Returns env vars with paths pointing to symlinks |
| **List versions** | `Available(args)` | sdk.go:105 | With file-based caching |
| **Get current** | `Current()` | sdk.go:656 | Priority: Project > Session > Global |

## CONVENTIONS

### SDK Path Structure
- **Install path:** `{VFOX_ROOT}/installs/{sdk}/v-{version}/{name}-{version}`
- **Additions:** `{InstallPath}/add-{name}-{version}`
- **Symlink dirs:** Determined by scope via `envContext.GetLinkDirPathByScope()`

### Plugin Hook Flow
```
Install: PreInstall → Download → PostInstall
Uninstall: PreUninstall → Remove directory
Use: PreUse (version resolution) → CreateSymlinks → SaveConfig
```

### Symlink Creation
- Check if symlink exists and points to correct target before recreating
- Use `env.CreateDirSymlink()` which handles old symlink removal
- Create for both main runtime and all additions

### Environment Variable Handling
- `EnvKeys()` returns actual installed paths
- `EnvKeysForScope()` returns paths pointing to symlinks (NOT actual paths)
- Plugin hook `EnvKeys()` determines PATH and other environment variables

### Version Resolution
1. Exact match check via `CheckRuntimeExist()`
2. PreUse hook can modify version
3. Fuzzy match (prefix) if no exact match
4. Returns original input if no match found

## ANTI-PATTERNS

### Architectural Violations
1. **Upward dependencies:** SDK layer MUST NOT import `manager` package
2. **Direct plugin calls from outside:** Only SDK layer should interact with plugin wrappers
3. **Bypassing SDK layer:** Manager or cmd should NOT call plugin hooks directly

### Code Quality
1. **No symlink creation without scope validation:** Always verify scope directory exists
2. **No version resolution error handling gaps:** Always check `CheckRuntimeExist()` before using
3. **No cache invalidation:** Available hook cache respects duration config only
4. **No symlink state validation:** Before creating, check if correct symlink already exists

### Error Handling
1. **No silent failures:** All hook errors should be wrapped with context
2. **No incomplete cleanup:** Install failure must remove partial directories
3. **No ignoring PreUse errors:** Only continue if `IsNoResultProvided(err)`
