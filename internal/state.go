package gateway

import (
	"sync"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	discord "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
	"golang.org/x/xerrors"
)

// TODO: Add some sort of ShardGroup parent counter for guilds so
// a guild is not deleted from cache if multiple ShardGroups are actively
// using it. When a shardgroup is made, it is incremented by 1 and when closed
// it is decremented by one. If after closing shardgroup it sees a guild with a
// count of 0 it will then call DeleteGuild then remove counter from the map.

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

func NewSandwichState() (st *SandwichState) {
	st = &SandwichState{
		GuildsMu: sync.RWMutex{},
		Guilds:   make(map[snowflake.ID]*discord.StateGuild),

		GuildMembersMu: sync.RWMutex{},
		GuildMembers:   make(map[snowflake.ID]*discord.StateGuildMembers),

		ChannelsMu: sync.RWMutex{},
		Channels:   make(map[snowflake.ID]*discord.Channel),

		RolesMu: sync.RWMutex{},
		Roles:   make(map[snowflake.ID]*discord.Role),

		EmojisMu: sync.RWMutex{},
		Emojis:   make(map[snowflake.ID]*discord.Emoji),

		UsersMu: sync.RWMutex{},
		Users:   make(map[snowflake.ID]*discord.User),
	}

	return st
}

func NewStateGuildMembers(g *discord.Guild) (gm *discord.StateGuildMembers) {
	gm = &discord.StateGuildMembers{
		GuildID:   g.ID,
		MembersMu: sync.RWMutex{},
		Members:   make(map[snowflake.ID]*discord.StateGuildMember),
	}

	return gm
}

func (st *SandwichState) GetGuildCount() (count int) {
	st.GuildsMu.RLock()
	count = len(st.Guilds)
	st.GuildsMu.RUnlock()

	return
}

func (sg *ShardGroup) GetGuildCount() (count int) {
	sg.GuildsMu.RLock()
	count = len(sg.Guilds)
	sg.GuildsMu.RUnlock()

	return
}

// Guild State

func (st *SandwichState) AddGuild(ctx *StateCtx, g *discord.Guild) (sg *discord.StateGuild) {
	sg = &discord.StateGuild{}

	for _, r := range g.Roles {
		st.AddRole(ctx, r)
		sg.RoleIDs = append(sg.RoleIDs, r.ID)
	}

	for _, c := range g.Channels {
		st.AddChannel(ctx, c)
		sg.ChannelIDs = append(sg.ChannelIDs, c.ID)
	}

	for _, e := range g.Emojis {
		st.AddEmoji(ctx, e)
		sg.EmojiIDs = append(sg.EmojiIDs, e.ID)
	}

	sg.Guild = g
	sg.Roles = make([]*discord.Role, 0, len(sg.RoleIDs))
	sg.Channels = make([]*discord.Channel, 0, len(sg.ChannelIDs))
	sg.Emojis = make([]*discord.Emoji, 0, len(sg.EmojiIDs))

	st.GuildsMu.Lock()
	st.Guilds[g.ID] = sg
	st.GuildsMu.Unlock()

	return
}

func (st *SandwichState) GetGuild(ctx *StateCtx, s snowflake.ID, expand bool) (g *discord.Guild, o bool) {
	st.GuildsMu.RLock()
	sg, o := st.Guilds[s]
	st.GuildsMu.RUnlock()

	if !o {
		return
	}

	g = sg.Guild

	// If expand is True, it will populate the Role, Channel and Emoji
	// slices from the State.
	if expand {
		for _, ri := range sg.RoleIDs {
			if r, ok := st.GetRole(ctx, ri); ok {
				sg.Roles = append(sg.Roles, r)
			} else {
				ctx.Sh.Logger.Warn().Msgf("GetGuild referenced role ID %d that was not in state", ri)
			}
		}

		for _, ci := range sg.ChannelIDs {
			if c, ok := st.GetChannel(ctx, ci); ok {
				sg.Channels = append(sg.Channels, c)
			} else {
				ctx.Sh.Logger.Warn().Msgf("GetGuild referenced channel ID %d that was not in state", ci)
			}
		}

		for _, ei := range sg.EmojiIDs {
			if e, ok := st.GetEmoji(ctx, ei); ok {
				sg.Emojis = append(sg.Emojis, e)
			} else {
				ctx.Sh.Logger.Warn().Msgf("GetGuild referenced emoji ID %d that was not in state", ei)
			}
		}
	}

	return g, o
}

func (st *SandwichState) RemoveGuild(ctx *StateCtx, s snowflake.ID) {
	st.GuildsMu.Lock()
	defer st.GuildsMu.Unlock()

	sg, o := st.Guilds[s]
	if !o {
		return
	}

	for _, ri := range sg.RoleIDs {
		st.RemoveRole(ctx, ri)
	}

	for _, ci := range sg.ChannelIDs {
		st.RemoveChannel(ctx, ci)
	}

	for _, ei := range sg.EmojiIDs {
		st.RemoveEmoji(ctx, ei)
	}

	delete(st.Guilds, s)
}

// Guild State Shardgroup Specific

func (st *SandwichState) AddGuildShardGroup(ctx *StateCtx, g *discord.Guild) {
	gs := st.AddGuild(ctx, g)

	ctx.Sh.ShardGroup.GuildsMu.Lock()
	ctx.Sh.ShardGroup.Guilds[g.ID] = gs
	ctx.Sh.ShardGroup.GuildsMu.Unlock()
}

func (st *SandwichState) RemoveGuildShardGroup(ctx *StateCtx, s snowflake.ID) {
	ctx.Sh.ShardGroup.GuildsMu.Lock()
	delete(ctx.Sh.ShardGroup.Guilds, s)
	ctx.Sh.ShardGroup.GuildsMu.Unlock()

	st.RemoveGuild(ctx, s)
}

// Member State

// AddMembers creates a StateGuildMember object if a guild does not have it,
// It also adds the User to the cache if it does not already exist.
func (st *SandwichState) AddMember(ctx *StateCtx, g *discord.Guild, m *discord.GuildMember) {
	st.GuildMembersMu.RLock()
	members, ok := st.GuildMembers[g.ID]
	st.GuildMembersMu.RUnlock()

	if !ok {
		ctx.Sg.Logger.Trace().
			Msgf("Created new GuildMembers entry for guild ID %d", g.ID)

		members = NewStateGuildMembers(g)

		st.GuildMembersMu.Lock()
		st.GuildMembers[g.ID] = members
		st.GuildMembersMu.Unlock()
	}

	st.AddUser(ctx, m.User)

	members.MembersMu.Lock()
	members.Members[m.User.ID] = discord.FromGuildMember(m)
	members.MembersMu.Unlock()
}

func (st *SandwichState) GetMember(ctx *StateCtx, g *discord.Guild, s snowflake.ID) (m *discord.GuildMember, o bool) {
	st.GuildMembersMu.RLock()
	gm, o := st.GuildMembers[g.ID]
	st.GuildMembersMu.RUnlock()

	if !o {
		return
	}

	gm.MembersMu.RLock()
	sgm, o := gm.Members[s]
	gm.MembersMu.RUnlock()

	if !o {
		return
	}

	u, o := st.GetUser(ctx, sgm.User)
	if !o {
		ctx.Sh.Logger.Warn().Msgf("GetMessage referenced user ID %d that was not in state", sgm.User)
	}

	return sgm.ToGuildMember(u), true
}

func (st *SandwichState) RemoveMember(ctx *StateCtx, g *discord.Guild, s snowflake.ID) {
	st.GuildMembersMu.RUnlock()
	gm, o := st.GuildMembers[g.ID]
	st.GuildMembersMu.RUnlock()

	if !o {
		return
	}

	gm.MembersMu.Lock()
	delete(gm.Members, s)
	gm.MembersMu.Unlock()

	return
}

// Channel State

func (st *SandwichState) AddChannel(ctx *StateCtx, c *discord.Channel) {
	st.ChannelsMu.Lock()
	st.Channels[c.ID] = c
	st.ChannelsMu.Unlock()
}

func (st *SandwichState) GetChannel(ctx *StateCtx, s snowflake.ID) (c *discord.Channel, o bool) {
	st.ChannelsMu.RLock()
	c, o = st.Channels[s]
	st.ChannelsMu.RUnlock()

	if !o {
		c.ID = s
	}

	return
}

func (st *SandwichState) RemoveChannel(ctx *StateCtx, s snowflake.ID) {
	st.ChannelsMu.Lock()
	delete(st.Channels, s)
	st.ChannelsMu.Unlock()
}

// Role State

func (st *SandwichState) AddRole(ctx *StateCtx, r *discord.Role) {
	st.RolesMu.Lock()
	st.Roles[r.ID] = r
	st.RolesMu.Unlock()
}

func (st *SandwichState) GetRole(ctx *StateCtx, s snowflake.ID) (r *discord.Role, o bool) {
	st.RolesMu.RLock()
	r, o = st.Roles[s]
	st.RolesMu.RUnlock()

	if !o {
		r.ID = s
	}

	return
}

func (st *SandwichState) RemoveRole(ctx *StateCtx, s snowflake.ID) {
	st.RolesMu.Lock()
	delete(st.Roles, s)
	st.RolesMu.Unlock()
}

// Emoji State

func (st *SandwichState) AddEmoji(ctx *StateCtx, e *discord.Emoji) {
	st.EmojisMu.Lock()
	st.Emojis[e.ID] = e
	st.EmojisMu.Unlock()
}

func (st *SandwichState) GetEmoji(ctx *StateCtx, s snowflake.ID) (e *discord.Emoji, o bool) {
	st.EmojisMu.RLock()
	e, o = st.Emojis[s]
	st.EmojisMu.RUnlock()

	if !o {
		e.ID = s
	}

	return
}

func (st *SandwichState) RemoveEmoji(ctx *StateCtx, s snowflake.ID) {
	st.EmojisMu.Lock()
	delete(st.Emojis, s)
	st.EmojisMu.Unlock()
}

// User state

func (st *SandwichState) AddUser(ctx *StateCtx, u *discord.User) {
	st.UsersMu.Lock()
	st.Users[u.ID] = u
	st.UsersMu.Unlock()
}

func (st *SandwichState) GetUser(ctx *StateCtx, s snowflake.ID) (u *discord.User, o bool) {
	st.UsersMu.RLock()
	u, o = st.Users[s]
	st.UsersMu.RUnlock()

	if !o {
		u.ID = s
	}

	return
}

func (st *SandwichState) RemoveUser(ctx *StateCtx, s snowflake.ID) {
	st.UsersMu.Lock()
	delete(st.Users, s)
	st.UsersMu.Unlock()
}

func init() {
	registerState("READY", StateReady)
	registerState("GUILD_CREATE", StateGuildCreate)
	registerState("GUILD_MEMBERS_CHUNK", StateGuildMembersChunk)
}
