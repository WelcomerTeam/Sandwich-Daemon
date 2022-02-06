package internal

import (
	"context"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/savsgio/gotils/strconv"
	"github.com/savsgio/gotils/strings"
	"golang.org/x/xerrors"
)

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (err error))

// List of handlers for dispatch events.
var dispatchHandlers = make(map[string]func(ctx *StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error))

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
	Guilds   map[discord.Snowflake]*sandwich_structs.StateGuild

	guildMembersMu sync.RWMutex
	GuildMembers   map[discord.Snowflake]*sandwich_structs.StateGuildMembers

	guildChannelsMu sync.RWMutex
	GuildChannels   map[discord.Snowflake]*sandwich_structs.StateGuildChannels

	guildRolesMu sync.RWMutex
	GuildRoles   map[discord.Snowflake]*sandwich_structs.StateGuildRoles

	guildEmojisMu sync.RWMutex
	GuildEmojis   map[discord.Snowflake]*sandwich_structs.StateGuildEmojis

	usersMu sync.RWMutex
	Users   map[discord.Snowflake]*sandwich_structs.StateUser

	dmChannelsMu sync.RWMutex
	dmChannels   map[discord.Snowflake]*sandwich_structs.StateDMChannel

	mutualsMu sync.RWMutex
	Mutuals   map[discord.Snowflake]*sandwich_structs.StateMutualGuilds
}

func NewSandwichState() (st *SandwichState) {
	st = &SandwichState{
		guildsMu: sync.RWMutex{},
		Guilds:   make(map[discord.Snowflake]*sandwich_structs.StateGuild),

		guildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[discord.Snowflake]*sandwich_structs.StateGuildMembers),

		guildChannelsMu: sync.RWMutex{},
		GuildChannels:   make(map[discord.Snowflake]*sandwich_structs.StateGuildChannels),

		guildRolesMu: sync.RWMutex{},
		GuildRoles:   make(map[discord.Snowflake]*sandwich_structs.StateGuildRoles),

		guildEmojisMu: sync.RWMutex{},
		GuildEmojis:   make(map[discord.Snowflake]*sandwich_structs.StateGuildEmojis),

		usersMu: sync.RWMutex{},
		Users:   make(map[discord.Snowflake]*sandwich_structs.StateUser),

		dmChannelsMu: sync.RWMutex{},
		dmChannels:   make(map[discord.Snowflake]*sandwich_structs.StateDMChannel),

		mutualsMu: sync.RWMutex{},
		Mutuals:   make(map[discord.Snowflake]*sandwich_structs.StateMutualGuilds),
	}

	return st
}

func (sh *Shard) OnEvent(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) {
	err := GatewayDispatch(ctx, sh, msg, trace)
	if err != nil {
		if xerrors.Is(err, ErrNoGatewayHandler) {
			sh.Logger.Warn().
				Int("op", int(msg.Op)).
				Str("type", msg.Type).
				Msg("Gateway sent unknown packet")
		}
	}
}

// OnDispatch handles routing of discord event.
func (sh *Shard) OnDispatch(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (err error) {
	if sh.Manager.ProducerClient == nil {
		return ErrProducerMissing
	}

	sh.Manager.eventBlacklistMu.RLock()
	contains := strings.Include(sh.Manager.eventBlacklist, msg.Type)
	sh.Manager.eventBlacklistMu.RUnlock()

	if contains {
		return
	}

	sh.Manager.configurationMu.RLock()
	cacheUsers := sh.Manager.Configuration.Caching.CacheUsers
	cacheMembers := sh.Manager.Configuration.Caching.CacheMembers
	storeMutuals := sh.Manager.Configuration.Caching.StoreMutuals
	sh.Manager.configurationMu.RUnlock()

	trace["state"] = discord.Int64(time.Now().Unix())

	result, continuable, err := StateDispatch(&StateCtx{
		context:      ctx,
		Shard:        sh,
		CacheUsers:   cacheUsers,
		CacheMembers: cacheMembers,
		StoreMutuals: storeMutuals,
	}, msg, trace)
	if err != nil {
		if !xerrors.Is(err, ErrNoDispatchHandler) {
			sh.Logger.Error().Err(err).Str("data", strconv.B2S(msg.Data)).Msg("Encountered error whilst handling " + msg.Type)
		}

		return err
	}

	if !continuable {
		return
	}

	sh.Manager.produceBlacklistMu.RLock()
	contains = strings.Include(sh.Manager.produceBlacklist, msg.Type)
	sh.Manager.produceBlacklistMu.RUnlock()

	if contains {
		return
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

func registerGatewayEvent(op discord.GatewayOp, handler func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (err error)) {
	gatewayHandlers[op] = handler
}

func registerDispatch(eventType string, handler func(ctx *StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error)) {
	dispatchHandlers[eventType] = handler
}

// GatewayDispatch handles selecting the proper gateway handler and executing it.
func GatewayDispatch(ctx context.Context, sh *Shard,
	event discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (err error) {
	if f, ok := gatewayHandlers[event.Op]; ok {
		return f(ctx, sh, event, trace)
	}

	sh.Logger.Warn().Int("op", int(event.Op)).Msg("No gateway handler found")

	return ErrNoGatewayHandler
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx *StateCtx,
	event discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		ctx.Logger.Trace().Str("type", event.Type).Msg("State Dispatch")

		return f(ctx, event, trace)
	}

	ctx.Logger.Warn().Str("type", event.Type).Msg("No dispatch handler found")

	return result, false, ErrNoDispatchHandler
}
