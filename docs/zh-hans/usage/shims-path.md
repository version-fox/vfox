# 垫片 & PATH <Badge type="tip" text=">= 0.5.0" vertical="middle" />

`vfox` 是通过直接操作`PATH`来进行版本管理的, 但是有些IDE并不会读取`PATH`环境变量,
所以我们需要一些额外的操作来让IDE读取到`vfox`的版本。

## Shims 目录

该目录用于存放所有全局SDK垫片文件 。

**位置**: `$HOME/.version-fox/shims`

```shell
$ vfox use -g nodejs@14.17.0
$ ~/.version-fox/shims/node -v
v14.17.0
```

::: warning 注意

`vfox` 只会处理插件指定目录下的所有二进制文件, 如果你通过其他安装工具(`npm`)安装二进制文件, `shims`目录下是不会包含的。

以`nodejs`为例:

```shell
$ vfox use -g nodejs@14.17.0
$ npm install -g prettier@3.1.0
$ ~/.version-fox/shims/node -v
v14.17.0
$ ~/.version-fox/shims/prettier -v # 文件不存在!!!!
```

> 并不打算提供重建`shim`的能力。 请使用`current`软链。

:::

::: tip 垫片实现

- **Windows**: `.exe` 和 `.shim` 文件
- **Unix**: 软链接

以`nodejs`为例:

- **Windows**: `node.exe` 和 `node.shim`
- **Unix**: `.version-fox/shims/node` -> `.version-fox/cache/nodejs/v-14.17.0/nodejs-14.17.0/bin/node`
  :::

## `current` 软链接

`vfox` 除了会将全局SDK垫片放置在`shims`目录下, 还会在`$HOME/.version-fox/cache/<sdk>/`目录下创建一个软链接`current`,
指向对应的SDK。

位置: `$HOME/.version-fox/cache/<sdk>/current`

以`Nodejs`为例:

```shell
$ vfox use -g nodejs@14.17.0
$ npm install -g prettier@3.1.0
$ ~/.version-fox/cache/nodejs/current/node -v
v14.17.0
$ ~/.version-fox/cache/nodejs/current/prettier -v  # 可以了!!!
3.1.0
```

::: tip

`vfox`对于版本管理的核心也是通过`current`软链接来实现的.

当你在命令行中进行切换时, 实际上会创建一个软链接指向对应的SDK版本, 并将`current`软链接存放在当前`Shell`的临时目录下,
以及配置到`PATH`中, 从而实现版本的切换。

:::

