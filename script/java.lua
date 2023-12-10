local http = require("http")
DownloadUrl = "https://github.com/AdoptOpenJDK/openjdk11-binaries/releases/download/jdk-11.0.9.1%2B1/OpenJDK11U-jdk_x64_linux_hotspot_11.0.9.1_1.tar.gz"

url = "https://api.github.com/repos/AdoptOpenJDK/openjdk11-binaries/releases/latest"

function download_url(ctx)
    os_type = ctx.os_type
    arch_type = ctx.arch_type
    version = ctx.version
    http.get({
        url = url,
        headers = {
            ["Accept"] = "application/vnd.github.v3+json",
        },

    })

    if arch_type == "amd64" then
        arch_type = "x64"
    end
    return string.format(DownloadUrl, version, version, os_type, arch_type, file_ext(ctx))
end

function file_ext(ctx)
    os_type = ctx.os_type
    if os_type == "windows" then
        return ".zip"
    end
    return ".tar.gz"
end

function name()
    return "node"
end

function env_keys(ctx)
    version_path = ctx.version_path
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        }
    }
end