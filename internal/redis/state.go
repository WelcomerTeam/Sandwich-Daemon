package redis

import "github.com/go-redis/redis/v8"

var ProcessGuildMembersChunk = redis.NewScript(
	`
		local redisPrefix = KEYS[1]
		local guildID = KEYS[2]
		local storeMutuals = KEYS[3] == true
		local cacheUsers = KEYS[4] == true

		local member
		local user

		local call = redis.call

		redis.log(3, "Received " .. #ARGV .. " member(s) in GuildMembersChunk")

		for i,k in pairs(ARGV) do
			member = cmsgpack.unpack(k)

			-- We do not want the user object stored in the member
			local user = member['user']
			user['id'] = string.format("%.0f",user['id'])

			member['user'] = nil
			member['id'] = user['id']

			if cacheUsers then
				redis.call("HSET", redisPrefix .. ":user", user['id'], cmsgpack.pack(user))
			end

			call("HSET", redisPrefix .. ":guild:" .. guildID .. ":members", user['id'], cmsgpack.pack(member))

			if storeMutuals then
				call("SADD", redisPrefix .. ":mutual:" .. user['id'], guildID)
			end
		end

		return true
	`)
