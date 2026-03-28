# FAQ

## How do I uninstall vfox?

Please refer to the [Uninstallation Guide](./uninstallation.md) for detailed instructions on how to completely remove vfox from your system.

## Switch xxx not work or the vfox use  command does not work ?

If your shell prompt `vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'`
that means you do not hook `vfox` into your shell, please hook it manually first.

Please refer to [Quick Start#_2-hook-vfox-to-your-shell](./quick-start.md#_2-hook-vfox-to-your-shell) to manually hook `vfox` into your shell.

## How should I use vfox in Docker, CI/CD, or other non-interactive shells?

Prefer `vfox exec`.

`vfox activate` works by installing shell hooks, which are intended for interactive shells. In Docker build steps, CI jobs, and other non-interactive shells, these hooks are usually not triggered automatically.

Recommended examples:

```bash
vfox exec nodejs@24.14.0 -- npm install -g pnpm
vfox exec nodejs@24.14.0 -- bash -lc 'node -v && npm -v'
```

Use `vfox use` when you want to persist version selection for Global, Project, or Session scope. Use `vfox exec` when you want a command to run immediately with the correct SDK environment.

## Why can't I select when use `use` and `search` commands in GitBash?

Related ISSUE: [Unable to select in GitBash](https://github.com/version-fox/vfox/issues/98)

The problem of not being able to select in GitBash only occurs when you directly open the native GitBash to run `vfox`. A temporary solution is:
1. Run `GitBash` through `Windows Terminal`
2. Run `GitBash` in `VS Code`
