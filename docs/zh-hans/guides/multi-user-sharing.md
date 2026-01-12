# 多用户共享

在服务器环境中，你可能希望多个用户共享同一套运行时SDK，以节省磁盘空间和简化管理。 

本指南介绍如何配置 vfox 以实现多用户共享 SDK。

## 工作原理

vfox 采用**用户配置与 SDK 安装分离**的设计，非常适合多用户环境：

- **共享目录**（`$VFOX_HOME`）：存放所有 SDK 文件和插件定义，被所有用户共享
- **用户目录**（`~/.vfox`）：存放每个用户的个人配置和版本选择

这样：
- ✅ SDK 文件只需安装一次，所有用户共享使用，节省磁盘空间
- ✅ 每个用户可以独立选择 SDK 版本和个人化配置
- ✅ 管理员可以统一管理共享配置，用户可以灵活覆盖

## 设置共享 SDK

### 1. 创建共享目录

首先，创建一个所有用户都能访问的共享目录：

<Tabs>
<TabItem label="Linux/macOS">

```bash
# 创建共享目录
sudo mkdir -p /opt/vfox

# 使用组权限（推荐 - 更安全）
sudo groupadd vfox
sudo chgrp vfox /opt/vfox
sudo chmod 2775 /opt/vfox
# 将用户添加到 vfox 组
sudo usermod -a -G vfox username
```

**关于权限的安全性说明**：

虽然也可以使用 `sudo chmod 1777 /opt/vfox` 让所有用户可读写，但这存在安全隐患：
- `777` 权限对所有用户完全开放，包括未授权用户
- 任何用户都可以删除或修改他人的 SDK 文件（即使有 sticky bit）

**推荐方案**：使用组权限（如上所示）
- ✅ 只有 vfox 组的成员才能访问
- ✅ 新文件自动继承组权限（setgid 位）
- ✅ 更符合最小权限原则

</TabItem>
<TabItem label="Windows">

```powershell
# 创建共享目录（可以放在任意位置，如 D:\vfox、E:\shared\vfox 等）
$vfoxPath = "D:\vfox"  # 修改为你想要的路径
New-Item -ItemType Directory -Path $vfoxPath -Force

# 为所有用户设置完全权限
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

### 2. 配置每个用户

每个需要使用 vfox 的用户都需要设置 `VFOX_HOME` 环境变量：

<Tabs>
<TabItem label="Bash">

```bash
# 添加到 ~/.bashrc
mkdir -p /opt/vfox
echo 'export VFOX_HOME=/opt/vfox' >> ~/.bashrc
source ~/.bashrc
```

</TabItem>
<TabItem label="ZSH">

```bash
# 添加到 ~/.zshrc
mkdir -p /opt/vfox
echo 'export VFOX_HOME=/opt/vfox' >> ~/.zshrc
source ~/.zshrc
```

</TabItem>
<TabItem label="Fish">

```shell
# 添加到 ~/.config/fish/config.fish
mkdir -p /opt/vfox
echo 'set -x VFOX_HOME /opt/vfox' >> ~/.config/fish/config.fish
source ~/.config/fish/config.fish
```

</TabItem>
<TabItem label="PowerShell">

```powershell
# 将下面的路径替换为你的共享目录路径
[System.Environment]::SetEnvironmentVariable('VFOX_HOME', 'D:\vfox', 'User')
```

</TabItem>
</Tabs>

### 3. 安装和配置 vfox

现在每个用户可以正常使用 vfox：

```bash
# 添加插件（SDK 会安装到共享目录）
vfox add java

# 安装 SDK
vfox install java@21

# 设置个人版本选择（保存在各自的家目录）
vfox use -g java@21
```

### 4. （可选）配置管理员默认值

#### 配置文件层级

vfox 1.0.0+ 支持**配置文件层级**，允许管理员在共享目录设置默认配置，用户可以灵活覆盖：

```
配置优先级（从高到低）：
1. 用户配置 (~/.vfox/config.yaml)     - 用户个性化设置（可选）
2. 共享配置 ($VFOX_HOME/config.yaml)  - 管理员设置的公司默认值（可选）
3. 内置默认值                          - vfox 预设配置
```

#### 创建共享配置

管理员可以在共享目录创建 `config.yaml` 文件设置公司级别的默认配置：

```bash
# 创建共享配置文件
sudo tee /opt/vfox/config.yaml > /dev/null <<EOF
# 公司级别的 vfox 配置

proxy:
  enable: true
  url: http://proxy.company.com:8080

registry:
  address: https://npm.company.com/registry

cache:
  availableHookDuration: 24h
EOF

# 设置权限：管理员可写，其他用户只读
sudo chmod 644 /opt/vfox/config.yaml
```

#### 用户配置方案

用户可以根据需要选择以下三种方案：

| 方案 | 描述 | 场景 |
|------|------|------|
| **继承公司配置** | 不创建 `~/.vfox/config.yaml` | 大多数用户，无特殊需求 |
| **部分覆盖** | 创建 `~/.vfox/config.yaml`，只配置需要的项 | 某些用户有特殊代理或仓库需求 |
| **完全自定义** | 创建 `~/.vfox/config.yaml`，设置所有项 | 少数用户需要完全自定义 |

::: tip 💡 配置合并规则
用户配置中的**非默认值**会覆盖共享配置，**未设置**的项会继承共享配置。这样既保证了公司统一管理，又给了用户充分的自定义空间。
:::

## 架构说明

设置 `VFOX_HOME=/opt/vfox` 后的目录结构：

```
/opt/vfox/                          # 共享目录（所有用户共享）
├── config.yaml                     # 共享配置（管理员设置，优先级高）
├── cache/                          # SDK 实际安装位置
│   ├── java/
│   │   └── v-21.0.0/
│   │       └── java-21.0.0/        # JDK 实际文件
│   └── nodejs/
│       └── v-20.9.0/
│           └── nodejs-20.9.0/      # Node.js 实际文件
└── plugins/                        # 插件定义
    ├── java/
    └── nodejs/

~/.vfox/                            # 用户目录（每个用户独立）
├── config.yaml                     # 用户个人配置（可选，覆盖共享配置）
├── .vfox.toml                      # 用户的版本选择
├── sdks/                           # 用户的符号链接
│   ├── java -> /opt/vfox/cache/java/v-21.0.0/java-21.0.0
│   └── nodejs -> /opt/vfox/cache/nodejs/v-20.9.0/nodejs-20.9.0
└── tmp/                            # 用户的临时文件
```


## 权限管理

### 用户首次安装

当第一个用户安装 SDK 时：

```bash
vfox install java@21
```

SDK 会被安装到 `/opt/vfox/cache/java/v-21.0.0/`，其他用户可以直接使用，无需重复安装。


## 迁移现有安装

如果你已经使用 vfox 并想迁移到共享模式：

::: info 💡 老版本用户注意
vfox 1.0.0 之前，用户目录名称是 `.version-fox`，而非 `.vfox`。如果你使用的是老版本，请将下面命令中的 `~/.vfox` 替换为 `~/.version-fox`。
:::

<Tabs>
<TabItem label="Linux/macOS">

```bash
# 1. 创建共享目录
sudo mkdir -p /opt/vfox
sudo groupadd vfox
sudo chgrp vfox /opt/vfox
sudo chmod 2775 /opt/vfox

# 2. 移动现有的 SDK 安装和插件
mkdir -p /opt/vfox/cache /opt/vfox/plugins
mv ~/.vfox/cache/* /opt/vfox/cache/
mv ~/.vfox/plugins/* /opt/vfox/plugins/

# 3. 设置 VFOX_HOME
export VFOX_HOME=/opt/vfox

# 4. 添加到 shell 配置
echo 'export VFOX_HOME=/opt/vfox' >> ~/.bashrc
```

</TabItem>
<TabItem label="Windows">

```powershell
# 1. 创建共享目录并设置权限（可以放在任意位置）
$vfoxPath = "D:\vfox"  # 修改为你想要的路径
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

# 2. 移动现有的 SDK 安装和插件
New-Item -ItemType Directory -Path "$vfoxPath\cache\" -Force
New-Item -ItemType Directory -Path "$vfoxPath\plugins\" -Force
Move-Item -Path "$env:USERPROFILE\.vfox\cache\*" -Destination "$vfoxPath\cache\" -Force
Move-Item -Path "$env:USERPROFILE\.vfox\plugins\*" -Destination "$vfoxPath\plugins\" -Force

# 3. 设置 VFOX_HOME 环境变量
[System.Environment]::SetEnvironmentVariable('VFOX_HOME', $vfoxPath, 'User')

# 4. 重启 PowerShell 或重新登录以使环境变量生效
```

</TabItem>
</Tabs>

## 注意事项

1. **权限问题**：SDK 安装后，其他用户需要至少有读权限才能使用
2. **插件更新**：插件定义在共享目录，更新会影响所有用户
3. **环境变量**：确保所有用户都设置了 `VFOX_HOME` 环境变量，指向同一个共享目录
4. **旧版本迁移**：vfox 1.0.0 之前用户目录名称是 `.version-fox`，迁移时需要注意

