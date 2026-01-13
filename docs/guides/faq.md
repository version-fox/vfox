# FAQ

## How do I uninstall vfox?

Please refer to the [Uninstallation Guide](./uninstallation.md) for detailed instructions on how to completely remove vfox from your system.

## Switch xxx not work or the vfox use  command does not work ?

If your shell prompt `vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'`
that means you do not hook `vfox` into your shell, please hook it manually first.

Please refer to [Quick Start#_2-hook-vfox-to-your-shell](./quick-start.md#_2-hook-vfox-to-your-shell) to manually hook `vfox` into your shell.

## Why can't I select when use `use` and `search` commands in GitBash?

Related ISSUE: [Unable to select in GitBash](https://github.com/version-fox/vfox/issues/98)

The problem of not being able to select in GitBash only occurs when you directly open the native GitBash to run `vfox`. A temporary solution is:
1. Run `GitBash` through `Windows Terminal`
2. Run `GitBash` in `VS Code`
