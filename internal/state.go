package internal

import (
	"fmt"
	"sync"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"github.com/savsgio/gotils/strings"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(sh *Shard, msg discord.GatewayPayload) (err error))

// List of handlers for dispatch events.
var dispatchHandlers = make(map[string]func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error))

type StateCtx struct {
	*Shard

	Vars map[string]interface{}
}

// SandwichState stores the collective state of all ShardGroups
// across all Managers.
type SandwichState struct {
	guildsMu sync.RWMutex
	Guilds   map[discord.Snowflake]*structs.StateGuild

	guildMembersMu sync.RWMutex
	GuildMembers   map[discord.Snowflake]*structs.StateGuildMembers

	channelsMu sync.RWMutex
	Channels   map[discord.Snowflake]*structs.StateChannel

	rolesMu sync.RWMutex
	Roles   map[discord.Snowflake]*structs.StateRole

	emojisMu sync.RWMutex
	Emojis   map[discord.Snowflake]*structs.StateEmoji

	usersMu sync.RWMutex
	Users   map[discord.Snowflake]*structs.StateUser
}

func NewSandwichState() (st *SandwichState) {
	st = &SandwichState{
		guildsMu: sync.RWMutex{},
		Guilds:   make(map[discord.Snowflake]*structs.StateGuild),

		guildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[discord.Snowflake]*structs.StateGuildMembers),

		channelsMu: sync.RWMutex{},
		Channels:   make(map[discord.Snowflake]*structs.StateChannel),

		rolesMu: sync.RWMutex{},
		Roles:   make(map[discord.Snowflake]*structs.StateRole),

		emojisMu: sync.RWMutex{},
		Emojis:   make(map[discord.Snowflake]*structs.StateEmoji),

		usersMu: sync.RWMutex{},
		Users:   make(map[discord.Snowflake]*structs.StateUser),
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

	err := GatewayDispatch(sh, msg)
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

func registerGatewayEvent(op discord.GatewayOp, handler func(sh *Shard, msg discord.GatewayPayload) (err error)) {
	gatewayHandlers[op] = handler
}

func registerDispatch(eventType string, handler func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error)) {
	dispatchHandlers[eventType] = handler
}

// GatewayDispatch handles selecting the proper gateway handler and executing it.
func GatewayDispatch(sh *Shard,
	event discord.GatewayPayload) (err error) {
	if f, ok := gatewayHandlers[event.Op]; ok {
		return f(sh, event)
	}

	return ErrNoGatewayHandler
}

func gatewayOpDispatch(sh *Shard, msg discord.GatewayPayload) (err error) {
	go func() {
		ticket := sh.Sandwich.EventPool.Wait()
		defer sh.Sandwich.EventPool.FreeTicket(ticket)

		err = sh.OnDispatch(msg)
		if err != nil && !xerrors.Is(err, ErrNoDispatchHandler) {
			sh.Logger.Error().Err(err).Msg("State dispatch failed")
		}
	}()

	return
}

func gatewayOpHeartbeat(sh *Shard, msg discord.GatewayPayload) (err error) {
	err = sh.SendEvent(discord.GatewayOpHeartbeat, sh.Sequence.Load())
	if err != nil {
		go sh.Sandwich.PublishSimpleWebhook(
			"Failed to send heartbeat",
			"`"+err.Error()+"`",
			fmt.Sprintf(
				"Manager: %s ShardGroup: %d ShardID: %d/%d",
				sh.Manager.Configuration.Identifier,
				sh.ShardGroup.ID,
				sh.ShardID,
				sh.ShardGroup.ShardCount,
			),
			EmbedColourDanger,
		)

		err = sh.Reconnect(websocket.StatusNormalClosure)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to reconnect")
		}
	}

	return
}

func gatewayOpReconnect(sh *Shard, msg discord.GatewayPayload) (err error) {
	sh.Logger.Info().Msg("Reconnecting in response to gateway")

	err = sh.Reconnect(WebsocketReconnectCloseCode)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to reconnect")
	}

	return
}

func gatewayOpInvalidSession(sh *Shard, msg discord.GatewayPayload) (err error) {
	resumable := json.Get(msg.Data, "d").ToBool()
	if !resumable {
		sh.SessionID.Store("")
		sh.Sequence.Store(0)
	}

	sh.Logger.Warn().Bool("resumable", resumable).Msg("Received invalid session")

	go sh.Sandwich.PublishSimpleWebhook(
		"Received invalid session from gateway",
		"",
		fmt.Sprintf(
			"Manager: %s ShardGroup: %d ShardID: %d/%d",
			sh.Manager.Configuration.Identifier,
			sh.ShardGroup.ID,
			sh.ShardID,
			sh.ShardGroup.ShardCount,
		),
		EmbedColourSandwich,
	)

	err = sh.Reconnect(WebsocketReconnectCloseCode)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to reconnect")
	}

	return
}

func gatewayOpHello(sh *Shard, msg discord.GatewayPayload) (err error) {
	hello := discord.Hello{}

	err = sh.decodeContent(msg, &hello)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	sh.LastHeartbeatSent.Store(now)
	sh.LastHeartbeatAck.Store(now)

	sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures
	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Msg("Received HELLO event from discord")

	return
}

func gatewayOpHeartbeatACK(sh *Shard, msg discord.GatewayPayload) (err error) {
	sh.LastHeartbeatAck.Store(time.Now().UTC())
	sh.Logger.Debug().
		Int64("RTT", sh.LastHeartbeatAck.Load().Sub(sh.LastHeartbeatSent.Load()).Milliseconds()).
		Msg("Received heartbeat ACK")

	return
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx *StateCtx,
	event discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		return f(ctx, event)
	}

	return result, false, ErrNoDispatchHandler
}

func init() {
	registerGatewayEvent(discord.GatewayOpDispatch, gatewayOpDispatch)
	registerGatewayEvent(discord.GatewayOpHeartbeat, gatewayOpHeartbeat)
	registerGatewayEvent(discord.GatewayOpReconnect, gatewayOpReconnect)
	registerGatewayEvent(discord.GatewayOpInvalidSession, gatewayOpInvalidSession)
	registerGatewayEvent(discord.GatewayOpHello, gatewayOpHello)
	registerGatewayEvent(discord.GatewayOpHeartbeatACK, gatewayOpHeartbeatACK)
}
