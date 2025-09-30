# Uninstallation

This guide will help you completely remove `vfox` from your system.

## 1. Remove Shell Hooks

First, you need to remove the `vfox` activation commands from your shell configuration files.

::: warning
Please select the instructions appropriate for your Shell!
:::

::: details Bash

Open your `~/.bashrc` file and remove the following line:

```shell
eval "$(vfox activate bash)"
```

After saving the file, reload your shell configuration:

```shell
source ~/.bashrc
```

:::

::: details ZSH

Open your `~/.zshrc` file and remove the following line:

```shell
eval "$(vfox activate zsh)"
```

After saving the file, reload your shell configuration:

```shell
source ~/.zshrc
```

:::

::: details Fish

Open your `~/.config/fish/config.fish` file and remove the following line:

```shell
vfox activate fish | source
```

After saving the file, reload your shell configuration:

```shell
source ~/.config/fish/config.fish
```

:::

::: details PowerShell

Open your PowerShell profile. You can find its location by running:

```powershell
$PROFILE
```

Common locations:
- `C:\Users\<username>\Documents\PowerShell\Microsoft.PowerShell_profile.ps1` (PowerShell 7+)
- `C:\Users\<username>\Documents\WindowsPowerShell\Microsoft.PowerShell_profile.ps1` (Windows PowerShell)

Remove the following line from the profile:

```powershell
Invoke-Expression "$(vfox activate pwsh)"
```

After saving the file, reload your PowerShell profile:

```powershell
. $PROFILE
```

:::

::: details Clink & Cmder

1. Find the scripts path:
   ```shell
   clink info | findstr scripts
   ```
2. Remove the `clink_vfox.lua` file from the scripts directory
3. Restart Clink or Cmder

:::

::: details Nushell

Open your Nushell config file (location shown by `$nu.config-path`) and remove the vfox activation line that was appended during installation.

:::

## 2. Uninstall vfox Binary

Remove the `vfox` executable from your system.

### Windows

::: details Scoop

```shell
scoop uninstall vfox
```

:::

::: details winget

```shell
winget uninstall vfox
```

:::

::: details Setup Installer

1. Open **Settings** > **Apps** > **Installed apps** (or **Control Panel** > **Programs** > **Uninstall a program**)
2. Find **vfox** in the list
3. Click **Uninstall** and follow the wizard

:::

::: details Manual Installation

1. Delete the directory where you extracted vfox
2. Remove the `vfox` installation directory from your `PATH` environment variable:
   - Open **System Properties** > **Environment Variables**
   - Find `Path` in **User variables** or **System variables**
   - Remove the entry pointing to the vfox directory
   - Click **OK** to save

:::

### Unix-like

::: details Homebrew

```shell
brew uninstall vfox
```

:::

::: details APT

```shell
sudo apt-get remove vfox
```

To also remove the repository configuration:

```shell
sudo rm /etc/apt/sources.list.d/versionfox.list
```

:::

::: details YUM

```shell
sudo yum remove vfox
```

To also remove the repository configuration:

```shell
sudo rm /etc/yum.repos.d/versionfox.repo
```

:::

::: details Manual Installation

Remove the vfox binary:

```shell
sudo rm /usr/local/bin/vfox
```

:::

## 3. Clean Up vfox Data (Optional)

If you want to completely remove all data stored by `vfox`, including installed SDKs, plugins, and configuration files:

::: warning
This will permanently delete all SDK versions you installed through vfox!
:::

### Remove vfox Data Directory

```shell
rm -rf ~/.version-fox
```

This directory contains:
- Installed SDK versions
- Plugin files
- Configuration files
- Global `.tool-versions` file
- Cache and temporary files

## Verify Uninstallation

To verify that `vfox` has been completely removed:

```shell
which vfox
# or
vfox --version
```

Both commands should return "command not found" or similar error messages.

## Troubleshooting

### vfox command still works after uninstallation

- Make sure you have closed and reopened all terminal windows after removing the shell hooks
- Check if there are multiple shell configuration files (e.g., `.bash_profile`, `.profile`, `.bashrc`) and ensure the vfox activation line is removed from all of them
- On Windows, restart your computer to ensure all environment variable changes take effect

### SDKs installed via vfox still appear in PATH

- Check if you have a `.tool-versions` file in your current directory or home directory
- Remove the vfox data directory as described in step 3 above
- On Windows, manually check your user environment variables and remove any SDK-related paths that were added by vfox
