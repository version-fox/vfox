nodeDownloadUrl = "https://nodejs.org/dist/v%s/node-v%s-%s-%s%s"

function download_url(ctx)
    os_type = ctx.os_type
    arch_type = ctx.arch_type
    version = ctx.version

    if arch_type == "amd64" then
        arch_type = "x64"
    end
    return string.format(nodeDownloadUrl, version, version, os_type, arch_type, file_ext(ctx))
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

function name()
    return {
        "node",
        "npm",
        "npx"
    }
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