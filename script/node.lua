---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...

OS_TYPE = ""
ARCH_TYPE = ""

nodeDownloadUrl = "https://nodejs.org/dist/v%s/node-v%s-%s-%s%s"

--- https://nodejs.org/dist/index.json

PLUGIN = {
    name = "node",
    author = "Lihan",
    version = "0.0.1",
    updateUrl = "https://raw.githubusercontent.com/aooohan/ktorm-generator/main/build.gradle.lua",
}

--- Return to target version download link
--- @param ctx table
--- @field ctx.version string version
--- @return string download url
function PLUGIN:DownloadUrl(ctx)
    version = ctx.version

    arch_type = ARCH_TYPE
    ext = ".tar.gz"
    if arch_type == "amd64" then
        arch_type = "x64"
    end
    if OS_TYPE == "windows" then
        ext = ".zip"
    end
    return string.format(nodeDownloadUrl, version, version, OS_TYPE, arch_type, ext)
end

--- Returns the available download versions for the target context
--- @param ctx table
--- @field ctx.version string version
function PLUGIN:Search(ctx)
    return {}
end

--- Return the need to set environment variables when use this version
--- @param ctx table {version, version_path}
--- @return {key = "JAVA_HOME", value = "xxxxxx"}
function PLUGIN:EnvKeys(ctx)
    version_path = ctx.version_path
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        }
    }
end
