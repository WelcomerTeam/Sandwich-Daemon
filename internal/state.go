package internal

import (
	"sync"
	"time"

	snowflake "github.com/WelcomerTeam/RealRock/snowflake"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"golang.org/x/xerrors"
)

var NoGatewayHandler = xerrors.New("No registered handler for gateway event")
var NoDispatchHandler = xerrors.New("No registered handler for dispatch event")

// List of handlers for dispatch events.
var dispatchHandlers = make(map[string]func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error))

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(ctx *StateCtx, msg discord.GatewayPayload) (err error))

type StateCtx struct {
	Sg *Sandwich
	Mg *Manager
	Sh *Shard

	Vars map[string]interface{}
}

// SandwichState stores the collective state of all ShardGroups
// across all Managers.
type SandwichState struct {
	guildsMu sync.RWMutex
	Guilds   map[snowflake.ID]*discord.StateGuild

	guildMembersMu sync.RWMutex
	GuildMembers   map[snowflake.ID]*discord.StateGuildMembers

	channelsMu sync.RWMutex
	Channels   map[snowflake.ID]*discord.StateChannel

	rolesMu sync.RWMutex
	Roles   map[snowflake.ID]*discord.StateRole

	emojisMu sync.RWMutex
	Emojis   map[snowflake.ID]*discord.StateEmoji

	usersMu sync.RWMutex
	Users   map[snowflake.ID]*discord.StateUser
}

func NewSandwichState() (st *SandwichState) {
	st = &SandwichState{
		guildsMu: sync.RWMutex{},
		Guilds:   make(map[snowflake.ID]*discord.StateGuild),

		guildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[snowflake.ID]*discord.StateGuildMembers),

		channelsMu: sync.RWMutex{},
		Channels:   make(map[snowflake.ID]*discord.StateChannel),

		rolesMu: sync.RWMutex{},
		Roles:   make(map[snowflake.ID]*discord.StateRole),

		emojisMu: sync.RWMutex{},
		Emojis:   make(map[snowflake.ID]*discord.StateEmoji),

		usersMu: sync.RWMutex{},
		Users:   make(map[snowflake.ID]*discord.StateUser),
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

	return NoGatewayHandler
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx *StateCtx,
	event discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		return f(ctx, event)
	}

	return result, false, NoDispatchHandler
}
