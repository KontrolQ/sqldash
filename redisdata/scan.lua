local arguments = {'SCAN', ARGV[1], 'COUNT', ARGV[2]}
if ARGV[3] ~= '' then arguments[#arguments+1] = 'MATCH'; arguments[#arguments+1] = ARGV[3] end
if ARGV[4] ~= '' then arguments[#arguments+1] = 'TYPE'; arguments[#arguments+1] = ARGV[4] end
local scanned = redis.call(unpack(arguments))
local result = {}
local bytes = 0
if #scanned[2] > 4096 then return redis.error_reply('SQLDASH_LIMIT') end
for _, key in ipairs(scanned[2]) do
    bytes = bytes + #key
    if #key > 8192 or bytes > 262144 then return redis.error_reply('SQLDASH_LIMIT') end
    local kind = redis.call('TYPE', key).ok
    if kind ~= 'none' then result[#result+1] = {key, kind} end
end
return {scanned[1], result}
