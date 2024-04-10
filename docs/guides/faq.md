# FAQ

## Switch xxx not work or the vfox use  command does not work ?

If your shell prompt `Warning: The current shell lacks hook support or configuration. It has switched to global scope
automatically` that means you do not hook `vfox` into your shell, please hook it manually first.

Please refer to [Quick Start#_2-hook-vfox-to-your-shell](./quick-start.md#_2-hook-vfox-to-your-shell) to manually hook `vfox` into your shell.

## Why does the PATH environment variable value repeat on Windows?

Only one situation will cause this, that is, you have used the SDK globally (`vfox use -g`), at this time, `vfox` will
operate the registry and write the SDK's `PATH` into the user environment variable (for the purpose of, **Shell that does not
support Hook function** can also use SDK, such as `CMD`).

But because of the existence of the `.tool-versions` mechanism, the `PATH` becomes the sum of `.tool-verions` and the user
environment variable `PATH`.

::: warning
The same SDK **can be repeated at most twice**, it will not be repeated indefinitely. If >2 times, please feedback to us.
:::

## Why can't I select when use `use` and `search` commands in GitBash?

Related ISSUE: [Unable to select in GitBash](https://github.com/version-fox/vfox/issues/98)

The problem of not being able to select in GitBash only occurs when you directly open the native GitBash to run `vfox`. A temporary solution is:
1. Run `GitBash` through `Windows Terminal`
2. Run `GitBash` in `VS Code`
