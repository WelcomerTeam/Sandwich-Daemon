package internal

import (
	"sync"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"github.com/savsgio/gotils/strings"
	"golang.org/x/xerrors"
)

// List of handlers for dispatch events.
var dispatchHandlers = make(map[string]func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error))

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(ctx *StateCtx, msg discord.GatewayPayload) (err error))

type StateCtx struct {
	*Shard

	Vars map[string]interface{}
}

// SandwichState stores the collective state of all ShardGroups
// across all Managers.
type SandwichState struct {
	guildsMu sync.RWMutex
	Guilds   map[discord.Snowflake]*discord.StateGuild

	guildMembersMu sync.RWMutex
	GuildMembers   map[discord.Snowflake]*discord.StateGuildMembers

	channelsMu sync.RWMutex
	Channels   map[discord.Snowflake]*discord.StateChannel

	rolesMu sync.RWMutex
	Roles   map[discord.Snowflake]*discord.StateRole

	emojisMu sync.RWMutex
	Emojis   map[discord.Snowflake]*discord.StateEmoji

	usersMu sync.RWMutex
	Users   map[discord.Snowflake]*discord.StateUser
}

func NewSandwichState() (st *SandwichState) {
	st = &SandwichState{
		guildsMu: sync.RWMutex{},
		Guilds:   make(map[discord.Snowflake]*discord.StateGuild),

		guildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[discord.Snowflake]*discord.StateGuildMembers),

		channelsMu: sync.RWMutex{},
		Channels:   make(map[discord.Snowflake]*discord.StateChannel),

		rolesMu: sync.RWMutex{},
		Roles:   make(map[discord.Snowflake]*discord.StateRole),

		emojisMu: sync.RWMutex{},
		Emojis:   make(map[discord.Snowflake]*discord.StateEmoji),

		usersMu: sync.RWMutex{},
		Users:   make(map[discord.Snowflake]*discord.StateUser),
	}

	return st
}

func (sh *Shard) OnEvent(msg discord.GatewayPayload) {
	fin := make(chan void, 1)

	go func() {
		since := time.Now()

		t := time.NewTicker(DispatchWarningTimeout)
		defer t.Stop()

		for {
			select {
			case <-fin:
				return
			case <-t.C:
				sh.Logger.Warn().
					Str("type", msg.Type).
					Int("op", int(msg.Op)).
					Dur("since", time.Now().Sub(since)).
					Msg("Event is taking too long")
			}
		}
	}()

	defer close(fin)

	err := GatewayDispatch(&StateCtx{
		Shard: sh,
	}, msg)
	if err != nil {
		if xerrors.Is(err, ErrNoGatewayHandler) {
			sh.Logger.Warn().
				Int("op", int(msg.Op)).
				Str("type", msg.Type).
				Msg("Gateway sent unknown packet")
		}
	}

	return
}

// OnDispatch handles routing of discord event.
func (sh *Shard) OnDispatch(msg discord.GatewayPayload) (err error) {
	if sh.Manager.ProducerClient == nil {
		return ErrProducerMissing
	}

	sh.Manager.eventBlacklistMu.RLock()
	contains := strings.Include(sh.Manager.eventBlacklist, msg.Type)
	sh.Manager.eventBlacklistMu.RUnlock()

	if contains {
		return
	}

	result, continuable, err := StateDispatch(&StateCtx{
		Shard: sh,
	}, msg)

	if err != nil {
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

	packet := sh.Sandwich.payloadPool.Get().(*structs.SandwichPayload)
	defer sh.Sandwich.payloadPool.Put(packet)

	packet.GatewayPayload = msg
	packet.Data = result.Data
	packet.Extra = result.Extra

	return sh.PublishEvent(packet)
}

func registerGatewayEvent(op discord.GatewayOp, handler func(ctx *StateCtx, msg discord.GatewayPayload) (err error)) {
	gatewayHandlers[op] = handler
}

func registerDispatch(eventType string, handler func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error)) {
	dispatchHandlers[eventType] = handler
}

// GatewayDispatch handles selecting the proper gateway handler and executing it.
func GatewayDispatch(ctx *StateCtx,
	event discord.GatewayPayload) (err error) {
	if f, ok := gatewayHandlers[event.Op]; ok {
		return f(ctx, event)
	}

	return ErrNoGatewayHandler
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx *StateCtx,
	event discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		return f(ctx, event)
	}

	return result, false, ErrNoDispatchHandler
}

func gatewayOpDispatch(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	go func() {
		ticket := ctx.Sandwich.EventPool.Wait()
		defer ctx.Sandwich.EventPool.FreeTicket(ticket)

		err = ctx.OnDispatch(msg)
		if err != nil && !xerrors.Is(err, ErrNoDispatchHandler) {
			ctx.Logger.Error().Err(err).Msg("State dispatch failed")
		}
	}()

	return
}

func gatewayOpHeartbeat(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpIdentify(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpStatusUpdate(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpVoiceStateUpdate(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpResume(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpReconnect(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpRequestGuildMembers(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpInvalidSession(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpHello(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func gatewayOpHeartbeatACK(ctx *StateCtx, msg discord.GatewayPayload) (err error) {
	return
}

func init() {
	registerGatewayEvent(discord.GatewayOpDispatch, gatewayOpDispatch)
	registerGatewayEvent(discord.GatewayOpHeartbeat, gatewayOpHeartbeat)
	registerGatewayEvent(discord.GatewayOpIdentify, gatewayOpIdentify)
	registerGatewayEvent(discord.GatewayOpStatusUpdate, gatewayOpStatusUpdate)
	registerGatewayEvent(discord.GatewayOpVoiceStateUpdate, gatewayOpVoiceStateUpdate)
	registerGatewayEvent(discord.GatewayOpResume, gatewayOpResume)
	registerGatewayEvent(discord.GatewayOpReconnect, gatewayOpReconnect)
	registerGatewayEvent(discord.GatewayOpRequestGuildMembers, gatewayOpRequestGuildMembers)
	registerGatewayEvent(discord.GatewayOpInvalidSession, gatewayOpInvalidSession)
	registerGatewayEvent(discord.GatewayOpHello, gatewayOpHello)
	registerGatewayEvent(discord.GatewayOpHeartbeatACK, gatewayOpHeartbeatACK)
}
