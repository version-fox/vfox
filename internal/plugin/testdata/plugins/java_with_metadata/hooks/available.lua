--- Return all available versions provided by this plugin
--- @param ctx table Empty table used as context, for future extension
--- @return table Descriptions of available versions and accompanying tool descriptions
function PLUGIN:Available(ctx)
    print("invoke Available Hook")
    return {
        {
            version = "xxxx",
            note = os.time(),
            addition = {
                {
                    name = "npm",
                    version = "8.8.8",
                }
            }
        }
    }
end