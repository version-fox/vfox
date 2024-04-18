# 插件相关命令

插件是`vfox`知道如何处理`Node.js`、`Java`、`Elixir`等不同工具的方式。

请参阅[创建插件](../plugins/create/howto.md)了解用于的支持更多工具的插件API。

## Available

列举[索引仓库](https://github.com/version-fox/vfox-plugins)中所有可用的插件。

**用法**
```shell
vfox available
```

## Add

添加插件,支持安装[仓库插件](../plugins/available.md)和自定义插件。

**用法**

```shell
vfox add [options] <plugin-name>
```
`plugin-name`: 插件名称， 如`nodejs`。

**选项**
- `-a, --alias`: 设置插件别名。
- `-s, --source`: 安装指定路径下的插件（可以是远程文件也可以是本地文件）


**例子**

**安装仓库插件**
```shell
$ vfox add --alias node nodejs
```

**安装自定义插件**
```shell
$ vfox add --source  https://github.com/version-fox/vfox-nodejs/releases/download/v0.0.5/vfox-nodejs_0.0.5.zip custom-node
```

## Info

查看指定插件信息

**用法**

```shell
vfox info <plugin-name>
```

## Remove

删除本地安装的插件。

**用法**

```shell
vfox remove <plugin-name>
```

::: danger 注意
删除插件，`vfox`会同步删除当前插件安装的所有版本运行时。
:::



## Update

更新指定的或全部已安装插件版本。

**用法**

```shell
vfox update <plugin-name>
vfox update --all # 更新所有已安装插件
```

