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


## `vfox use` 说已经切换了，但 `node` / `java` / …… 还是旧的

`vfox` 之前的 `PATH` 里有同名的命令，shell 根本走不到 `vfox` 提供的那个。`vfox use` 之后，`vfox` 会告诉你那个可执行文件是哪个；`vfox doctor <sdk>` 会给出同样的详细信息。

常见原因：`pnpm runtime set node -g` 装的 `node`（放在 `$PNPM_HOME/bin`），以及在 `eval "$(vfox activate bash)"` 之后又往 `PATH` 加目录的启动文件：`vfox` 会把自己的目录放到激活时 `PATH` 里原有内容之前，但激活**之后**才加入的目录仍排在它前面。删掉冲突的那一份，把 `vfox activate` 那一行挪到添加目录的那一行之后，或者执行 `vfox config path.mode aggressive`，让 `vfox` 的目录也排到这些目录之前。

系统自带的 JDK，或操作系统排到 `vfox` 目录之前的 Homebrew `node`，默认不会再压过 `vfox`。如果 `vfox doctor` 仍然报告这类目录，可能是 shell 在升级 `vfox` 之前就已启动（重新打开终端）、`path.mode` 被设成了 `legacy`（执行 `vfox config path.mode baseline` 恢复默认值），或者该目录列在 `path.preserve` 里。详见 [PATH 顺序设置](./configuration.md#path-顺序设置)。

## 怎样使用本机上已经装好的 JDK（或其他 SDK）？

把它链接进来。不会下载也不会复制，之后从 `vfox` 里移除这个版本时 SDK 本身不受影响：

```shell
vfox link java@17-system /usr/lib/jvm/java-17-openjdk-amd64
vfox use java@17-system
```

在 CI runner 上，`vfox link --detect java -y` 会把插件能找到的 JDK 都链接进来。见 [`vfox link`](../usage/core-commands.md#link)。

## SDK 在 `vfox` 安装之后又被改动了（`flutter upgrade` 等）

`vfox` 按目录管理版本，没办法阻止 SDK 自己在原地更新。不过它能发现：当插件报告某个已安装的版本和安装时不一样了，`vfox use` 和 `vfox doctor` 会给出警告。用 `vfox install --reinstall <sdk>@<version>` 把它恢复原样；想换到更新的版本，请用 `vfox` 安装那个版本，而不是在原地更新。

## GitBash下`use`和`search`命令无法进行选择?

相关ISSUE: [GitBash下无法进行选择](https://github.com/version-fox/vfox/issues/98)

仅限直接打开原生GitBash运行`vfox`会出现无法进行选择的问题, 临时解决办法:
1. 通过`Windows Terminal`运行`GitBash`
2. 在`VS Code`中运行`GitBash`
