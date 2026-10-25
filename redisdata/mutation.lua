local mode = ARGV[1]
local expiry = ARGV[2]
local result = 1
if #ARGV > 2 then result = redis.call(unpack(ARGV, 3)) end
if mode == 'persistent' then redis.call('PERSIST', KEYS[#KEYS]) end
if mode == 'relative' then redis.call('PEXPIRE', KEYS[#KEYS], expiry) end
if mode == 'absolute' then redis.call('PEXPIREAT', KEYS[#KEYS], expiry) end
return result
