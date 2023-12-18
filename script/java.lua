---  Default global variable
---  OS_TYPE:  windows, linux, darwin
---  ARCH_TYPE: 386, amd64, arm, arm64  ...

OS_TYPE = ""
ARCH_TYPE = ""

SearchUrl = "https://api.adoptium.net/v3/assets/latest/%s/hotspot?os=%s&architecture=%s"
AvailableVersionsUrl = "https://api.adoptium.net/v3/info/available_releases"



PLUGIN = {
    name = "adoptium-jdk",
    author = "Han Li",
    version = "0.0.1",
    updateUrl = "",
}

function PLUGIN:PreInstall(ctx)
end

function PLUGIN:PostInstall(ctx)
end

function PLUGIN:Available(ctx)
    return {}
end

function PLUGIN:EnvKeys(ctx)
    version_path = ctx.version_path
    return {
        {
            key = "PATH",
            value = version_path .. "/bin"
        }
    }
end
