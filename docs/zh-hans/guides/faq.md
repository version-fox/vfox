# 常见问题

## 如何卸载vfox?

请参考[卸载指南](./uninstallation.md)了解如何从系统中完全删除vfox的详细说明。

## 切换xxx不生效？ `vfox use`命令不生效？

如果你看到提示`"vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'`
则说明你没有将`vfox`正确挂在到你的`Shell`上。

请按照[快速入门#_2-挂载vfox到你的shell](./quick-start.md#_2-挂载vfox到你的shell)步骤进行手动挂载。

## 在 Docker、CI/CD 或其他非交互 Shell 中应该怎么使用 vfox？

推荐优先使用 `vfox exec`。

`vfox activate` 的工作方式是安装 Shell Hook，这更适合交互式 Shell。在 Docker 构建步骤、CI Job 以及其他非交互 Shell 中，这些 Hook 通常不会自动触发。

推荐示例：

```bash
vfox exec nodejs@24.14.0 -- npm install -g pnpm
vfox exec nodejs@24.14.0 -- bash -lc 'node -v && npm -v'
```

如果你希望把版本选择持久化到 Global、Project 或 Session 作用域，请使用 `vfox use`。如果你希望某条命令立刻在正确的 SDK 环境中执行，请使用 `vfox exec`。


## GitBash下`use`和`search`命令无法进行选择?

相关ISSUE: [GitBash下无法进行选择](https://github.com/version-fox/vfox/issues/98)

仅限直接打开原生GitBash运行`vfox`会出现无法进行选择的问题, 临时解决办法:
1. 通过`Windows Terminal`运行`GitBash`
2. 在`VS Code`中运行`GitBash`
