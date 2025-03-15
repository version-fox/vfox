# 如何将插件提交到索引仓库?

:::tip 提醒
请注意，这个文档是关于如何将插件提交到[索引仓库](https://github.com/version-fox/vfox-plugins)的。

如果你想要了解如何创建一个插件，请查看[这里](./howto.md)。
:::


## 索引仓库(Registry)

`vfox` 插件索引仓库是一个用于**收集和分发**各种**vfox插件**的仓库。 方便用户通过`vfox add <plugin-name>`命令来快速安装对应插件。

索引仓库是一个公共仓库, 任何人都可以提交**vfox插件**到这个仓库中。

索引仓库主要分为两个部分:
- `plugins`: 用于存放插件的`manifest.json`文件, 以插件短名为文件名。例如`nodejs.json`
- `sources`: 用于存放插件的数据源信息, 以插件短名为文件名。例如`nodejs.json`

仓库将会自动定时(间隔一小时)通过`sources`中的信息检索插件的最新版本信息以及校验插件可用性, 并将获取的`manifest`信息存放在`plugins`目录下。


::: tip 仓库地址
`vfox`默认将会从[插件仓库](https://version-fox.github.io/vfox-plugins)检索插件。

如果你想使用**自己的索引仓库或第三方镜像仓库**, 请按照[插件注册表地址](../../guides/configuration.md#插件注册表地址)进行配置。
:::

## 提交插件

1. 首先，你需要按照[插件创建指南](./howto.md)创建一个插件。
2. 维护一个`manifest.json`文件，用于描述插件最新版本信息。(如果基于[vfox-plugin-template](https://github.com/version-fox/vfox-plugin-template)开发, 在发版时会自动生成)
    ```json
    {
      "downloadUrl": "https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip",
      "notes": [],
      "version": "0.0.5",
      "homepage": "https://github.com/version-fox/vfox-nodejs",
      "minRuntimeVersion": "0.2.6",
      "license": "Apache 2.0",
      "description": "Node.js runtime environment.",
      "name": "nodejs"
    }
    ```
    - `downloadUrl`: 插件的下载地址
    - `notes`: 插件的更新日志
    - `version`: 插件的版本
    - `homepage`: 插件的主页
    - `minRuntimeVersion`: 插件所需的最低`vfox`版本
    - `license`: 插件的许可证
    - `description`: 插件的描述
    - `name`: 插件的短名
3. 在`sources/<name>.json`中创建一个带有您希望`vfox`使用的短名的文件, 例如`sources/nodejs.json`
4. 在`sources/<name>.json`中添加插件的manifest地址信息，例如:
    ```json
    {
      "name": "nodejs",
      "manifestUrl": "https://github.com/version-fox/vfox-nodejs/releases/download/manifest/manifest.json",
      "test": {
        "version": "21.7.1",
        "check": "node -v",
        "resultRegx": "v21.7.1"
      }
    }
    ```
    - `name`: 插件的短名
    - `manifestUrl`: 插件的manifest地址
    - `test`: 插件的测试信息
        - `version`: 插件的版本
        - `check`: 测试命令
        - `resultRegx`: 测试结果正则表达式
5. 最后, 提交一个PR到[索引仓库](https://github.com/version-fox/vfox-plugins/).
6. PR被合并后，插件将会被自动添加到索引仓库中, 并每隔一小时检查一次插件的更新情况。



