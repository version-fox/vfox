# 常见问题

## 切换xxx不生效？ `vfox use`命令不生效？

如果你看到提示`Warning: The current shell lacks hook support or configuration. It has switched to global scope automatically`
则说明你没有将`vfox`正确挂在到你的`Shell`上。

请按照[快速入门#_2-挂载vfox到你的shell](./quick-start.md#_2-挂载vfox到你的shell)步骤进行手动挂载。


## Windows下PATH环境变量值重复?

只有一种情况下会出现这种情况, 就是你全局(`vfox use -g`)使用过SDK, 这个时候`vfox`会操作注册表,将SDK的`PATH`写入用户环境变量当中(为的是,
**不支持Hook功能**的Shell也能使用SDK, 例如`CMD`)。

但是因为`.tool-versions`机制的存在, 所以`PATH`就变成了`.tool-verions` + 用户环境变量`PATH`两部分组成。

::: warning
同一个SDK**最多重复两条**, 不会无限重复。如果>2次, 请反馈给我们。
::: 

## GitBash下`use`和`search`命令无法进行选择?

相关ISSUE: [GitBash下无法进行选择](https://github.com/version-fox/vfox/issues/98)

仅限直接打开原生GitBash运行`vfox`会出现无法进行选择的问题, 临时解决办法:
1. 通过`Windows Terminal`运行`GitBash`
2. 在`VS Code`中运行`GitBash`
