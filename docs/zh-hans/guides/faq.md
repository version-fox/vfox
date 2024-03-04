# 常见问题

## 切换xxx不生效？ `vfox use`命令不生效？

如果你看到提示`Warning: The current shell lacks hook support or configuration. It has switched to global scope automatically`
则说明你没有将`vfox`正确挂在到你的`Shell`上。

请按照[快速入门#_2-挂载vfox到你的shell](./quick-start.md#_2-挂载vfox到你的shell)步骤进行手动挂载。

## 为什么我删除不了插件?

```text
我先是 add 了 java/adoptium-jdk ，然后尝试安装 v21 ，因为下载慢就中途退出了，然后尝试 remove 命令去掉这个 plugin ，得到错误信息 "java/adoptium-jdk not installed"。

那么我想换另一个源，执行 "vfox add java/azul-jdk" 时，也得到错误信息 "plugin java already exists"，现在是进退不能了。
```

在`vfox`理念中, 插件即SDK、SDK即插件. 你可以将插件理解为`vfox`的一种扩展, 用于管理不同的工具和运行环境。

以`nodejs/npmmirror`插件为例, `nodejs`是分类, `npmmirror`是插件名, 插件内部`name`字段标注的叫**SDK名**。

所以, 在删除插件时, 需要使用**SDK名**(这里就是`nodejs`)进行删除, 而不是插件名`nodejs/npmirror`或`npmmirror`。

```bash
$ vfox remove nodejs
```

在删除之前, 你可以使用`vfox ls`查看当前已安装的插件(即SDK名称), 然后再进行删除。

## Windows下PATH环境变量值重复?

只有一种情况下会出现这种情况, 就是你全局(`vfox use -g`)使用过SDK, 这个时候`vfox`会操作注册表,将SDK的`PATH`写入用户环境变量当中(为的是,
**不支持Hook功能**的Shell也能使用SDK, 例如`CMD`)。

但是因为`.tool-versions`机制的存在, 所以`PATH`就变成了`.tool-verions` + 用户环境变量`PATH`两部分组成。

::: warning
同一个SDK**最多重复两条**, 不会无限重复。如果>2次, 请反馈给我们。
::: 