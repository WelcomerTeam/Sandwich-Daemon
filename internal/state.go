package internal

import (
	"sync"

	snowflake "github.com/WelcomerTeam/RealRock/snowflake"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"golang.org/x/xerrors"
)

var NoEventHandler = xerrors.New("No registered handler for event")

var stateHandlers = make(map[string]func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error))

type StateCtx struct {
	Sg *Sandwich
	Mg *Manager
	Sh *Shard

	Vars map[string]interface{}
}

// SandwichState stores the collective state of all ShardGroups
// accross all Managers.
type SandwichState struct {
	GuildsMu sync.RWMutex
	Guilds   map[snowflake.ID]*discord.StateGuild

	GuildMembersMu sync.RWMutex
	GuildMembers   map[snowflake.ID]*discord.StateGuildMembers

	ChannelsMu sync.RWMutex
	Channels   map[snowflake.ID]*discord.StateChannel

	RolesMu sync.RWMutex
	Roles   map[snowflake.ID]*discord.StateRole

	EmojisMu sync.RWMutex
	Emojis   map[snowflake.ID]*discord.StateEmoji

	UsersMu sync.RWMutex
	Users   map[snowflake.ID]*discord.StateUser
}

func NewSandwichState() (st *SandwichState) {
	st = &SandwichState{
		GuildsMu: sync.RWMutex{},
		Guilds:   make(map[snowflake.ID]*discord.StateGuild),

		GuildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[snowflake.ID]*discord.StateGuildMembers),

		ChannelsMu: sync.RWMutex{},
		Channels:   make(map[snowflake.ID]*discord.StateChannel),

		RolesMu: sync.RWMutex{},
		Roles:   make(map[snowflake.ID]*discord.StateRole),

		EmojisMu: sync.RWMutex{},
		Emojis:   make(map[snowflake.ID]*discord.StateEmoji),

		UsersMu: sync.RWMutex{},
		Users:   make(map[snowflake.ID]*discord.StateUser),
	}

	return st
}

func registerState(eventType string, handler func(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error)) {
	stateHandlers[eventType] = handler
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx *StateCtx,
	event discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	if f, ok := stateHandlers[event.Type]; ok {
		return f(ctx, event)
	}

	return result, false, NoEventHandler
}
