# PROJECT KNOWLEDGE BASE - internal/plugin/

**Generated:** 2026-01-17
**Commit:** v0.x
**Component:** Lua-based plugin system

## OVERVIEW
Lua VM integration for extensible SDK management with hook-based plugin architecture.

## STRUCTURE
```
internal/plugin/
├── luai/              # Lua VM integration
│   ├── codec/         # Go-Lua type marshaling (encode/decode)
│   └── module/        # Built-in Lua libraries (http, json, html, archiver, strings, file)
└── testdata/          # Test plugin fixtures
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| **Hook definitions** | `plugin.go` | Required/optional hooks, HookFuncMap |
| **Plugin interface** | `plugin.go` | Plugin interface with 8 methods |
| **Lua VM setup** | `luai/vm.go` | NewLuaVM(), Prepare(), LimitPackagePath() |
| **Plugin loading** | `lua_plugin.go` | CreateLuaPlugin(), loads main.lua or metadata.lua + hooks/ |
| **Hook execution** | `lua_plugin.go` | Available(), PreInstall(), EnvKeys(), etc. |
| **Codec** | `luai/codec/` | Marshal/Unmarshal for Go-Lua type conversion |
| **Plugin validation** | `wrapper.go` | validate() checks required hooks, name format |
| **Hook contexts** | `model.go` | HookCtx and HookResult types |

## CONVENTIONS

### Plugin Loading (Two Formats)
**Single file:** `main.lua` with plugin object and all hooks defined
**Multi-file:** `metadata.lua` + `hooks/*.lua` (each hook in separate file)

**File loading priority:** hooks/ > lib/ (for multi-file plugins)

### Hook Functionality
**Required hooks:** Available(), PreInstall(version), EnvKeys(version)
**Optional hooks:** PostInstall(version), PreUse(version), ParseLegacyFile(), PreUninstall(version)

**Hook calling:** Lua functions called with colon syntax (`pluginObj:method()`), pluginObj implicitly passed as first argument

### Type Marshaling
- Go structs ↔ Lua LTable via codec.Marshal()/codec.Unmarshal()
- Context structures (HookCtx) passed to Lua hooks
- Hook results returned as Lua tables, unmarshaled to Go structs

### Global Lua Variables
- `OS_TYPE`: Platform (darwin, linux, windows)
- `ARCH_TYPE`: Architecture (amd64, arm64, etc.)
- `Runtime`: RuntimeInfo with osType, archType, version, pluginDirPath
- `PLUGIN`: Plugin object with metadata and hook functions
- `navigator`: HTTP user agent for network requests

### Plugin Validation
- Name must match regex: `^[a-zA-Z][a-zA-Z0-9_\-]*$`
- Required hooks must be present via HasFunction() check
- Plugin object must be defined (global "PLUGIN")

## ANTI-PATTERNS

### Plugin Development
1. **Missing required hooks:** Plugin must define Available(), PreInstall(), EnvKeys()
2. **Invalid plugin name:** Must start with letter, use only alphanumeric/underscore/hyphen
3. **No result from required hooks:** Returning nil from Available/PreInstall/EnvKeys causes errors
4. **Bypassing codec:** Don't directly manipulate Lua state, use Marshal/Unmarshal for type safety

### VM Management
1. **Not closing VM:** Always call plugin.Close() to release Lua VM resources
2. **Unlimited package path:** Use LimitPackagePath() to restrict module search scope
3. **Setting globals before loading:** Set OS_TYPE, ARCH_TYPE, Runtime AFTER loading plugin scripts (prevents overwriting)

### Module System
1. **Importing outside module system:** Lua plugins should only use built-in modules (http, json, html, archiver, strings, file)
2. **Excessive network requests:** PreInstall should use available SDK info, not make redundant HTTP calls
