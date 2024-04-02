local vfox_sdk_table = {}
local vfox_available = io.popen('vfox available')
for line in vfox_available:lines() do
    local sdk, yes = line:gsub('%c%[%d+m', ''):match('^(%a+)%s+(%u%u%u?)')
    if sdk and yes and (yes == 'YES' or yes == 'NO') then
        table.insert(vfox_sdk_table, sdk)
    end
end
vfox_available:close()

local vfox_sdk = clink.argmatcher():nofiles():addarg(vfox_sdk_table)

local vfox_ls_func = function()
    local pre, ls = '', {}
    local vfox_ls = io.popen('vfox ls')
    for line in vfox_ls:lines() do
        local txt = line:gsub('%c%[%d+m', ''):match('(%a.+)')
        if txt then
            if string.find(txt, 'v') == 1 then
                table.insert(ls, pre .. '@' .. string.sub(txt, 2))
            else
                pre = txt
            end
        end
    end
    vfox_ls:close()
    return ls
end
local vfox_ls_version = clink.argmatcher():nofiles():addarg({ vfox_ls_func })

local vfox_use_list = clink.argmatcher():nofiles():addarg({ vfox_ls_func, vfox_sdk })
local vfox_use = clink.argmatcher():nofiles():addarg({ vfox_use_list }):addflags({
    '--global' .. vfox_use_list, '-g' .. vfox_use_list,
    '--session' .. vfox_use_list, '-s' .. vfox_use_list,
    '--project' .. vfox_use_list, '-p' .. vfox_use_list,
})

local vfox_shell = clink.argmatcher():nofiles():addarg('bash', 'zsh', 'pwsh', 'fish', 'clink')

local vfox_env = clink.argmatcher():nofiles():addarg({ vfox_sdk }):addflags({
    '--shell' .. vfox_shell, '-s' .. vfox_shell,
    '--cleanup', '-c',
    '--json', '-j',
})

clink.argmatcher('vfox'):nofiles():addarg({
    'add' .. clink.argmatcher():nofiles():addarg({ vfox_sdk }):addflags('--source', '-s', '--alias'),
    'use' .. vfox_use, 'u' .. vfox_use,
    'info' .. vfox_sdk,
    'remove' .. vfox_sdk,
    'search' .. vfox_sdk,
    'update' .. vfox_sdk,
    'available',
    'current' .. vfox_sdk, 'c' .. vfox_sdk,
    'list' .. vfox_sdk, 'ls' .. vfox_sdk,
    'uninstall' .. vfox_ls_version, 'un' .. vfox_ls_version,
    'install' .. vfox_sdk, 'i' .. vfox_sdk,
    'env' .. vfox_env,
    'activate' .. vfox_shell,
})

local vfox_setenv = function(str)
    local key, val = str:match('^set "(.+)=(.*)"')
    if key and val then
        return os.setenv(key, val ~= '' and val or nil)
    end
end

os.setenv('__VFOX_PID', os.getpid())
local vfox_activate = io.popen('vfox activate clink')
for line in vfox_activate:lines() do
    if not vfox_setenv(line) then
        io.popen(line):close()
    end
end
vfox_activate:close()

local vfox_prompt = clink.promptfilter(30)
function vfox_prompt:filter(prompt)
    local env = io.popen('vfox env -s clink')
    for line in env:lines() do
        vfox_setenv(line)
    end
    env:close()
end
