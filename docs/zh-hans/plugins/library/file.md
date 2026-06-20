# File 标准库

`vfox` 为 Lua 插件提供基本的路径工具。使用 `require("file")` 访问这些接口。所有接口都同时支持文件和目录，但不提供文件内容读写能力。

## 复制、删除和移动路径

```lua
local file = require("file")

file.copy(source_path, destination_path)
file.move(source_path, destination_path)
file.remove(path)
```

## 创建符号链接

```lua
local file = require("file")
file.symlink(source_path, link_path)
```

路径可以是绝对路径，也可以是相对于当前 `vfox` 进程的路径。所有操作成功时均返回 `true`，失败时抛出 Lua 错误。复制和删除目录时会递归处理其内容。
