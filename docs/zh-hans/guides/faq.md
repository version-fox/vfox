# 常见问题

## 如何卸载vfox?

请参考[卸载指南](./uninstallation.md)了解如何从系统中完全删除vfox的详细说明。

## 切换xxx不生效？ `vfox use`命令不生效？

如果你看到提示`"vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'`
则说明你没有将`vfox`正确挂在到你的`Shell`上。

请按照[快速入门#_2-挂载vfox到你的shell](./quick-start.md#_2-挂载vfox到你的shell)步骤进行手动挂载。


## GitBash下`use`和`search`命令无法进行选择?

相关ISSUE: [GitBash下无法进行选择](https://github.com/version-fox/vfox/issues/98)

仅限直接打开原生GitBash运行`vfox`会出现无法进行选择的问题, 临时解决办法:
1. 通过`Windows Terminal`运行`GitBash`
2. 在`VS Code`中运行`GitBash`
