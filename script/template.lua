---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...

OS_TYPE = ""
ARCH_TYPE = ""

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
--- @return string download url
function PLUGIN:DownloadUrl(ctx)
    return ""
end

--- Returns the available download versions for the target context
--- @param ctx table
--- @field ctx.version string version
--- @return table
---         version will as a input argument to DownloadUrl
---         notes  on target version, eg LTS, EOL etc.
function PLUGIN:Available(ctx)
    return {
        {
            version = "xxxx",
            note = "LTS"
        }
    }
end

--- Return the need to set environment variables when use this version
--- @param ctx table {version, version_path}
--- @return table Some variables must be set, it is recommended to set
--- the corresponding HOME environment variable, and use the corresponding
--- HOME environment variable in the PATH.
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
