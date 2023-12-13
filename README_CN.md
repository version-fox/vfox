<p style="" align="center">
  <img src="./logo.png" alt="Logo" width="250" height="250">
</p>
<h1 style="margin-top: -40px">VersionFox</h1>

[![Go Report Card](https://goreportcard.com/badge/github.com/aooohan/version-fox)](https://goreportcard.com/report/github.com/aooohan/version-fox)
[![Go Reference](https://pkg.go.dev/badge/github.com/aooohan/version-fox.svg)](https://pkg.go.dev/github.com/aooohan/version-fox)
[![GitHub](https://img.shields.io/github/license/aooohan/version-fox)]()

## 介绍

`vf` 是一个用于管理SDK版本的工具，它允许你通过命令行快速安装和切换不同版本的SDK, 并且SDK是以Lua脚本形式作为插件进行提供,
也就是说这允许你实现自己的SDK来源, 或者使用别人分享的插件来安装SDK. 这都取决于你的想象力. ;)

### 安装

TODO

### 使用

### 插件

仓库...

### Examples
```bash

```bash
$ vf install node@20.10.0
Install node@20.10.0 success!

$ vf install node@18.10.0
Install node@18.10.0 success!

$ vf use node@20.10.0
Now using node@20.10.0

$ node -v
v20.10.0

$ vf ls node (installed)
-> 20.10.0 (current)
-> 18.10.0

$ vf use node@18.10.0
Now using node@18.10.0

$ node -v
v18.10.0

$ vf uninstall node@20.10.0
Uninstall node@20.10.0 success!

$ vf ls node
-> 18.10.0 (current)
```

## TODO

## Supported SDK Plugins

- [x] Node.js https://nodejs.org/dist/
- [ ] Python
- [ ] Go
- [ ] Java
- [ ] Rust
- [ ] Ruby
  // etc...

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any
contributions you make are greatly appreciated.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a pull request

## License

Distributed under the Apache 2.0 License. See [LICENSE](./LICENSE) for more information.

