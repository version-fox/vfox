# Multi-User Sharing

In server environments, you may want multiple users to share the same runtime SDK to save disk space and simplify management.

This guide explains how to configure vfox to enable multi-user SDK sharing.

## How It Works

vfox uses a **separation of user configuration and SDK installation** design, which is ideal for multi-user environments:

- **Shared directory** (`$VFOX_HOME`): Stores all SDK files and plugin definitions, shared by all users
- **User directory** (`~/.vfox`): Stores each user's personal configuration and version selections

This way:
- ✅ SDK files only need to be installed once, shared by all users, saving disk space
- ✅ Each user can independently choose SDK versions and customize their configuration
- ✅ Administrators can manage shared configuration centrally while users can flexibly override it

## Setting Up Shared SDK

### 1. Create a Shared Directory

First, create a shared directory that all users can access:

<Tabs>
<TabItem label="Linux/macOS">

```bash
# Create shared directory
sudo mkdir -p /opt/vfox

# Use group permissions (recommended - more secure)
sudo groupadd vfox
sudo chgrp vfox /opt/vfox
sudo chmod 2775 /opt/vfox
# Add user to vfox group
sudo usermod -a -G vfox username
```

**Security Notes on Permissions**:

While you could use `sudo chmod 1777 /opt/vfox` to allow all users read/write access, this has security risks:
- `777` permissions expose the directory to all users, including unauthorized ones
- Any user can delete or modify other users' SDK files (even with sticky bit)

**Recommended approach**: Use group permissions (as shown above)
- ✅ Only members of the vfox group can access it
- ✅ New files automatically inherit group permissions (setgid bit)
- ✅ Follows the principle of least privilege

</TabItem>
<TabItem label="Windows">

```powershell
# Create shared directory (can be placed anywhere, e.g., D:\vfox, E:\shared\vfox, etc.)
$vfoxPath = "D:\vfox"  # Modify to your desired path
New-Item -ItemType Directory -Path $vfoxPath -Force

# Set full permissions for all users
$acl = Get-Acl $vfoxPath
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "Users",
    "FullControl",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)
$acl.SetAccessRule($rule)
Set-Acl $vfoxPath $acl
```

</TabItem>
</Tabs>

### 2. Configure Each User

Each user who wants to use vfox needs to set the `VFOX_HOME` environment variable:

<Tabs>
<TabItem label="Bash">

```bash
# Add to ~/.bashrc
mkdir -p /opt/vfox
echo 'export VFOX_HOME=/opt/vfox' >> ~/.bashrc
source ~/.bashrc
```

</TabItem>
<TabItem label="ZSH">

```bash
# Add to ~/.zshrc
mkdir -p /opt/vfox
echo 'export VFOX_HOME=/opt/vfox' >> ~/.zshrc
source ~/.zshrc
```

</TabItem>
<TabItem label="Fish">

```shell
# Add to ~/.config/fish/config.fish
mkdir -p /opt/vfox
echo 'set -x VFOX_HOME /opt/vfox' >> ~/.config/fish/config.fish
source ~/.config/fish/config.fish
```

</TabItem>
<TabItem label="PowerShell">

```powershell
# Replace the path below with your shared directory path
[System.Environment]::SetEnvironmentVariable('VFOX_HOME', 'D:\vfox', 'User')
```

</TabItem>
</Tabs>

### 3. Install and Configure vfox

Now each user can use vfox normally:

```bash
# Add plugin (SDK will be installed to shared directory)
vfox add java

# Install SDK
vfox install java@21

# Set personal version choice (saved in each user's home directory)
vfox use -g java@21
```

### 4. (Optional) Configure Administrator Defaults

#### Configuration File Hierarchy

vfox 1.0.0+ supports **configuration file hierarchy**, allowing administrators to set default configuration in the shared directory, which users can flexibly override:

```
Configuration priority (highest to lowest):
1. User config (~/.vfox/config.yaml)     - User personalization (optional)
2. Shared config ($VFOX_HOME/config.yaml)  - Administrator/company defaults (optional)
3. Built-in defaults                       - vfox preset configuration
```

#### Create Shared Configuration

Administrators can create a `config.yaml` file in the shared directory to set company-level defaults:

```bash
# Create shared configuration file
sudo tee /opt/vfox/config.yaml > /dev/null <<EOF
# Company-level vfox configuration

proxy:
  enable: true
  url: http://proxy.company.com:8080

registry:
  address: https://npm.company.com/registry

cache:
  availableHookDuration: 24h
EOF

# Set permissions: administrator writable, others read-only
sudo chmod 644 /opt/vfox/config.yaml
```

#### User Configuration Options

Users can choose from the following three options based on their needs:

| Option | Description | Use Case |
|--------|-------------|----------|
| **Inherit company config** | Don't create `~/.vfox/config.yaml` | Most users with no special requirements |
| **Partial override** | Create `~/.vfox/config.yaml` with only needed items | Some users with specific proxy or registry needs |
| **Full customization** | Create `~/.vfox/config.yaml` with all items | Few users needing complete customization |

::: tip 💡 Configuration Merge Rules
Non-default values in user configuration will override shared configuration, and unset items will inherit from shared configuration. This ensures company-wide management while giving users sufficient customization flexibility.
:::

## Architecture Overview

Directory structure after setting `VFOX_HOME=/opt/vfox`:

```
/opt/vfox/                          # Shared directory (shared by all users)
├── config.yaml                     # Shared config (set by administrator, higher priority)
├── cache/                          # Actual SDK installation location
│   ├── java/
│   │   └── v-21.0.0/
│   │       └── java-21.0.0/        # JDK actual files
│   └── nodejs/
│       └── v-20.9.0/
│           └── nodejs-20.9.0/      # Node.js actual files
└── plugins/                        # Plugin definitions
    ├── java/
    └── nodejs/

~/.vfox/                            # User directory (independent for each user)
├── config.yaml                     # User personal config (optional, overrides shared config)
├── .vfox.toml                      # User's version selection
├── sdks/                           # User's symlinks
│   ├── java -> /opt/vfox/cache/java/v-21.0.0/java-21.0.0
│   └── nodejs -> /opt/vfox/cache/nodejs/v-20.9.0/nodejs-20.9.0
└── tmp/                            # User's temporary files
```

## Permission Management

### First User Installation

When the first user installs an SDK:

```bash
vfox install java@21
```

The SDK will be installed to `/opt/vfox/cache/java/v-21.0.0/`, and other users can use it directly without reinstalling.

## Migrating Existing Installation

If you're already using vfox and want to migrate to shared mode:

::: info 💡 Note for Pre-1.0.0 Users
Before vfox 1.0.0, the user directory was named `.version-fox`, not `.vfox`. If you're using an older version, replace `~/.vfox` with `~/.version-fox` in the commands below.
:::

<Tabs>
<TabItem label="Linux/macOS">

```bash
# 1. Create shared directory
sudo mkdir -p /opt/vfox
sudo groupadd vfox
sudo chgrp vfox /opt/vfox
sudo chmod 2775 /opt/vfox

# 2. Move existing SDK installations and plugins
mkdir -p /opt/vfox/cache /opt/vfox/plugins
mv ~/.vfox/cache/* /opt/vfox/cache/
mv ~/.vfox/plugins/* /opt/vfox/plugins/

# 3. Set VFOX_HOME
export VFOX_HOME=/opt/vfox

# 4. Add to shell configuration
echo 'export VFOX_HOME=/opt/vfox' >> ~/.bashrc
```

</TabItem>
<TabItem label="Windows">

```powershell
# 1. Create shared directory and set permissions (can be placed anywhere)
$vfoxPath = "D:\vfox"  # Modify to your desired path
New-Item -ItemType Directory -Path $vfoxPath -Force
$acl = Get-Acl $vfoxPath
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "Users",
    "FullControl",
    "ContainerInherit,ObjectInherit",
    "None",
    "Allow"
)
$acl.SetAccessRule($rule)
Set-Acl $vfoxPath $acl

# 2. Move existing SDK installations and plugins
New-Item -ItemType Directory -Path "$vfoxPath\cache\" -Force
New-Item -ItemType Directory -Path "$vfoxPath\plugins\" -Force
Move-Item -Path "$env:USERPROFILE\.vfox\cache\*" -Destination "$vfoxPath\cache\" -Force
Move-Item -Path "$env:USERPROFILE\.vfox\plugins\*" -Destination "$vfoxPath\plugins\" -Force

# 3. Set VFOX_HOME environment variable
[System.Environment]::SetEnvironmentVariable('VFOX_HOME', $vfoxPath, 'User')

# 4. Restart PowerShell or re-login to apply the environment variable
```

</TabItem>
</Tabs>

## Important Notes

1. **Permission issues**: After SDK installation, other users need at least read permissions to use it
2. **Plugin updates**: Plugin definitions are in the shared directory, updates affect all users
3. **Environment variables**: Make sure all users set the `VFOX_HOME` environment variable pointing to the same shared directory
4. **Old version migration**: Before vfox 1.0.0, the user directory was named `.version-fox`, be aware of this when migrating

