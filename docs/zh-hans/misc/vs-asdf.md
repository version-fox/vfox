# 比asdf-vm好在哪里?

`vfox` 与 `asdf-vm` 目标一致, 即一个工具管理所有的运行时版本, 且都是用`.tool-versions`文件来记录版本信息。

但是 `vfox` 有以下优势:

## 操作系统兼容性

| 操作系统兼容性    | Windows(非WSL) | Linux | macOS |
|------------|---------------|-------|-------|
| asdf-vm    | ❌             | ✅     | ✅     |
| VersionFox | ✅             | ✅     | ✅     |

`asdf-vm`是`Shell`实现的工具, 所以对于**原生Windows**环境并**不支持**!

而[vfox](https://github.com/version-fox/vfox)是`Golang` + `Lua`实现的, 因此天然支持`Windows`和其他操作系统。



## 性能对比

![performance.png](/performance.png)

上图是对两个工具最核心的功能进行基准测试, 会发现[vfox](https://github.com/version-fox/vfox)大约比`asdf-vm`快**5倍**!

`asdf-vm`的执行速度之所以较慢，主要是由于其`垫片`机制。简单来说，当你尝试运行如`node`这样的命令时，`asdf-vm`
会首先查找对应的垫片，然后根据`.tool-versions`文件或全局配置来确定使用哪个版本的`node`。这个查找和确定版本的过程会消耗一定的时间，从而影响了命令的执行速度。

相比之下，[vfox](https://github.com/version-fox/vfox)
采用了直接操作环境变量的方式来管理版本，它会直接设置和切换环境变量，从而避免了查找和确定版本的过程。因此，[vfox](https://github.com/version-fox/vfox)
在执行速度上要比使用`垫片`机制的`asdf-vm`快得多。

`asdf-vm`生态很强, 但是他对**Windows原生**无能为力, 虽然[vfox](https://github.com/version-fox/vfox)很新,
但是性能和平台兼容方面做的比`asdf-vm`更好。

