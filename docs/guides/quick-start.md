# Quick Start

Here we take `Nodejs` as an example to introduce how to use `vfox`.

## 1. Installation

### Windows

<Tabs>
<TabItem label="Scoop">

```shell
scoop install vfox
```

</TabItem>
<TabItem label="winget">

```shell
winget install vfox
```

</TabItem>
<TabItem label="Setup Installer">

Go to the [Releases](https://github.com/version-fox/vfox/releases) page to download the latest version of the `setup` installer, then follow the installation wizard to install.

</TabItem>
</Tabs>

### Unix-like

<Tabs>
<TabItem label="Homebrew">

```shell
brew install vfox
```

</TabItem>
<TabItem label="APT (Debian/Ubuntu)">

```shell
echo "deb [trusted=yes] https://apt.fury.io/versionfox/ /" | sudo tee /etc/apt/sources.list.d/versionfox.list
sudo apt-get update
sudo apt-get install vfox
```

</TabItem>
<TabItem label="YUM (CentOS/Fedora)">

```shell
echo '[vfox]
name=VersionFox Repo
baseurl=https://yum.fury.io/versionfox/
enabled=1
gpgcheck=0' | sudo tee /etc/yum.repos.d/versionfox.repo

sudo yum install vfox
```

</TabItem>
<TabItem label="Install Script">

```shell
curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash
```

**User-local Installation (no sudo required)**

If you want to install `vfox` to your user directory (`~/.local/bin`) instead of system-wide, you can use the `--user` flag. This is particularly useful for environments where you don't have sudo access:

```shell
curl -sSL https://raw.githubusercontent.com/version-fox/vfox/main/install.sh | bash -s -- --user
```

This will:

- Install `vfox` to `~/.local/bin` (no sudo required)
- Automatically create the directory if it doesn't exist
- Provide instructions to add `~/.local/bin` to your `PATH`

</TabItem>
</Tabs>

## 2. Hook vfox to your Shell

> [!WARNING] ⚠️ Warning
> Please select a command suitable for your Shell from below to execute!

<Tabs>
<TabItem label="Bash">

```shell
echo 'eval "$(vfox activate bash)"' >> ~/.bashrc
source ~/.bashrc
```

</TabItem>
<TabItem label="ZSH">

```shell
echo 'eval "$(vfox activate zsh)"' >> ~/.zshrc
```

</TabItem>
<TabItem label="Fish">

```shell
echo 'vfox activate fish | source' >> ~/.config/fish/config.fish
```

</TabItem>
<TabItem label="PowerShell">

Create PowerShell configuration:

```powershell
if (-not (Test-Path -Path $PROFILE)) { New-Item -Type File -Path $PROFILE -Force }; Add-Content -Path $PROFILE -Value 'Invoke-Expression "$(vfox activate pwsh)"'
```

If PowerShell prompts "cannot be loaded because the execution of scripts is disabled on this system", **run PowerShell as Administrator** and execute:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
```

Type `Y` and press Enter to confirm.

</TabItem>
<TabItem label="Clink & Cmder">

1. Find the scripts path:

    ```shell
    clink info | findstr scripts
    ```

2. Copy [clink_vfox.lua](https://github.com/version-fox/vfox/blob/main/internal/shell/clink_vfox.lua) to the scripts directory
3. Restart Clink or Cmder

</TabItem>
<TabItem label="Nushell">

```shell
vfox activate nushell $nu.default-config-dir | save --append $nu.config-path
```

</TabItem>
</Tabs>


## 3. Add a Plugin

**Command**: `vfox add <plugin-name>`

After installing vfox, you also need to install the corresponding plugin to manage SDK.

::: tip 💡 Tip
You can use the `vfox available` command to see all available plugins.
:::

```bash
vfox add nodejs
```

## 4. Install a Runtime

After the plugin is successfully installed, you can install the corresponding version of Node.js.

**Command**: `vfox install nodejs@<version>`

```bash
vfox install nodejs@21.5.0
```

::: warning ⚠️ About the Latest Version

`latest` is a special marker that takes the first version (usually the latest) from the list of available versions returned by the plugin. Both `install` and `use` commands support this marker, but **it is not recommended for production use**.

```bash
vfox install nodejs@latest
vfox use -g nodejs@latest
```

**Why not recommended?** `latest` always points to the current latest version, but new versions may introduce breaking changes or unstable features, which can easily cause compatibility issues in your project.

::: tip 💡 Recommended Approach
Always use exact version numbers to ensure project stability and reproducibility. You can use `vfox search nodejs` to query all available versions.
:::

::: tip 💡 Auto-install Plugin
`install` and `search` commands will automatically detect and install missing plugins.
:::


## 5. Switch Runtime

**Command**: `vfox use [-p -g -s] [--unlink] nodejs[@<version>]`

`vfox` supports three scopes, with version priority from high to low:

**Project > Session > Global > System**

### Scope Overview

| Scope | Command | SDK Path | Effective Range |
|-------|---------|----------|-----------------|
| **Project** | `vfox use -p` | `$PWD/.vfox/sdks` | Current project directory |
| **Session** | `vfox use -s` | `~/.vfox/tmp/<pid>/sdks` | Current Shell session |
| **Global** | `vfox use -g` | `~/.vfox/sdks` | Global |

::: info 📖 How It Works

vfox creates directory symlinks in different scopes pointing to actual SDK installation directories, and adds these paths to the `PATH` environment variable in priority order, enabling version switching.

**PATH Priority Example**:

```bash
# Project > Session > Global > System
$PWD/.vfox/sdks/nodejs/bin:~/.vfox/tmp/<pid>/nodejs/bin:~/.vfox/sdks/nodejs/bin:/usr/bin:...
```

:::

---

### Project (Project Scope)

::: tip 💡 Recommended
For project development, each project can have independent tool versions.
:::

**Usage**:

```bash
# Use nodejs in current project directory
vfox use -p nodejs@20.9.0
```

**After execution, vfox will**:

1. **Create directory symlinks**: Create a symlink in `$PWD/.vfox/sdks/nodejs` pointing to the actual installation directory
2. **Auto-add to .gitignore**: If a `.gitignore` file exists, vfox will automatically add the `.vfox/` directory to the ignore list to prevent committing to the repository
3. **Update PATH**: Insert `$PWD/.vfox/sdks/nodejs/bin` at the front of `PATH`
4. **Save configuration**: Write version information to `.vfox.toml` file

This way, when you execute `node` command in the project directory, Shell will find your project-level nodejs at the front of PATH, ensuring the version matches project requirements.

**Visual Example**:

```bash
# 1. Execute command
$ vfox use -p nodejs@20.9.0

# 2. View created symlink
$ ls -la .vfox/sdks/nodejs
lrwxr-xr-x  1 user  staff  nodejs -> /Users/user/.vfox/cache/nodejs/v-20.9.0/nodejs-20.9.0

# 3. View updated PATH
$ echo $PATH
/project/path/.vfox/sdks/nodejs/bin:/previous/paths:...
#                  ↑ Project-level nodejs at the front

# 4. View configuration file
$ cat .vfox.toml
[tools]
nodejs = "20.9.0"

# 5. Verify version (using project-level version)
$ node -v
v20.9.0
```

::: warning 💡 Highly Recommended
Commit `.vfox.toml` to your repository and add `.vfox` directory to `.gitignore`. This way team members can share version configuration.
:::

::: danger ⚠️ About the --unlink Parameter

If you don't want to create symlinks in the project directory, you can use the `--unlink` parameter:

```bash
vfox use -p --unlink nodejs@20.9.0
```

**Note**: After using `--unlink`, the Project scope downgrades to Session scope (configuration recorded in .vfox.toml but without creating symlinks). **We strongly recommend keeping the default behavior** (creating symlinks).
:::

---

### Session (Session Scope)

::: tip 💡 Temporary Testing
For temporarily testing a specific version, automatically expires when you close the current Shell window.
:::

**Usage**:

```bash
vfox use -s nodejs@18.0.0
```

When you close the Shell window, the temporary directory and configuration are cleaned up automatically, not affecting other Shell sessions.

---

### Global (Global Scope)

::: tip 💡 User-Level Default Version
For setting user-level default versions, available for all projects (unless overridden by Project or Session).
:::

**Usage**:

```bash
vfox use -g nodejs@21.5.0
```

## Demo

Text descriptions are not as intuitive as pictures. Let's see the demo directly!

![nodejs](/demo-full.gif)

## Quick Start Complete! 🎉

Congratulations on completing the `vfox` quick start! Now you can:

- ✅ Quickly install and switch between different versions of development tools
- ✅ Configure independent tool versions for your projects
- ✅ Temporarily test specific tool versions
- ✅ Share consistent development environment configuration with your team

**Next Steps**:

Use `vfox --help` to see more commands and options, or visit [All Commands](../usage/all-commands.md) to learn more features. 

