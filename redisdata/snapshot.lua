local key = KEYS[1]
local position = ARGV[1]
local skipped = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local budget = tonumber(ARGV[4])
local kind = redis.call('TYPE', key).ok
local total = 0
local nextposition = ''
local nextoffset = 0
local complete = false
local data = {}
local metadata = {}
local used = 0
local usedelements = 0
local digest = ''
local function number(value)
    return string.format('%.0f', value)
end
local function size(value)
    if type(value) == 'table' then
        local length, elements = 0, 1
        for _, child in ipairs(value) do
            local childlength, childelements = size(child)
            length = length + childlength
            elements = elements + childelements
            if elements > 16384 then return math.huge, elements end
        end
        return length, elements
    end
    return #tostring(value), 1
end
local function append(value)
    local length, elements = size(value)
    if #data >= limit or used + length > budget or usedelements + elements > 16384 then return false end
    data[#data + 1] = value
    used = used + length
    usedelements = usedelements + elements
    return true
end
if kind == 'none' then
    return {kind, '0', '', '0', '1', {}, {}, ''}
elseif kind == 'string' then
    total = redis.call('STRLEN', key)
    local start = tonumber(position)
    if start > total then return redis.error_reply('SQLDASH_INPUT') end
    local content = redis.call('GETRANGE', key, start, start + budget - 1)
    data = {content}
    if start + #content < total then nextposition = number(start + #content) end
    complete = start == 0 and #content == total
elseif kind == 'hash' or kind == 'set' then
    local command = kind == 'hash' and 'HSCAN' or 'SSCAN'
    total = redis.call(kind == 'hash' and 'HLEN' or 'SCARD', key)
    local scanned = redis.call(command, key, position, 'COUNT', limit)
    local stride = kind == 'hash' and 2 or 1
    local entries = #scanned[2] / stride
    if skipped > entries then return redis.error_reply('SQLDASH_STALE') end
    local index = skipped
    while index < entries do
        local value = scanned[2][index * stride + 1]
        if stride == 2 then value = {value, scanned[2][index * stride + 2]} end
        if not append(value) then
            if #data == 0 then return redis.error_reply('SQLDASH_LIMIT') end
            break
        end
        index = index + 1
    end
    if index < entries then
        nextposition = position
        nextoffset = index
    elseif scanned[1] ~= '0' then
        nextposition = scanned[1]
    end
    complete = position == '0' and skipped == 0 and nextposition == '' and #data == total
elseif kind == 'list' or kind == 'zset' then
    total = redis.call(kind == 'list' and 'LLEN' or 'ZCARD', key)
    local start = tonumber(position)
    if start > total then return redis.error_reply('SQLDASH_STALE') end
    local entries
    if kind == 'list' then
        entries = redis.call('LRANGE', key, start, start + limit - 1)
    else
        entries = redis.call('ZRANGE', key, start, start + limit - 1, 'WITHSCORES')
    end
    local stride = kind == 'list' and 1 or 2
    for index = 1, #entries, stride do
        local value = kind == 'list' and {number(start + #data), entries[index]} or {entries[index], entries[index + 1]}
        if not append(value) then
            if #data == 0 then return redis.error_reply('SQLDASH_LIMIT') end
            break
        end
    end
    if start + #data < total then nextposition = number(start + #data) end
    complete = start == 0 and #data == total
elseif kind == 'stream' then
    total = redis.call('XLEN', key)
    local start = position == '0' and '-' or '(' .. position
    local entries = redis.call('XRANGE', key, start, '+', 'COUNT', limit + 1)
    for index, entry in ipairs(entries) do
        if not append(entry) then
            if #data == 0 then return redis.error_reply('SQLDASH_LIMIT') end
            break
        end
    end
    if #entries > #data then nextposition = data[#data][1] end
    complete = position == '0' and #data == total
    if complete then
        local information = redis.call('XINFO', 'STREAM', key)
        local groups = 0
        for index = 1, #information, 2 do
            if information[index] == 'groups' then groups = information[index + 1] end
        end
        if groups <= limit then
            local groupdata = redis.call('XINFO', 'GROUPS', key)
            if size(groupdata) + used <= budget then
                for _, group in ipairs(groupdata) do
                    local converted = {}
                    for index = 1, #group, 2 do
                        local value = group[index + 1]
                        if type(value) == 'number' then value = number(value) end
                        if value == false then value = '' end
                        converted[#converted + 1] = group[index]
                        converted[#converted + 1] = value
                    end
                    metadata[#metadata + 1] = converted
                end
                digest = redis.sha1hex(cjson.encode({data, information, metadata}))
            else complete = false end
        else complete = false end
    end
elseif kind == 'ReJSON-RL' then
    total = -1
    local memory = redis.call('MEMORY', 'USAGE', key)
    if memory <= budget then
        local content = redis.call('JSON.GET', key, '.')
        total = #content
        if #content <= budget then data = {content}; complete = true end
    end
    metadata = {{'memoryBytes', number(memory)}}
else
    total = -1
    local memory = redis.call('MEMORY', 'USAGE', key)
    metadata = {{'nativeType', kind}, {'memoryBytes', number(memory)}}
    if memory <= budget / 4 then
        local serialized = redis.call('DUMP', key)
        if #serialized <= budget then data = {serialized}; total = #serialized; complete = true end
    end
end
if complete and kind ~= 'stream' then
    local serialized
    if kind ~= 'string' and kind ~= 'hash' and kind ~= 'list' and kind ~= 'set' and kind ~= 'zset' and kind ~= 'ReJSON-RL' then
        serialized = data[1]
    else
        serialized = redis.call('DUMP', key)
    end
    if #serialized <= budget then digest = redis.sha1hex(serialized) else complete = false end
end
return {kind, number(total), nextposition, number(nextoffset), complete and '1' or '0', data, metadata, digest}
