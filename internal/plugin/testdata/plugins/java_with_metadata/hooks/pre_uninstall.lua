


function PLUGIN:PreUninstall(ctx)
    printTable(ctx)
    local mainSdkInfo = ctx.main
    local mpath = mainSdkInfo.path
    local mversion = mainSdkInfo.version
    local mname = mainSdkInfo.name
    local sdkInfo = ctx.sdkInfo['sdk-name']
    local path = sdkInfo.path
    local version = sdkInfo.version
    local name = sdkInfo.name
end