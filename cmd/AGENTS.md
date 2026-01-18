# cmd/

## OVERVIEW
CLI layer implementing 22 commands using urfave/cli v3 framework.

## STRUCTURE
```
cmd/
├── cmd.go          # urfave/cli app setup, flags, version handling
└── commands/       # 22 command files (one per command)
```

## WHERE TO LOOK
| Location | Purpose |
|----------|---------|
| `cmd/cmd.go` | App initialization, 18 registered commands, global flags (debug, version) |
| `commands/base.go` | Category constants: `CategorySDK`, `CategoryPlugin` |
| `commands/activate.go` | Hidden command for shell integration (`Hidden: true`) |
| `commands/install.go` | SDK installation with `--all`, `--yes` flags |
| `commands/add.go` | Plugin management with `--source`, `--alias` flags |
| `commands/use.go` | Version switching with scope flags (`--global`, `--project`, `--session`, `--unlink`) |

## CONVENTIONS

### Command Pattern
Each command file exports a `*cli.Command` variable (e.g., `var Install = &cli.Command{...}`) with:
- `Name`: Lowercase command name
- `Usage`: Short description
- `Category`: `CategorySDK` or `CategoryPlugin` (from base.go)
- `Action`: Command handler function
- `Flags`: Optional CLI flags (alias common: `--all/-a`, `--yes/-y`)

### Category Constants
```go
const CategorySDK = "SDK"       // Commands: install, use, uninstall, current, list, available, search, exec, env
const CategoryPlugin = "Plugin" // Commands: add, remove, update, upgrade
```

### Hidden Command
`activate` is hidden (`Hidden: true`) - only called via shell hooks, never in `vfox --help`. Generates shell-specific environment initialization scripts with concurrent SDK processing (errgroup pattern).

### SDK Argument Format
Commands parse `sdk@version` format (e.g., `nodejs@21.5.0`). Version prefix `v` is automatically stripped (e.g., `v20.0.0` → `20.0.0`). Special tag `@latest` resolves to first available version.

### Manager Lifecycle
```go
manager, err := internal.NewSdkManager()
defer manager.Close()  // ALWAYS close to release resources
```

## ANTI-PATTERNS

1. **Direct SDK operations**: Commands must delegate to Manager layer, never call plugin hooks directly
2. **Missing Category**: Every command must have `Category` field set (CategorySDK or CategoryPlugin)
3. **Non-hidden activate**: Never remove `Hidden: true` from activate command
4. **Bypassing Manager**: Never instantiate SDK objects directly - use `manager.LookupSdk()` or `manager.LookupSdkWithInstall()`
5. **No defer Close()**: Forgetting `defer manager.Close()` causes resource leaks
