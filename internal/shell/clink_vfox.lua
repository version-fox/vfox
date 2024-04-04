local vfox_sdk_table = {}
local vfox_available = io.popen('vfox available')
for line in vfox_available:lines() do
    local sdk, yes = line:gsub('%c%[%d+m', ''):match('^(%a+)%s+(%u%u%u?)')
    if sdk and yes and (yes == 'YES' or yes == 'NO') then
        table.insert(vfox_sdk_table, sdk)
    end
end
vfox_available:close()

local vfox_ls_func = function()
    local pre, ls = '', {}
    local vfox_ls = io.popen('vfox ls')
    for line in vfox_ls:lines() do
        local txt = line:gsub('%c%[%d+m', ''):match('^%A+(%a.+)')
        if txt then
            if string.find(txt, 'v') == 1 then
                ls[pre] = true
                table.insert(ls, pre .. '@' .. string.sub(txt, 2))
            else
                pre = txt
                ls[pre] = false
            end
        end
    end
    vfox_ls:close()
    return ls
end

local vfox_sdk = clink.argmatcher():nofiles():addarg(function()
    local ls, res = vfox_ls_func(), {}
    for k, v in pairs(ls) do
        if type(v) == 'boolean' then
            table.insert(res, k)
        end
    end
    return res
end):addflags('--help', '-h')
local vfox_use = clink.argmatcher():nofiles():addarg(function()
    local ls = vfox_ls_func()
    for k, v in pairs(ls) do
        if v == true then
            table.insert(ls, k)
        end
    end
    return ls
end):addflags('--global', '-g', '--session', '-s', '--project', '-p', '--help', '-h')
local vfox_help = clink.argmatcher():nofiles():addflags('--help', '-h')
local vfox_shell = clink.argmatcher():nofiles():addarg('bash', 'zsh', 'pwsh', 'fish', 'clink')
local vfox_ls_version = clink.argmatcher():nofiles():addarg({ vfox_ls_func }):addflags('--help', '-h')

clink.argmatcher('vfox'):nofiles():addarg({
    'add' .. clink.argmatcher():nofiles():addarg(vfox_sdk_table):addflags('--source', '-s', '--alias', '--help', '-h'),
    'use' .. vfox_use, 'u' .. vfox_use,
    'info' .. vfox_sdk,
    'remove' .. vfox_sdk,
    'search' .. vfox_sdk,
    'update' .. vfox_sdk,
    'available' .. vfox_help,
    'current' .. vfox_sdk, 'c' .. vfox_sdk,
    'list' .. vfox_sdk, 'ls' .. vfox_sdk,
    'uninstall' .. vfox_ls_version, 'un' .. vfox_ls_version,
    'install' .. vfox_sdk, 'i' .. vfox_sdk,
    'env' .. clink.argmatcher():nofiles():addflags({
        '--shell' .. vfox_shell, '-s' .. vfox_shell,
        '--cleanup', '-c',
        '--json', '-j',
        '--help', '-h',
    }),
    'activate' .. vfox_shell,
    'help', 'h',
}):addflags('--debug', '--help', '-h', '--version', '-v', '-V')

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
