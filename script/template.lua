
---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...
---  VERSION: 1.0.0
------
PLUGIN = {
    name = "java",
    author = "Lihan",
    version = "0.0.1",
    -- github 或 http
    updateUrl = "https://raw.githubusercontent.com/aooohan/ktorm-generator/main/build.gradle.lua",
}

function PLUGIN:DownloadUrl(ctx)
    -- TODO 从官网获取最新版本
    -- TODO 也可以是本地已经装了的???
    return ""
end

function PLUGIN:Search(ctx)
    return search(ctx)
end

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
