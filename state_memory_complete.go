package sandwich

import (
	"context"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

// StateProviderMemory is a state provider that stores all data in memory.

type StateProviderMemory struct {
	guilds        *csmap.CsMap[discord.Snowflake, discord.Guild]
	guildMembers  *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.GuildMember]]
	guildChannels *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.Channel]]
	guildRoles    *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.Role]]
	guildEmojis   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.Emoji]]
	voiceStates   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.VoiceState]]
	users         *csmap.CsMap[discord.Snowflake, discord.User]
	userMutuals   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, bool]]
}

func NewStateProviderMemory() *StateProviderMemory {
	return &StateProviderMemory{
		guilds:        csmap.Create[discord.Snowflake, discord.Guild](),
		guildMembers:  csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.GuildMember]](),
		guildChannels: csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.Channel]](),
		guildRoles:    csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.Role]](),
		guildEmojis:   csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.Emoji]](),
		voiceStates:   csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, discord.VoiceState]](),
		users:         csmap.Create[discord.Snowflake, discord.User](),
		userMutuals:   csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, bool]](),
	}
}

func (s *StateProviderMemory) GetGuild(_ context.Context, guildID discord.Snowflake) (discord.Guild, bool) {
	guild, guildExists := s.guilds.Load(guildID)

	if guildChannels, exists := s.guildChannels.Load(guildID); exists {
		guildChannels.Range(func(_ discord.Snowflake, value discord.Channel) bool {
			guild.Channels = append(guild.Channels, value)

			return true
		})
	}

	if guildRoles, exists := s.guildRoles.Load(guildID); exists {
		guildRoles.Range(func(_ discord.Snowflake, value discord.Role) bool {
			guild.Roles = append(guild.Roles, value)

			return true
		})
	}

	if guildEmojis, exists := s.guildEmojis.Load(guildID); exists {
		guildEmojis.Range(func(_ discord.Snowflake, value discord.Emoji) bool {
			guild.Emojis = append(guild.Emojis, value)

			return true
		})
	}

	return guild, guildExists
}

func (s *StateProviderMemory) SetGuild(ctx context.Context, guildID discord.Snowflake, guild discord.Guild) {
	s.SetGuildMembers(ctx, guildID, guild.Members)
	clear(guild.Members)

	s.SetGuildChannels(ctx, guildID, guild.Channels)
	clear(guild.Channels)

	s.SetGuildRoles(ctx, guildID, guild.Roles)
	clear(guild.Roles)

	s.SetGuildEmojis(ctx, guildID, guild.Emojis)
	clear(guild.Emojis)

	s.guilds.Store(guildID, guild)
}

func (s *StateProviderMemory) GetGuildMembers(_ context.Context, guildID discord.Snowflake) ([]discord.GuildMember, bool) {
	guildMembersState, exists := s.guildMembers.Load(guildID)
	if !exists {
		return nil, false
	}

	var guildMembers []discord.GuildMember

	guildMembersState.Range(func(_ discord.Snowflake, value discord.GuildMember) bool {
		guildMembers = append(guildMembers, value)

		return true
	})

	return guildMembers, exists
}

func (s *StateProviderMemory) SetGuildMembers(_ context.Context, guildID discord.Snowflake, guildMembers []discord.GuildMember) {
	guildMembersState, ok := s.guildMembers.Load(guildID)
	if !ok {
		guildMembersState = &syncmap.Map[discord.Snowflake, discord.GuildMember]{}

		s.guildMembers.Store(guildID, guildMembersState)
	}

	for _, member := range guildMembers {
		guildMembersState.Store(member.User.ID, member)
	}
}

func (s *StateProviderMemory) GetGuildMember(_ context.Context, guildID, userID discord.Snowflake) (discord.GuildMember, bool) {
	members, ok := s.guildMembers.Load(guildID)
	if !ok {
		return discord.GuildMember{}, false
	}

	return members.Load(userID)
}

func (s *StateProviderMemory) SetGuildMember(_ context.Context, guildID discord.Snowflake, member discord.GuildMember) {
	guildMembersState, ok := s.guildMembers.Load(guildID)
	if !ok {
		guildMembersState = &syncmap.Map[discord.Snowflake, discord.GuildMember]{}

		s.guildMembers.Store(guildID, guildMembersState)
	}

	guildMembersState.Store(member.User.ID, member)
}

func (s *StateProviderMemory) RemoveGuildMember(_ context.Context, guildID, userID discord.Snowflake) {
	guildMembersState, ok := s.guildMembers.Load(guildID)
	if !ok {
		return
	}

	guildMembersState.Delete(userID)
}

func (s *StateProviderMemory) GetGuildChannels(_ context.Context, guildID discord.Snowflake) ([]discord.Channel, bool) {
	guildChannelsState, ok := s.guildChannels.Load(guildID)
	if !ok {
		return nil, false
	}

	var guildChannels []discord.Channel

	guildChannelsState.Range(func(_ discord.Snowflake, value discord.Channel) bool {
		guildChannels = append(guildChannels, value)

		return true
	})

	return guildChannels, true
}

func (s *StateProviderMemory) SetGuildChannels(_ context.Context, guildID discord.Snowflake, channels []discord.Channel) {
	guildChannelsState, ok := s.guildChannels.Load(guildID)
	if !ok {
		guildChannelsState = &syncmap.Map[discord.Snowflake, discord.Channel]{}

		s.guildChannels.Store(guildID, guildChannelsState)
	}

	for _, channel := range channels {
		guildChannelsState.Store(channel.ID, channel)
	}
}

func (s *StateProviderMemory) GetGuildChannel(_ context.Context, guildID, channelID discord.Snowflake) (discord.Channel, bool) {
	guildChannelsState, ok := s.guildChannels.Load(guildID)
	if !ok {
		return discord.Channel{}, false
	}

	return guildChannelsState.Load(channelID)
}

func (s *StateProviderMemory) SetGuildChannel(_ context.Context, guildID discord.Snowflake, channel discord.Channel) {
	guildChannelsState, ok := s.guildChannels.Load(guildID)
	if !ok {
		guildChannelsState = &syncmap.Map[discord.Snowflake, discord.Channel]{}

		s.guildChannels.Store(guildID, guildChannelsState)
	}

	guildChannelsState.Store(channel.ID, channel)
}

func (s *StateProviderMemory) RemoveGuildChannel(_ context.Context, guildID, channelID discord.Snowflake) {
	guildChannelsState, ok := s.guildChannels.Load(guildID)
	if !ok {
		return
	}

	guildChannelsState.Delete(channelID)
}

func (s *StateProviderMemory) GetGuildRoles(_ context.Context, guildID discord.Snowflake) ([]discord.Role, bool) {
	guildRolesState, ok := s.guildRoles.Load(guildID)
	if !ok {
		return nil, false
	}

	var guildRoles []discord.Role

	guildRolesState.Range(func(_ discord.Snowflake, value discord.Role) bool {
		guildRoles = append(guildRoles, value)

		return true
	})

	return guildRoles, true
}

func (s *StateProviderMemory) SetGuildRoles(_ context.Context, guildID discord.Snowflake, roles []discord.Role) {
	guildRolesState, ok := s.guildRoles.Load(guildID)
	if !ok {
		guildRolesState = &syncmap.Map[discord.Snowflake, discord.Role]{}

		s.guildRoles.Store(guildID, guildRolesState)
	}

	for _, role := range roles {
		guildRolesState.Store(role.ID, role)
	}
}

func (s *StateProviderMemory) GetGuildRole(_ context.Context, guildID, roleID discord.Snowflake) (discord.Role, bool) {
	guildRolesState, ok := s.guildRoles.Load(guildID)
	if !ok {
		return discord.Role{}, false
	}

	return guildRolesState.Load(roleID)
}

func (s *StateProviderMemory) SetGuildRole(_ context.Context, guildID discord.Snowflake, role discord.Role) {
	guildRolesState, ok := s.guildRoles.Load(guildID)
	if !ok {
		guildRolesState = &syncmap.Map[discord.Snowflake, discord.Role]{}

		s.guildRoles.Store(guildID, guildRolesState)
	}

	guildRolesState.Store(role.ID, role)
}

func (s *StateProviderMemory) RemoveGuildRole(_ context.Context, guildID, roleID discord.Snowflake) {
	guildRolesState, ok := s.guildRoles.Load(guildID)
	if !ok {
		return
	}

	guildRolesState.Delete(roleID)
}

func (s *StateProviderMemory) GetGuildEmojis(_ context.Context, guildID discord.Snowflake) ([]discord.Emoji, bool) {
	guildEmojisState, ok := s.guildEmojis.Load(guildID)
	if !ok {
		return nil, false
	}

	var guildEmojis []discord.Emoji

	guildEmojisState.Range(func(_ discord.Snowflake, value discord.Emoji) bool {
		guildEmojis = append(guildEmojis, value)

		return true
	})

	return guildEmojis, true
}

func (s *StateProviderMemory) SetGuildEmojis(_ context.Context, guildID discord.Snowflake, emojis []discord.Emoji) {
	guildEmojisState, ok := s.guildEmojis.Load(guildID)
	if !ok {
		guildEmojisState = &syncmap.Map[discord.Snowflake, discord.Emoji]{}

		s.guildEmojis.Store(guildID, guildEmojisState)
	}

	for _, emoji := range emojis {
		guildEmojisState.Store(emoji.ID, emoji)
	}
}

func (s *StateProviderMemory) GetGuildEmoji(_ context.Context, guildID, emojiID discord.Snowflake) (discord.Emoji, bool) {
	guildEmojisState, ok := s.guildEmojis.Load(guildID)
	if !ok {
		return discord.Emoji{}, false
	}

	return guildEmojisState.Load(emojiID)
}

func (s *StateProviderMemory) SetGuildEmoji(_ context.Context, guildID discord.Snowflake, emoji discord.Emoji) {
	guildEmojisState, ok := s.guildEmojis.Load(guildID)
	if !ok {
		guildEmojisState = &syncmap.Map[discord.Snowflake, discord.Emoji]{}

		s.guildEmojis.Store(guildID, guildEmojisState)
	}

	guildEmojisState.Store(emoji.ID, emoji)
}

func (s *StateProviderMemory) RemoveGuildEmoji(_ context.Context, guildID, emojiID discord.Snowflake) {
	guildEmojisState, ok := s.guildEmojis.Load(guildID)
	if !ok {
		return
	}

	guildEmojisState.Delete(emojiID)
}

func (s *StateProviderMemory) GetVoiceStates(_ context.Context, guildID discord.Snowflake) ([]discord.VoiceState, bool) {
	voiceStatesState, ok := s.voiceStates.Load(guildID)
	if !ok {
		return nil, false
	}

	var voiceStates []discord.VoiceState

	voiceStatesState.Range(func(_ discord.Snowflake, value discord.VoiceState) bool {
		voiceStates = append(voiceStates, value)

		return true
	})

	return voiceStates, true
}

func (s *StateProviderMemory) GetVoiceState(_ context.Context, guildID, userID discord.Snowflake) (discord.VoiceState, bool) {
	voiceStatesState, ok := s.voiceStates.Load(guildID)
	if !ok {
		return discord.VoiceState{}, false
	}

	return voiceStatesState.Load(userID)
}

func (s *StateProviderMemory) SetVoiceState(_ context.Context, guildID discord.Snowflake, voiceState discord.VoiceState) {
	voiceStatesState, ok := s.voiceStates.Load(guildID)
	if !ok {
		voiceStatesState = &syncmap.Map[discord.Snowflake, discord.VoiceState]{}

		s.voiceStates.Store(guildID, voiceStatesState)
	}

	voiceStatesState.Store(voiceState.UserID, voiceState)
}

func (s *StateProviderMemory) RemoveVoiceState(_ context.Context, guildID, userID discord.Snowflake) {
	voiceStatesState, ok := s.voiceStates.Load(guildID)
	if !ok {
		return
	}

	voiceStatesState.Delete(userID)
}

func (s *StateProviderMemory) GetUser(_ context.Context, userID discord.Snowflake) (discord.User, bool) {
	user, ok := s.users.Load(userID)

	return user, ok
}

func (s *StateProviderMemory) SetUser(_ context.Context, userID discord.Snowflake, user discord.User) {
	s.users.Store(userID, user)
}

func (s *StateProviderMemory) GetUserMutualGuilds(_ context.Context, userID discord.Snowflake) ([]discord.Snowflake, bool) {
	userMutualsState, ok := s.userMutuals.Load(userID)
	if !ok {
		return nil, false
	}

	var userMutuals []discord.Snowflake

	userMutualsState.Range(func(key discord.Snowflake, _ bool) bool {
		userMutuals = append(userMutuals, key)

		return true
	})

	return userMutuals, true
}

func (s *StateProviderMemory) AddUserMutualGuild(_ context.Context, userID, guildID discord.Snowflake) {
	userMutualsState, ok := s.userMutuals.Load(userID)
	if !ok {
		userMutualsState = &syncmap.Map[discord.Snowflake, bool]{}

		s.userMutuals.Store(userID, userMutualsState)
	}

	userMutualsState.Store(guildID, true)
}

func (s *StateProviderMemory) RemoveUserMutualGuild(_ context.Context, userID, guildID discord.Snowflake) {
	userMutualsState, ok := s.userMutuals.Load(userID)
	if !ok {
		return
	}

	userMutualsState.Delete(guildID)
}
