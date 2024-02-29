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

![performence.png](/performence.png)

上图是对两个工具最核心的功能进行基准测试, 会发现[vfox](https://github.com/version-fox/vfox)大约比`asdf-vm`快**5倍**!

`asdf-vm`的执行速度之所以较慢，主要是由于其`垫片`机制。简单来说，当你尝试运行如`node`这样的命令时，`asdf-vm`
会首先查找对应的垫片，然后根据`.tool-versions`文件或全局配置来确定使用哪个版本的`node`。这个查找和确定版本的过程会消耗一定的时间，从而影响了命令的执行速度。

相比之下，[vfox](https://github.com/version-fox/vfox)
采用了直接操作环境变量的方式来管理版本，它会直接设置和切换环境变量，从而避免了查找和确定版本的过程。因此，[vfox](https://github.com/version-fox/vfox)
在执行速度上要比使用`垫片`机制的`asdf-vm`快得多。

`asdf-vm`生态很强, 但是他对**Windows原生**无能为力, 虽然[vfox](https://github.com/version-fox/vfox)很新,
但是性能和平台兼容方面做的比`asdf-vm`更好。



### 插件换源

大多数时候, 我们会因为网络问题而困扰, 所以切换下载源的操作是必不可少的。

以Nodejs为例:
`asdf-vm`是通过`asdf-vm/asdf-nodejs`插件实现了对于Nodejs的支持, 而该插件是通过预定义一个环境变量来修改下载源, 如下所示:
```markdown
-   `NODEJS_ORG_MIRROR`: (Legacy) overrides the default mirror used for downloading the distibutions, alternative to the `NODE_BUILD_MIRROR_URL` node-build env var
```
这种方法的优点是相当灵活, 你可以切换任何镜像源。 但是*缺点*也很明显, 那就是*使用这个插件之前, 如果你不阅读这个插件的`README`, 你是不知道该怎么操作的*, 对用户不友好。

只涉及一种SDK的使用, 这种成本也可以接受。 那如果涉及多个呢? `Flutter`、`Python`等等**需要配置镜像的SDK**呢? 你是不是依然需要查看对应支持插件的README文档, 也算一种学习成本对不对!


而[vfox](https://github.com/version-fox/vfox)选择了另一种方式即*一个镜像源对应一个插件*, 如下所示:
```bash
$ vfox add nodejs/nodejs # 使用官方下载源
$ vfox add nodejs/npmmirror # 使用npmmirror镜像

$ vfox add python/python # 官方下载源
$ vfox add python/npmmirror 

$ vfox add flutter/flutter
$ vfox add flutter/flutter-cn
```

虽然仓库的插件多了, 但是用户使用起来心智负担低了, 也没有乱七八糟的环境变量需要配置, 对用户非常友好!
