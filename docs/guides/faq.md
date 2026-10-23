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

## `vfox use` says it switched, but `node` / `java` / … is still the old one

Something in front of `vfox` in `PATH` provides a command of the same name, so the shell never gets to `vfox`'s. After
`vfox use`, `vfox` says which executable that is. `vfox doctor <sdk>` shows the same in detail.

The usual causes are a `node` installed by `pnpm runtime set node -g` (it lands in `$PNPM_HOME/bin`) and a startup file
that adds a directory to `PATH` after `eval "$(vfox activate bash)"`: `vfox` puts its directories before what was in
`PATH` when it was activated, but what is added afterwards stays in front of them. Remove the competing copy, move the
`vfox activate` line below the line that adds the directory, or run `vfox config path.mode aggressive`, which puts
`vfox`'s directories before those as well.

A system JDK or a Homebrew `node` that the OS puts before `vfox`'s directories does not win by default. If `vfox doctor`
still reports one, the shell was started before `vfox` was upgraded (open a new terminal), `path.mode` is `legacy`
(`vfox config path.mode baseline` restores the default), or the directory is listed in `path.preserve`. See
[PATH Order Settings](./configuration.md#path-order-settings).

## How do I use a JDK (or any SDK) that is already installed on my machine?

Link it. Nothing is downloaded or copied, and removing the version from `vfox` later leaves the SDK alone:

```shell
vfox link java@17-system /usr/lib/jvm/java-17-openjdk-amd64
vfox use java@17-system
```

On a CI runner, `vfox link --detect java -y` links the JDKs the plugin can find. See [`vfox link`](../usage/core-commands.md#link).

## An SDK changed after `vfox` installed it (`flutter upgrade`, …)

`vfox` manages versions by directory, so it cannot stop an SDK from updating itself in place. It notices, though:
`vfox use` and `vfox doctor` warn when the plugin reports that an installed version is no longer what was installed.
Put it back with `vfox install --reinstall <sdk>@<version>`, and move to a newer version by installing that version
with `vfox` instead.

## Why can't I select when use `use` and `search` commands in GitBash?

Related ISSUE: [Unable to select in GitBash](https://github.com/version-fox/vfox/issues/98)

The problem of not being able to select in GitBash only occurs when you directly open the native GitBash to run `vfox`. A temporary solution is:
1. Run `GitBash` through `Windows Terminal`
2. Run `GitBash` in `VS Code`
