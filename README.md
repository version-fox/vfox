# VersionFox(vf)

[![Go Report Card](https://goreportcard.com/badge/github.com/aooohan/version-fox)](https://goreportcard.com/report/github.com/aooohan/version-fox)
[![Go Reference](https://pkg.go.dev/badge/github.com/aooohan/version-fox.svg)](https://pkg.go.dev/github.com/aooohan/version-fox)
[![GitHub](https://img.shields.io/github/license/aooohan/version-fox)]()

## Intro

`vf` is a tool for sdk version management, which allows you to quickly install and use different versions of targeted
sdk via the command line.

### Examples

plugin == sdk 
```bash
$ vf add <name> <plugin-url>
$ vf remove <name> (will remove plugins and installed sdk)
$ vf install <name>@<version>
$ vf uninstall <name>@<version>
$ vf use <name>@<version>
$ vf ls [<name>] (list installed sdk)



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

## Supported SDKs

- [x] Node.js https://nodejs.org/dist/
- [ ] Python
- [ ] Go
- [ ] Java
- [ ] Rust
- [ ] Ruby
  // etc...

## IDEA  !!!

- [ ] --local --global

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any
contributions you make are greatly appreciated.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a pull request

## License

Distributed under the Apache 2.0 License. See `LICENSE` for more information.

## Storage structure

-> .version-fox
-> env.sh
-> plugin
-> [plugin-name].lua
-> [plugin2-name].lua
-> .cache
-> [sdk1-name]
-> [v-version]
-> [sdk1-name]
-> [v-version]

## Arch

-> SdkManager
-> Handler
-> Source

## TODO



