package internal

import (
	"context"
	"encoding/json"
	"errors"
	"runtime/debug"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/savsgio/gotils/strconv"
	"github.com/savsgio/gotils/strings"
)

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error)

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
	Guilds *csmap.CsMap[discord.Snowflake, *discord.Guild]

	GuildMembers *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateGuildMembers]

	GuildChannels *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateGuildChannels]

	GuildRoles *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateGuildRoles]

	GuildEmojis *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateGuildEmojis]

	Users *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateUser]

	DmChannels *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateDMChannel]

	Mutuals *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateMutualGuilds]

	GuildVoiceStates *csmap.CsMap[discord.Snowflake, *sandwich_structs.StateGuildVoiceStates]
}

func NewSandwichState() *SandwichState {
	state := &SandwichState{
		Guilds: csmap.Create(
			csmap.WithSize[discord.Snowflake, *discord.Guild](1000),
		),

		GuildMembers: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateGuildMembers](1000),
		),

		GuildChannels: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateGuildChannels](1000),
		),

		GuildRoles: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateGuildRoles](1000),
		),

		GuildEmojis: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateGuildEmojis](1000),
		),

		Users: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateUser](1000),
		),

		DmChannels: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateDMChannel](1000),
		),

		Mutuals: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateMutualGuilds](1000),
		),

		GuildVoiceStates: csmap.Create(
			csmap.WithSize[discord.Snowflake, *sandwich_structs.StateGuildVoiceStates](1000),
		),
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
	}
}

// OnDispatch handles routing of discord event.
func (sh *Shard) OnDispatch(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
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
	contains := strings.Include(sh.Manager.eventBlacklist, msg.Type)
	sh.Manager.eventBlacklistMu.RUnlock()

	if contains {
		return nil
	}

	sh.Manager.configurationMu.RLock()
	cacheUsers := sh.Manager.Configuration.Caching.CacheUsers
	cacheMembers := sh.Manager.Configuration.Caching.CacheMembers
	storeMutuals := sh.Manager.Configuration.Caching.StoreMutuals
	sh.Manager.configurationMu.RUnlock()

	if trace == nil {
		trace = csmap.Create(
			csmap.WithSize[string, discord.Int64](uint64(1)),
		)
	}

	trace.Store("state", discord.Int64(time.Now().Unix()))

	result, continuable, err := StateDispatch(&StateCtx{
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
	contains = strings.Include(sh.Manager.produceBlacklist, msg.Type)
	sh.Manager.produceBlacklistMu.RUnlock()

	if contains {
		return nil
	}

	packet, _ := sh.Sandwich.payloadPool.Get().(*sandwich_structs.SandwichPayload)

	if packet == nil {
		return errors.New("failed to get sandwich payload from pool")
	}

	defer sh.Sandwich.payloadPool.Put(packet)

	// Directly copy op, sequence and type from original message.
	packet.Op = msg.Op
	packet.Sequence = msg.Sequence
	packet.Type = msg.Type

	// Setting result.Data will override what is sent to consumers.
	packet.Data = result.Data

	// Extra contains any extra information such as before state and if it is a lazy guild.
	packet.Extra = csmap.Create(
		csmap.WithSize[string, json.RawMessage](uint64(len(result.Extra))),
	)

	for key, value := range result.Extra {
		packet.Extra.Store(key, value)
	}

	packet.Trace = trace

	return sh.PublishEvent(ctx, packet)
}

func registerGatewayEvent(op discord.GatewayOp, handler func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error) {
	gatewayHandlers[op] = handler
}

func registerDispatch(eventType string, handler func(ctx *StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result sandwich_structs.StateResult, ok bool, err error)) {
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
func StateDispatch(ctx *StateCtx,
	event discord.GatewayPayload, trace sandwich_structs.SandwichTrace,
) (result sandwich_structs.StateResult, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		ctx.Logger.Trace().Str("type", event.Type).Msg("State Dispatch")

		return f(ctx, event, trace)
	}

	return WildcardEvent(ctx, event, trace)
}
