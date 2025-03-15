package internal

import (
	"context"
	"errors"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/savsgio/gotils/strconv"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	gotils_strings "github.com/savsgio/gotils/strings"
)

var AllowEventPassthrough = strings.ToLower(os.Getenv("ALLOW_EVENT_PASSTHROUGH")) == "true"

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error)

// List of handlers for dispatch events.
var dispatchHandlers = make(map[string]func(ctx StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error))

type StateCtx struct {
	CacheUsers   bool
	CacheMembers bool
	Stateless    bool
	StoreMutuals bool

	context context.Context
	*Shard
}

// SandwichState stores the collective state of all ShardGroups
// across all Managers.
type SandwichState struct {
	guildsMu sync.RWMutex
	Guilds   map[discord.Snowflake]sandwich_structs.StateGuild

	guildMembersMu sync.RWMutex
	GuildMembers   map[discord.Snowflake]*sandwich_structs.StateGuildMembers

	guildChannelsMu sync.RWMutex
	GuildChannels   map[discord.Snowflake]*sandwich_structs.StateGuildChannels

	guildRolesMu sync.RWMutex
	GuildRoles   map[discord.Snowflake]*sandwich_structs.StateGuildRoles

	guildEmojisMu sync.RWMutex
	GuildEmojis   map[discord.Snowflake]*sandwich_structs.StateGuildEmojis

	usersMu sync.RWMutex
	Users   map[discord.Snowflake]sandwich_structs.StateUser

	dmChannelsMu sync.RWMutex
	dmChannels   map[discord.Snowflake]sandwich_structs.StateDMChannel

	mutualsMu sync.RWMutex
	Mutuals   map[discord.Snowflake]*sandwich_structs.StateMutualGuilds

	guildVoiceStatesMu sync.RWMutex
	GuildVoiceStates   map[discord.Snowflake]*sandwich_structs.StateGuildVoiceStates
}

func NewSandwichState() *SandwichState {
	state := &SandwichState{
		guildsMu: sync.RWMutex{},
		Guilds:   make(map[discord.Snowflake]sandwich_structs.StateGuild),

		guildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[discord.Snowflake]*sandwich_structs.StateGuildMembers),

		guildChannelsMu: sync.RWMutex{},
		GuildChannels:   make(map[discord.Snowflake]*sandwich_structs.StateGuildChannels),

		guildRolesMu: sync.RWMutex{},
		GuildRoles:   make(map[discord.Snowflake]*sandwich_structs.StateGuildRoles),

		guildEmojisMu: sync.RWMutex{},
		GuildEmojis:   make(map[discord.Snowflake]*sandwich_structs.StateGuildEmojis),

		usersMu: sync.RWMutex{},
		Users:   make(map[discord.Snowflake]sandwich_structs.StateUser),

		dmChannelsMu: sync.RWMutex{},
		dmChannels:   make(map[discord.Snowflake]sandwich_structs.StateDMChannel),

		mutualsMu: sync.RWMutex{},
		Mutuals:   make(map[discord.Snowflake]*sandwich_structs.StateMutualGuilds),

		guildVoiceStatesMu: sync.RWMutex{},
		GuildVoiceStates:   make(map[discord.Snowflake]*sandwich_structs.StateGuildVoiceStates),
	}

	return state
}

func (sh *Shard) OnEvent(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) {
	err := GatewayDispatch(ctx, sh, msg, trace)
	if err != nil {
		if errors.Is(err, ErrNoGatewayHandler) {
			sh.Logger.Warn().
				Int("op", int(msg.Op)).
				Str("type", msg.Type).
				Msg("Gateway sent unknown packet")
		}

		sh.Logger.Error().Err(err).Msg("Gateway dispatch failed")
	}
}

// OnDispatch handles routing of discord event.
func (sh *Shard) OnDispatch(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (err error) {
	defer func() {
		if r := recover(); r != nil {
			errorMessage, ok := r.(error)

			if ok {
				sh.Logger.Error().
					Err(errorMessage).
					Int("op", int(msg.Op)).
					Str("type", msg.Type).
					Int("seq", int(msg.Sequence)).
					Bytes("data", msg.Data).
					Msg("Recovered panic in OnDispatch")
			} else {
				sh.Logger.Error().
					Str("err", "[unknown]").
					Int("op", int(msg.Op)).
					Str("type", msg.Type).
					Int("seq", int(msg.Sequence)).
					Bytes("data", msg.Data).
					Msg("Recovered panic in OnDispatch")
			}

			println(string(debug.Stack()))
		}
	}()

	if sh.Manager.ProducerClient == nil {
		return ErrProducerMissing
	}

	sh.Manager.eventBlacklistMu.RLock()
	contains := gotils_strings.Include(sh.Manager.eventBlacklist, msg.Type)
	sh.Manager.eventBlacklistMu.RUnlock()

	if contains {
		return nil
	}

	sh.Manager.configurationMu.RLock()
	cacheUsers := sh.Manager.Configuration.Caching.CacheUsers
	cacheMembers := sh.Manager.Configuration.Caching.CacheMembers
	storeMutuals := sh.Manager.Configuration.Caching.StoreMutuals
	sh.Manager.configurationMu.RUnlock()

	trace["state"] = discord.Int64(time.Now().Unix())

	result, continuable, err := StateDispatch(StateCtx{
		context:      ctx,
		Shard:        sh,
		CacheUsers:   cacheUsers,
		CacheMembers: cacheMembers,
		StoreMutuals: storeMutuals,
	}, msg, trace)
	if err != nil {
		if !errors.Is(err, ErrNoDispatchHandler) {
			sh.Logger.Error().Err(err).Str("data", strconv.B2S(msg.Data)).Msg("Encountered error whilst handling " + msg.Type)
		}

		return err
	}

	sh.ShardGroup.floodgateMu.RLock()
	floodgate := sh.ShardGroup.floodgate
	sh.ShardGroup.floodgateMu.RUnlock()

	if !floodgate || !continuable {
		return nil
	}

	sh.Manager.produceBlacklistMu.RLock()
	contains = gotils_strings.Include(sh.Manager.produceBlacklist, msg.Type)
	sh.Manager.produceBlacklistMu.RUnlock()

	if contains {
		return nil
	}

	packet, _ := sh.Sandwich.payloadPool.Get().(*sandwich_structs.SandwichPayload)
	defer sh.Sandwich.payloadPool.Put(packet)

	// Directly copy op, sequence and type from original message.
	packet.Op = msg.Op
	packet.Sequence = msg.Sequence
	packet.Type = msg.Type

	// Setting result.Data will override what is sent to consumers.
	packet.Data = result.Data

	// Extra contains any extra information such as before state and if it is a lazy guild.
	packet.Extra = result.Extra

	packet.Trace = trace

	return sh.PublishEvent(ctx, packet)
}

func registerGatewayEvent(op discord.GatewayOp, handler func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error) {
	gatewayHandlers[op] = handler
}

func registerDispatch(eventType string, handler func(ctx StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error)) {
	dispatchHandlers[eventType] = handler
}

// GatewayDispatch handles selecting the proper gateway handler and executing it.
func GatewayDispatch(ctx context.Context, sh *Shard,
	event discord.GatewayPayload, trace sandwich_structs.SandwichTrace,
) error {
	if f, ok := gatewayHandlers[event.Op]; ok {
		return f(ctx, sh, event, trace)
	}

	sh.Logger.Warn().Int("op", int(event.Op)).Msg("No gateway handler found")

	return ErrNoGatewayHandler
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx StateCtx, event discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		return f(ctx, event, trace)
	}

	ctx.Logger.Warn().
		Str("type", event.Type).
		Str("data", gotils_strconv.B2S(event.Data)).
		Int32("seq", event.Sequence).
		Int("op", int(event.Op)).
		Msg("No dispatch handler found")

	if AllowEventPassthrough {
		return OnPassthrough(ctx, event, trace)
	}

	return result, true, ErrNoDispatchHandler
}
