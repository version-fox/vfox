# Comparison with asdf-vm

`vfox` and `asdf-vm` have the same goal, that is, a tool to manage all runtime versions, and all versions are recorded
in the `.tool-versions` file.

But `vfox` has the following advantages:

## Platform compatibility

| Tool    | Windows (non-WSL) | Linux | macOS |
|---------|-------------------|-------|-------|
| asdf-vm | ❌                 | ✅     | ✅     |
| vfox    | ✅                 | ✅     | ✅     |

`asdf-vm` is a `Shell`-based tool, so it does not support **native Windows** environments!

`vfox` is implemented in `Golang` + `Lua`, so it naturally supports `Windows` and other operating systems.

## Performance comparison

![performance.png](/performance.png)

The above figure is a benchmark test of the two tools' most core functions. It will be found that `vfox`
is about **5 times** faster than `asdf-vm`!

The reason why `asdf-vm` is slower is mainly due to its `shim` mechanism. Simply put, when you try to run a command
like `node`, `asdf-vm` will first look for the corresponding `shim`, and then determine which version of `node` to use
based on the `.tool-versions` file or global configuration. This process of finding and determining the version will
consume
a certain amount of time, thereby affecting the speed of command execution.

In contrast, `vfox` uses the direct operation of environment variables to manage versions, and it will directly set and
switch
environment variables, thereby avoiding the process of finding and determining the version. Therefore, `vfox` is much
faster
than `asdf-vm` using the `shim` mechanism.

`asdf-vm` ecosystem is very strong, but it is powerless for **Windows native**. Although `vfox` is very new,
it does better in terms of performance and platform compatibility than `asdf-vm`.
