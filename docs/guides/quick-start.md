# Quick Start

Here we take `Nodejs` as an example to introduce how to use `vfox`.

## 1. Installation

### Windows

::: details Scoop
```shell
scoop install vfox
```
:::

::: details winget
```shell
winget install vfox
```
:::

::: details Setup Installer
Please go to the [Releases](https://github.com/version-fox/vfox/releases) page to download the latest version of
the `setup` installer, and then follow the
installation wizard to install.
:::

::: details Manual Installation

1. Download the latest version of the `zip` installer from [Releases](https://github.com/version-fox/vfox/releases)
2. Configure the `PATH` environment variable to add the `vfox` installation directory to the `PATH` environment
   variable.
   :::

### Unix-like

::: details Homebrew

```shell
$ brew install vfox
```

:::

::: details APT

```shell
 echo "deb [trusted=yes] https://apt.fury.io/versionfox/ /" | sudo tee /etc/apt/sources.list.d/versionfox.list
 sudo apt-get update
 sudo apt-get install vfox
```

:::

::: details YUM

```shell
echo '[vfox]
name=VersionFox Repo
baseurl=https://yum.fury.io/versionfox/
enabled=1
gpgcheck=0' | sudo tee /etc/yum.repos.d/versionfox.repo

sudo yum install vfox
```

:::

::: details Manual Installation

```shell
$ curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

:::

## 2. Hook `vfox` to your `Shell`

::: warning
Please select a command suitable for your Shell from below to execute!
:::

::: details Bash

```shell
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
```

:::

::: details ZSH

```shell
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
```

:::

::: details Fish

```shell
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish
```

:::

::: details PowerShell

```PowerShell
if (-not (Test-Path -Path $PROFILE)) { New-Item -Type File -Path $PROFILE -Force }; Add-Content -Path $PROFILE -Value 'Invoke-Expression "$(vfox activate pwsh)"'
```

If PowerShell prompts: `cannot be loaded because the execution of scripts is disabled on this system`.**Open PowerShell** with **Run as Administrator**.Then, run this command in PowerShell

```shell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
# After that type Y and press Enter.
y
```

:::

::: details Clink & Cmder

1. Find the scripts path:
   ```shell
   clink info | findstr scripts
   ```
2. Copy [clink_vfox.lua](https://github.com/version-fox/vfox/blob/main/internal/shell/clink_vfox.lua) to the scripts directory
3. Restart Clink or Cmder
:::

::: details Nushell

```shell
vfox activate nushell $nu.default-config-dir | save --append $nu.config-path
```

:::

Afterward, open a new terminal.

## 3. Add a plugin

**Command**: `vfox add <plugin-name>`

After you have installed [vfox](https://github.com/version-fox/vfox), you still can't do anything. **You need to install the corresponding plugin first**.

::: tip
If you don't know which plugin to add, you can use the `vfox available` command to see all available plugins.
:::

```bash 
$ vfox add nodejs
```

## 4. Install a runtime

After the plugin is successfully installed, you can install the corresponding version of Nodejs.

**Command**: `vfox install nodejs@<version>`

We only install the latest available `latest` version:

```
$ vfox install nodejs@latest
```

Of course, we can also install a specific version:

```bash
$ vfox install nodejs@21.5.0
```

::: warning
`vfox` forces the use of an exact version. `latest` is a behavior that is parsed to the actual version number at
runtime, depending on the plugin's implementation.

If you **don't know the specific version **, you can use `vfox search nodejs` to see all available versions.
:::

::: tip 
`install` and `search` commands will check if the plugin is already added locally. If not, they will **automatically
add the plugin**.
:::


## 5. Switch runtime

**Command**: `vfox use [-p -g -s] nodejs[@<version>]`

`vfox` supports three scopes, each with a different range of effects:

### Global

**It takes effect globally**

```shell
$ vfox use -g nodejs
```

::: tip

`Global` is managed in the `$HOME/.version-fox/.tool-versions` file. 

The contents of the `.tool-versions` file as follows:

```text
nodejs 21.5.0
```

:::

::: danger Does not take effect after execution?
Please check if there is a runtime installed **previously** through other means in the `$PATH`!

For **Windows** users:

1. Please ensure that the system environment variable `Path` does not contain the runtime installed **previously**
   through
   other means!

2. `vfox` will automatically add the installed runtime to the **user environment variable** `Path`.

3. If there is a runtime installed **previously** through other means in your `Path`, please remove it manually!
 :::


### Project

**Different versions for different projects**

```shell
$ vfox use -p nodejs
```

`vfox` will **automatically detect whether there is a `.tool-versions` file** in the directory when you enter a directory. 
If it exists, `vfox` will **automatically switch to the version specified by the project**.

::: tip
`Project` is managed in the `$PWD/.tool-versions` file (current working directory). 
::: 

::: warning Default scope

If you do not specify a scope, `vfox` will use the default scope. Different systems have different scopes:

For **Windows**: The default scope is `Global`

For **Unix-like**: The default scope is `Session`
:::

### Session

**Different versions for different Shells**

```shell
$ vfox use -s nodejs
```

The session scope takes effect only for the current shell session. In other words, the versions are not shared between.

The main purpose of this scope is to meet **temporary needs**. 
When you close the current terminal, `vfox` will **automatically switch back to the `Global`/`Project` version**.


::: tip

`Session` is managed in the `$HOME/.version-fox/tmp/<shell-pid>/.tool-versions` file (temporary directory). 

:::



## Demo

::: tip
Sometimes, text expressions are far less intuitive than pictures, so let's go directly to the effect picture!
:::

![nodejs](/demo-full.gif)

## Guide Complete!

That completes the Getting Started guide for `vfox`🎉 You can now manage `nodejs` versions for your project. Follow
similar
steps for each type of tool in your project!

`vfox` has many more commands to become familiar with, you can see them all by running `vfox --help` or `vfox`. 

