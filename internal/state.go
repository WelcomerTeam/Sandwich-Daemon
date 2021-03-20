package gateway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	discord "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/xerrors"
)

var NoHandler = xerrors.New("No registered handler for event")

var stateHandlers = make(map[string]func(ctx *StateCtx,
	msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error))

type StateCtx struct {
	Sg *Sandwich
	Mg *Manager
	Sh *Shard

	Vars map[string]interface{}
}

// registerState registers a state handler.
func registerState(eventType string, handler func(ctx *StateCtx,
	msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error)) {
	stateHandlers[eventType] = handler
}

// StateDispatch handles selecting the proper state handler and executing it.
func (sg *Sandwich) StateDispatch(ctx *StateCtx,
	event discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	if f, ok := stateHandlers[event.Type]; ok {
		return f(ctx, event)
	}

	return result, false, xerrors.Errorf("failed to dispatch: %w", NoHandler)
}

// CreateKey creates a redis key from a format and values.
func (mg *Manager) CreateKey(key string, values ...interface{}) string {
	return mg.Configuration.Caching.RedisPrefix + ":" + fmt.Sprintf(key, values...)
}

// StateReady handles the READY event.
func StateReady(ctx *StateCtx, msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	var packet discord.Ready

	var guildPayload discord.GuildCreate

	err = json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return result, false, xerrors.Errorf("Failed to unmarshal message: %w", err)
	}

	ctx.Sh.Logger.Info().Msg("Received READY payload")

	ctx.Sh.Lock()
	ctx.Sh.sessionID = packet.SessionID
	ctx.Sh.User = packet.User
	ctx.Sh.Unlock()

	events := make([]discord.ReceivedPayload, 0)
	guildIDs := make([]snowflake.ID, 0, len(packet.Guilds))

	ctx.Sh.UnavailableMu.Lock()
	ctx.Sh.Unavailable = make(map[snowflake.ID]bool)

	for _, guild := range packet.Guilds {
		ctx.Sh.Unavailable[guild.ID] = guild.Unavailable
	}
	ctx.Sh.UnavailableMu.Unlock()

	guildCreateEvents := 0

	// If true will only run events once finished loading.
	// Todo: Add to sandwich configuration.
	preemptiveEvents := false

	t := time.NewTicker(timeoutDuration)

ready:
	for {
		select {
		case err := <-ctx.Sh.ErrorCh:
			if !xerrors.Is(err, context.Canceled) {
				ctx.Sh.Logger.Error().Err(err).Msg("Encountered error whilst waiting lazy loading")
			}

			break ready
		case msg := <-ctx.Sh.MessageCh:
			if msg.Type == "GUILD_CREATE" {
				guildCreateEvents++

				if err = ctx.Sh.decodeContent(msg, &guildPayload); err != nil {
					ctx.Sh.Logger.Error().Err(err).Msg("Failed to unmarshal GUILD_CREATE")
				} else {
					guildIDs = append(guildIDs, guildPayload.ID)
				}

				t.Reset(timeoutDuration)
			}

			if preemptiveEvents {
				events = append(events, msg)
			} else if err = ctx.Sh.OnDispatch(msg); err != nil && !xerrors.Is(err, NoHandler) {
				ctx.Sh.Logger.Error().Err(err).Msg("Failed dispatching event")
			}
		case <-t.C:
			ctx.Sh.Manager.Sandwich.ConfigurationMu.RLock()

			if ctx.Sh.Manager.Configuration.Caching.RequestMembers {
				for _, guildID := range guildIDs {
					ctx.Sh.Logger.Trace().Msgf("Requesting guild members for guild %d", guildID)

					if err := ctx.Sh.SendEvent(discord.GatewayOpRequestGuildMembers, discord.RequestGuildMembers{
						GuildID: guildID,
						Query:   "",
						Limit:   0,
					}); err != nil {
						ctx.Sh.Logger.Error().Err(err).Msgf("Failed to request guild members")
					}
				}
			}
			ctx.Sh.Manager.Sandwich.ConfigurationMu.RUnlock()

			break ready
		}
	}

	ctx.Sh.ready <- void{}
	if err := ctx.Sh.SetStatus(structs.ShardReady); err != nil {
		ctx.Sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
	}

	if preemptiveEvents {
		ctx.Sh.Logger.Debug().Int("events", len(events)).Msg("Dispatching preemptive events")

		for _, event := range events {
			ctx.Sh.Logger.Debug().Str("type", event.Type).Send()

			if err = ctx.Sh.OnDispatch(event); err != nil {
				ctx.Sh.Logger.Error().Err(err).Msg("Failed whilst dispatching preemptive events")
			}
		}

		ctx.Sh.Logger.Debug().Msg("Finished dispatching events")
	}

	return result, false, nil
}

// StateGuildCreate handles the GUILD_CREATE event.
func StateGuildCreate(ctx *StateCtx, msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	var packet discord.GuildCreate

	err = json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return result, false, xerrors.Errorf("Failed to unmarshal message: %w", err)
	}

	sg := discord.StateGuild{}

	roles, emojis, channels := sg.FromGuild(packet.Guild)

	if k, err := msgpack.Marshal(sg); err == nil {
		err = ctx.Mg.RedisClient.HSet(ctx.Mg.ctx, ctx.Mg.CreateKey("guilds"), sg.ID, k).Err()
		if err != nil {
			ctx.Mg.Logger.Error().Err(err).Msg("Failed to push guild to redis")
		}
	}

	if len(roles) > 0 {
		err = ctx.Mg.RedisClient.HSet(ctx.Mg.ctx, ctx.Mg.CreateKey("guild:%s:roles", sg.ID), roles).Err()
		if err != nil {
			ctx.Mg.Logger.Error().Err(err).Msg("Failed to push guild roles to redis")
		}
	}

	if len(emojis) > 0 {
		err = ctx.Mg.RedisClient.HSet(ctx.Mg.ctx, ctx.Mg.CreateKey("emojis"), emojis).Err()
		if err != nil {
			ctx.Mg.Logger.Error().Err(err).Msg("Failed to push guild emojis to redis")
		}
	}

	if len(channels) > 0 {
		err = ctx.Mg.RedisClient.HSet(ctx.Mg.ctx, ctx.Mg.CreateKey("channels"), channels).Err()
		if err != nil {
			ctx.Mg.Logger.Error().Err(err).Msg("Failed to push guild channels to redis")
		}
	}

	lazy, _ := ctx.Vars["lazy"].(bool)

	ctx.Sh.UnavailableMu.RLock()
	ok, unavailable := ctx.Sh.Unavailable[sg.ID]
	ctx.Sh.UnavailableMu.RUnlock()

	// Check if the guild is unavailable.
	if ok {
		if unavailable {
			ctx.Sh.Logger.Trace().Str("id", sg.ID.String()).Msg("Lazy loaded guild")

			lazy = true || lazy
		}

		ctx.Sh.UnavailableMu.Lock()
		delete(ctx.Sh.Unavailable, sg.ID)
		ctx.Sh.UnavailableMu.Unlock()
	}

	return structs.StateResult{
		Data: sg,
		Extra: map[string]interface{}{
			"lazy": lazy,
		},
	}, true, nil
}

// StateGuildMembersChunk handles the GUILD_MEMBERS_CHUNK event.
func StateGuildMembersChunk(ctx *StateCtx, msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	if !ctx.Mg.Configuration.Caching.CacheMembers {
		return
	}

	var packet discord.GuildMembersChunk

	err = json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return result, false, xerrors.Errorf("Failed to unmarshal message: %w", err)
	}

	members := make([]interface{}, 0, len(packet.Members))

	// Create list of msgpacked members
	for _, member := range packet.Members {
		if ma, err := msgpack.Marshal(member); err == nil {
			members = append(members, ma)
		}
	}

	err = ctx.Mg.RedisClient.Eval(
		ctx.Mg.ctx,
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

				redis.log(3, user['id'], type(user['id']), )

				if cacheUsers then
						redis.call("HSET", redisPrefix .. ":user", user['id'], cmsgpack.pack(user))
				end

				call("HSET", redisPrefix .. ":guild:" .. guildID .. ":members", user['id'], cmsgpack.pack(member))

				if storeMutuals then
						call("SADD", redisPrefix .. ":mutual:" .. user['id'], guildID)
				end

		end
		`,
		[]string{
			ctx.Mg.Configuration.Caching.RedisPrefix,
			packet.GuildID.String(),
			strconv.FormatBool(ctx.Mg.Configuration.Caching.StoreMutuals),
			strconv.FormatBool(ctx.Mg.Configuration.Caching.CacheUsers),
		},
		members,
	).Err()
	if err != nil {
		return result, false, xerrors.Errorf("Failed to process guild member chunks: %w", err)
	}

	// We do not want to send member chunks to
	// consumers as they will have no use.

	return result, false, nil
}

func init() { //nolint
	registerState("READY", StateReady)
	registerState("GUILD_CREATE", StateGuildCreate)
	registerState("GUILD_MEMBERS_CHUNK", StateGuildMembersChunk)
}
