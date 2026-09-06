# FS 标准库

`vfox` 为 Lua 插件提供基本的路径工具。使用 `require("fs")` 访问这些接口。所有接口都同时支持文件和目录，但不提供文件内容读写能力。

## 复制、删除和移动路径

```lua
local fs = require("fs")

fs.copy(source_path, destination_path)
fs.move(source_path, destination_path)
fs.remove(path)
```

## 创建符号链接

```lua
local fs = require("fs")
fs.symlink(source_path, link_path)
```

路径可以是绝对路径，也可以是相对于当前 `vfox` 进程的路径。所有操作成功时均返回 `true`，失败时抛出 Lua 错误。复制和删除目录时会递归处理其内容。与 `mv` 一样，`move` 的目标是新路径时会执行重命名；目标是已有目录时，会将源路径按原 basename 移入该目录。

复制目录时会保留符号链接及其目标，不会沿链接递归复制。`symlink` 的源路径相对于当前进程目录解析，即使链接创建在子目录中也是如此。
