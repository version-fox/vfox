# Shims & PATH <Badge type="tip" text=">= 0.5.0" vertical="middle" />

`vfox` manage versions by manipulating `PATH` directly, but some IDEs don't read the `PATH` environment variable, so we
need some extra operations to let the IDE used.

## Shims Directory

This directory is used to store all shims of global SDK.

**Location**: `$HOME/.version-fox/shims`

```shell
$ vfox use -g nodejs@14.17.0
$ ~/.version-fox/shims/node -v
v14.17.0
```

::: warning

`vfox` only handles all binary files in the installation directory. If you install binary files through other
installation tools (`npm`), the `shims` directory will not contain them.

Take `nodejs` as an example:
```shell
$ vfox use -g nodejs@14.17.0
$ npm install -g prettier@3.1.0
$ ~/.version-fox/shims/node -v
v14.17.0
$ ~/.version-fox/shims/prettier -v # File not found!!!!
```

> Do not intend to provide the ability to rebuild `shim`. Please use the `current` soft link.

:::

::: tip Shim Implementation

- **Windows**: `.exe` and `.shim` files
- **Unix**: Soft link

Take `nodejs` as an example:
- **Windows**: `node.exe` and `node.shim`
- **Unix**: `.version-fox/shims/node` -> `.version-fox/cache/nodejs/v-14.17.0/nodejs-14.17.0/bin/node`
:::

## `current` Soft Link

`vfox` will create a soft link `current` in the `$HOME/.version-fox/cache/<sdk>/` directory, pointing to the corresponding SDK.

**Location**: `$HOME/.version-fox/cache/<sdk>/current`

Take `Nodejs` as an example:

```shell
$ vfox use -g nodejs@14.17.0
$ npm install -g prettier@3.1.0
$ ~/.version-fox/cache/nodejs/current/node -v
v14.17.0
$ ~/.version-fox/cache/nodejs/current/prettier -v  # Ok!!!
3.1.0
```

::: tip

`vfox`'s core for version management is also implemented through the `current` soft link.

When you switch on Shell, a soft link is actually created to point to the corresponding SDK version, and the `current`
soft link is stored in the temporary directory of the current `Shell`, as well as configured to `PATH`, to achieve version switching.

:::

