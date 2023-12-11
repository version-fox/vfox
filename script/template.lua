---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...

PLUGIN = {
    name = "java",
    author = "Lihan",
    version = "0.0.1",
    -- github 或 http TODO 升级功能
    updateUrl = "https://raw.githubusercontent.com/aooohan/ktorm-generator/main/build.gradle.lua",
}

--- Return to target version download link
--- @param ctx table
--- @field ctx.version string version
--- @field ctx.os_type string os type
--- @field ctx.arch_type string arch type
--- @return string download url
function PLUGIN:DownloadUrl(ctx)
    -- TODO 从官网获取最新版本
    -- TODO 也可以是本地已经装了的???
    return ""
end

--- TODO 可能会出现不一致的版本情况 !! 也就是version怎么解析的问题

--- Returns the available download versions for the target context
---
--- @param ctx table {version1,version2...}
function PLUGIN:Search(ctx)
    return search(ctx)
end

--- Return the need to set environment variables when use this version
--- @param ctx table {version, version_path}
--- @return
--- {key = "JAVA_HOME", value = "xxxxxx"}
function PLUGIN:EnvKeys(ctx)
    return {
        {
            key = "JAVA_HOME",
            value = ctx.version_path .. "/bin"
        },
        {
            key = "PATH",
            value = version_path .. "/bin"
        }
    }
end
