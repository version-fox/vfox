

local util = {}



function util:PreInstall(ctx)
    local version = ctx.version
    local runtimeVersion = ctx.runtimeVersion
    print(OS_TYPE)
    return {
        --- Version number
        version = "version",
        --- remote URL or local file path [optional]
        url = "xxx",
        --- SHA256 checksum [optional]
        sha256 = "xxx",
        --- md5 checksum [optional]
        md5 = "xxx",
        --- sha1 checksum [optional]
        sha1 = "xxx",
        --- sha512 checksum [optional]
        sha512 = "xx",
        --- additional need files [optional]
        addition = {
            {
                --- additional file name !
                name = "xxx",
                --- remote URL or local file path [optional]
                url = "xxx",
                --- SHA256 checksum [optional]
                sha256 = "xxx",
                --- md5 checksum [optional]
                md5 = "xxx",
                --- sha1 checksum [optional]
                sha1 = "xxx",
                --- sha512 checksum [optional]
                sha512 = "xx",
            }
        }
    }
end


return util