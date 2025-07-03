package sandwich

import (
	"context"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/WelcomerTeam/Sandwich-Daemon/pkg/syncmap"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

// StateProviderMemoryOptimized is a state provider that stores all data in memory.
// This uses memory aligned data structures to store the data and loses some fields.

type StateProviderMemoryOptimized struct {
	Guilds        *csmap.CsMap[discord.Snowflake, StateGuild]
	GuildMembers  *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateGuildMember]]
	GuildChannels *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateChannel]]
	GuildRoles    *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateRole]]
	GuildEmojis   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateEmoji]]
	VoiceStates   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateVoiceState]]
	GuildStickers *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateSticker]]
	Users         *csmap.CsMap[discord.Snowflake, StateUser]
	UserMutuals   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, bool]]
}

func NewStateProviderMemoryOptimized() *StateProviderMemoryOptimized {
	stateProvider := &StateProviderMemoryOptimized{
		Guilds: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, StateGuild](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		GuildMembers: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateGuildMember]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		GuildChannels: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateChannel]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		GuildRoles: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateRole]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		GuildEmojis: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateEmoji]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		VoiceStates: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateVoiceState]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		GuildStickers: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateSticker]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		Users: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, StateUser](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
		UserMutuals: csmap.Create(
			csmap.WithCustomHasher[discord.Snowflake, *syncmap.Map[discord.Snowflake, bool]](func(key discord.Snowflake) uint64 {
				return uint64(key)
			}),
		),
	}

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()

		for range t.C {
			stateProvider.UpdateStateMetricsFromStateProvider()
		}
	}()

	return stateProvider
}

func (s *StateProviderMemoryOptimized) UpdateStateMetricsFromStateProvider() {
	guildChannelsCount := 0
	s.GuildChannels.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, StateChannel]) bool {
		guildChannelsCount += value.Count()

		return false
	})

	guildRolesCount := 0

	s.GuildRoles.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, StateRole]) bool {
		guildRolesCount += value.Count()

		return false
	})

	guildEmojisCount := 0
	s.GuildEmojis.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, StateEmoji]) bool {
		guildEmojisCount += value.Count()

		return false
	})

	guildMembersCount := 0
	s.GuildMembers.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, StateGuildMember]) bool {
		guildMembersCount += value.Count()

		return false
	})

	userMutualsCount := 0
	s.UserMutuals.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, bool]) bool {
		userMutualsCount += value.Count()

		return false
	})

	stickersCount := 0
	s.GuildStickers.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, StateSticker]) bool {
		stickersCount += value.Count()

		return false
	})

	voiceStatesCount := 0
	s.VoiceStates.Range(func(_ discord.Snowflake, value *syncmap.Map[discord.Snowflake, StateVoiceState]) bool {
		voiceStatesCount += value.Count()

		return false
	})

	usersCount := s.Users.Count()
	guildsCount := s.Guilds.Count()

	UpdateStateMetrics(
		guildMembersCount,
		guildRolesCount,
		guildEmojisCount,
		usersCount,
		guildChannelsCount,
		stickersCount,
		guildsCount,
		voiceStatesCount,
	)
}

func (s *StateProviderMemoryOptimized) GetGuilds(_ context.Context) ([]*discord.Guild, bool) {
	RecordStateRequest()
	guilds := make([]*discord.Guild, 0, s.Guilds.Count())

	s.Guilds.Range(func(_ discord.Snowflake, value StateGuild) bool {
		guilds = append(guilds, s.fillGuild(value))

		return true
	})

	RecordStateHitWithValue(float64(len(guilds)))
	return guilds, true
}

func (s *StateProviderMemoryOptimized) fillGuild(guildState StateGuild) *discord.Guild {
	guild := StateGuildToDiscord(guildState)

	if guildChannels, exists := s.GuildChannels.Load(guildState.ID); exists {
		guildChannels.Range(func(_ discord.Snowflake, value StateChannel) bool {
			guild.Channels = append(guild.Channels, StateChannelToDiscord(value))

			return true
		})
	}

	if guildRoles, exists := s.GuildRoles.Load(guildState.ID); exists {
		guildRoles.Range(func(_ discord.Snowflake, value StateRole) bool {
			guild.Roles = append(guild.Roles, StateRoleToDiscord(value))

			return true
		})
	}

	if guildEmojis, exists := s.GuildEmojis.Load(guildState.ID); exists {
		guildEmojis.Range(func(_ discord.Snowflake, value StateEmoji) bool {
			guild.Emojis = append(guild.Emojis, StateEmojiToDiscord(value))

			return true
		})
	}

	if guildStickers, exists := s.GuildStickers.Load(guildState.ID); exists {
		guildStickers.Range(func(_ discord.Snowflake, value StateSticker) bool {
			guild.Stickers = append(guild.Stickers, StateStickerToDiscord(value))

			return true
		})
	}

	return &guild
}

func (s *StateProviderMemoryOptimized) GetGuild(_ context.Context, guildID discord.Snowflake) (*discord.Guild, bool) {
	RecordStateRequest()
	guildState, guildExists := s.Guilds.Load(guildID)
	if !guildExists {
		RecordStateMiss()

		return nil, false
	}

	RecordStateHit()
	return s.fillGuild(guildState), true
}

func (s *StateProviderMemoryOptimized) SetGuild(ctx context.Context, guildID discord.Snowflake, guild discord.Guild) {
	if len(guild.Members) > 0 {
		s.SetGuildMembers(ctx, guildID, guild.Members)
	}

	if len(guild.Channels) > 0 {
		s.SetGuildChannels(ctx, guildID, guild.Channels)
	}

	if len(guild.Roles) > 0 {
		s.SetGuildRoles(ctx, guildID, guild.Roles)
	}

	if len(guild.Emojis) > 0 {
		s.SetGuildEmojis(ctx, guildID, guild.Emojis)
	}

	if len(guild.Stickers) > 0 {
		s.SetGuildStickers(ctx, guildID, guild.Stickers)
	}

	s.Guilds.Store(guildID, DiscordToStateGuild(guild))
}

func (s *StateProviderMemoryOptimized) GetGuildMembers(_ context.Context, guildID discord.Snowflake) ([]*discord.GuildMember, bool) {
	RecordStateRequest()
	guildMembersState, exists := s.GuildMembers.Load(guildID)
	if !exists {
		RecordStateMiss()

		return nil, false
	}

	var guildMembers []*discord.GuildMember

	guildMembersState.Range(func(_ discord.Snowflake, value StateGuildMember) bool {
		guildMember := StateGuildMemberToDiscord(value)
		guildMembers = append(guildMembers, &guildMember)

		return true
	})

	RecordStateHitWithValue(float64(len(guildMembers)))
	return guildMembers, true
}

func (s *StateProviderMemoryOptimized) SetGuildMembers(ctx context.Context, guildID discord.Snowflake, guildMembers []discord.GuildMember) {
	guildMembersState, ok := s.GuildMembers.Load(guildID)
	if !ok {
		guildMembersState = &syncmap.Map[discord.Snowflake, StateGuildMember]{}

		s.GuildMembers.Store(guildID, guildMembersState)
	}

	for _, member := range guildMembers {
		guildMembersState.Store(member.User.ID, DiscordToStateGuildMember(member))

		if member.User != nil {
			s.SetUser(ctx, member.User.ID, *member.User)
		}
	}
}

func (s *StateProviderMemoryOptimized) GetGuildMember(_ context.Context, guildID, userID discord.Snowflake) (*discord.GuildMember, bool) {
	RecordStateRequest()
	members, ok := s.GuildMembers.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	member, ok := members.Load(userID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	guildMember := StateGuildMemberToDiscord(member)
	RecordStateHit()
	return &guildMember, true
}

func (s *StateProviderMemoryOptimized) SetGuildMember(ctx context.Context, guildID discord.Snowflake, member discord.GuildMember) {
	guildMembersState, ok := s.GuildMembers.Load(guildID)
	if !ok {
		guildMembersState = &syncmap.Map[discord.Snowflake, StateGuildMember]{}

		s.GuildMembers.Store(guildID, guildMembersState)
	}

	guildMembersState.Store(member.User.ID, DiscordToStateGuildMember(member))

	if member.User != nil {
		s.SetUser(ctx, member.User.ID, *member.User)
	}
}

func (s *StateProviderMemoryOptimized) RemoveGuildMember(_ context.Context, guildID, userID discord.Snowflake) {
	guildMembersState, ok := s.GuildMembers.Load(guildID)
	if !ok {
		return
	}

	guildMembersState.Delete(userID)
}

func (s *StateProviderMemoryOptimized) GetGuildChannels(_ context.Context, guildID discord.Snowflake) ([]*discord.Channel, bool) {
	RecordStateRequest()
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	var guildChannels []*discord.Channel

	guildChannelsState.Range(func(_ discord.Snowflake, value StateChannel) bool {
		guildChannel := StateChannelToDiscord(value)
		guildChannels = append(guildChannels, &guildChannel)

		return true
	})

	RecordStateHitWithValue(float64(len(guildChannels)))
	return guildChannels, true
}

func (s *StateProviderMemoryOptimized) SetGuildChannels(_ context.Context, guildID discord.Snowflake, channels []discord.Channel) {
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		guildChannelsState = &syncmap.Map[discord.Snowflake, StateChannel]{}

		s.GuildChannels.Store(guildID, guildChannelsState)
	}

	for _, channel := range channels {
		guildChannelsState.Store(channel.ID, DiscordToStateChannel(channel))
	}
}

func (s *StateProviderMemoryOptimized) GetGuildChannel(_ context.Context, guildID, channelID discord.Snowflake) (*discord.Channel, bool) {
	RecordStateRequest()
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	channelState, ok := guildChannelsState.Load(channelID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	guildChannel := StateChannelToDiscord(channelState)
	RecordStateHit()
	return &guildChannel, true
}

func (s *StateProviderMemoryOptimized) SetGuildChannel(_ context.Context, guildID discord.Snowflake, channel discord.Channel) {
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		guildChannelsState = &syncmap.Map[discord.Snowflake, StateChannel]{}

		s.GuildChannels.Store(guildID, guildChannelsState)
	}

	guildChannelsState.Store(channel.ID, DiscordToStateChannel(channel))
}

func (s *StateProviderMemoryOptimized) RemoveGuildChannel(_ context.Context, guildID, channelID discord.Snowflake) {
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		return
	}

	guildChannelsState.Delete(channelID)
}

func (s *StateProviderMemoryOptimized) GetGuildRoles(_ context.Context, guildID discord.Snowflake) ([]*discord.Role, bool) {
	RecordStateRequest()
	guildRolesState, ok := s.GuildRoles.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	var guildRoles []*discord.Role

	guildRolesState.Range(func(_ discord.Snowflake, value StateRole) bool {
		guildRole := StateRoleToDiscord(value)
		guildRoles = append(guildRoles, &guildRole)

		return true
	})

	RecordStateHit()
	return guildRoles, true
}

func (s *StateProviderMemoryOptimized) SetGuildRoles(_ context.Context, guildID discord.Snowflake, roles []discord.Role) {
	guildRolesState, ok := s.GuildRoles.Load(guildID)
	if !ok {
		guildRolesState = &syncmap.Map[discord.Snowflake, StateRole]{}

		s.GuildRoles.Store(guildID, guildRolesState)
	}

	for _, role := range roles {
		guildRolesState.Store(role.ID, DiscordToStateRole(role))
	}
}

func (s *StateProviderMemoryOptimized) GetGuildRole(_ context.Context, guildID, roleID discord.Snowflake) (*discord.Role, bool) {
	RecordStateRequest()
	guildRolesState, ok := s.GuildRoles.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	role, ok := guildRolesState.Load(roleID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	guildRole := StateRoleToDiscord(role)
	RecordStateHit()
	return &guildRole, true
}

func (s *StateProviderMemoryOptimized) SetGuildRole(_ context.Context, guildID discord.Snowflake, role discord.Role) {
	guildRolesState, ok := s.GuildRoles.Load(guildID)
	if !ok {
		guildRolesState = &syncmap.Map[discord.Snowflake, StateRole]{}

		s.GuildRoles.Store(guildID, guildRolesState)
	}

	guildRolesState.Store(role.ID, DiscordToStateRole(role))
}

func (s *StateProviderMemoryOptimized) RemoveGuildRole(_ context.Context, guildID, roleID discord.Snowflake) {
	guildRolesState, ok := s.GuildRoles.Load(guildID)
	if !ok {
		return
	}

	guildRolesState.Delete(roleID)
}

func (s *StateProviderMemoryOptimized) GetGuildEmojis(_ context.Context, guildID discord.Snowflake) ([]*discord.Emoji, bool) {
	RecordStateRequest()
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	var guildEmojis []*discord.Emoji

	guildEmojisState.Range(func(_ discord.Snowflake, value StateEmoji) bool {
		guildEmoji := StateEmojiToDiscord(value)
		guildEmojis = append(guildEmojis, &guildEmoji)

		return true
	})

	RecordStateHitWithValue(float64(len(guildEmojis)))
	return guildEmojis, true
}

func (s *StateProviderMemoryOptimized) SetGuildEmojis(_ context.Context, guildID discord.Snowflake, emojis []discord.Emoji) {
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		guildEmojisState = &syncmap.Map[discord.Snowflake, StateEmoji]{}

		s.GuildEmojis.Store(guildID, guildEmojisState)
	}

	for _, emoji := range emojis {
		guildEmojisState.Store(emoji.ID, DiscordToStateEmoji(emoji))
	}
}

func (s *StateProviderMemoryOptimized) GetGuildEmoji(_ context.Context, guildID, emojiID discord.Snowflake) (*discord.Emoji, bool) {
	RecordStateRequest()
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	emoji, ok := guildEmojisState.Load(emojiID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	guildEmoji := StateEmojiToDiscord(emoji)
	RecordStateHit()
	return &guildEmoji, true
}

func (s *StateProviderMemoryOptimized) SetGuildEmoji(_ context.Context, guildID discord.Snowflake, emoji discord.Emoji) {
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		guildEmojisState = &syncmap.Map[discord.Snowflake, StateEmoji]{}

		s.GuildEmojis.Store(guildID, guildEmojisState)
	}

	guildEmojisState.Store(emoji.ID, DiscordToStateEmoji(emoji))
}

func (s *StateProviderMemoryOptimized) RemoveGuildEmoji(_ context.Context, guildID, emojiID discord.Snowflake) {
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		return
	}

	guildEmojisState.Delete(emojiID)
}

func (s *StateProviderMemoryOptimized) GetGuildStickers(_ context.Context, guildID discord.Snowflake) ([]*discord.Sticker, bool) {
	RecordStateRequest()
	guildStickersState, ok := s.GuildStickers.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	var guildStickers []*discord.Sticker

	guildStickersState.Range(func(_ discord.Snowflake, value StateSticker) bool {
		guildSticker := StateStickerToDiscord(value)
		guildStickers = append(guildStickers, &guildSticker)

		return true
	})

	RecordStateHitWithValue(float64(len(guildStickers)))
	return guildStickers, true
}

func (s *StateProviderMemoryOptimized) SetGuildStickers(_ context.Context, guildID discord.Snowflake, stickers []discord.Sticker) {
	guildStickersState, ok := s.GuildStickers.Load(guildID)
	if !ok {
		guildStickersState = &syncmap.Map[discord.Snowflake, StateSticker]{}

		s.GuildStickers.Store(guildID, guildStickersState)
	}

	for _, sticker := range stickers {
		guildStickersState.Store(sticker.ID, DiscordToStateSticker(sticker))
	}
}

func (s *StateProviderMemoryOptimized) GetGuildSticker(_ context.Context, guildID, stickerID discord.Snowflake) (*discord.Sticker, bool) {
	RecordStateRequest()
	guildStickersState, ok := s.GuildStickers.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	sticker, ok := guildStickersState.Load(stickerID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	guildSticker := StateStickerToDiscord(sticker)
	RecordStateHit()
	return &guildSticker, true
}

func (s *StateProviderMemoryOptimized) SetGuildSticker(_ context.Context, guildID discord.Snowflake, sticker discord.Sticker) {
	guildStickersState, ok := s.GuildStickers.Load(guildID)
	if !ok {
		guildStickersState = &syncmap.Map[discord.Snowflake, StateSticker]{}

		s.GuildStickers.Store(guildID, guildStickersState)
	}

	guildStickersState.Store(sticker.ID, DiscordToStateSticker(sticker))
}

func (s *StateProviderMemoryOptimized) RemoveGuildSticker(_ context.Context, guildID, stickerID discord.Snowflake) {
	guildStickersState, ok := s.GuildStickers.Load(guildID)
	if !ok {
		return
	}

	guildStickersState.Delete(stickerID)
}

func (s *StateProviderMemoryOptimized) GetVoiceStates(_ context.Context, guildID discord.Snowflake) ([]*discord.VoiceState, bool) {
	RecordStateRequest()
	voiceStatesState, ok := s.VoiceStates.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	var voiceStates []*discord.VoiceState

	voiceStatesState.Range(func(_ discord.Snowflake, value StateVoiceState) bool {
		voiceState := StateVoiceStateToDiscord(value)
		voiceStates = append(voiceStates, &voiceState)

		return true
	})

	RecordStateHitWithValue(float64(len(voiceStates)))
	return voiceStates, true
}

func (s *StateProviderMemoryOptimized) GetVoiceState(_ context.Context, guildID, userID discord.Snowflake) (*discord.VoiceState, bool) {
	RecordStateRequest()
	voiceStatesState, ok := s.VoiceStates.Load(guildID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	voiceStateState, ok := voiceStatesState.Load(userID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	voiceState := StateVoiceStateToDiscord(voiceStateState)
	RecordStateHit()
	return &voiceState, true
}

func (s *StateProviderMemoryOptimized) SetVoiceState(_ context.Context, guildID discord.Snowflake, voiceState discord.VoiceState) {
	voiceStatesState, ok := s.VoiceStates.Load(guildID)
	if !ok {
		voiceStatesState = &syncmap.Map[discord.Snowflake, StateVoiceState]{}

		s.VoiceStates.Store(guildID, voiceStatesState)
	}

	voiceStatesState.Store(voiceState.UserID, DiscordToStateVoiceState(voiceState))
}

func (s *StateProviderMemoryOptimized) RemoveVoiceState(_ context.Context, guildID, userID discord.Snowflake) {
	voiceStatesState, ok := s.VoiceStates.Load(guildID)
	if !ok {
		return
	}

	voiceStatesState.Delete(userID)
}

func (s *StateProviderMemoryOptimized) GetUser(_ context.Context, userID discord.Snowflake) (*discord.User, bool) {
	RecordStateRequest()
	userState, ok := s.Users.Load(userID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	user := StateUserToDiscord(userState)
	RecordStateHit()
	return &user, true
}

func (s *StateProviderMemoryOptimized) SetUser(_ context.Context, userID discord.Snowflake, user discord.User) {
	s.Users.Store(userID, DiscordToStateUser(user))
}

func (s *StateProviderMemoryOptimized) GetUserMutualGuilds(_ context.Context, userID discord.Snowflake) ([]discord.Snowflake, bool) {
	RecordStateRequest()
	userMutualsState, ok := s.UserMutuals.Load(userID)
	if !ok {
		RecordStateMiss()

		return nil, false
	}

	var userMutuals []discord.Snowflake

	userMutualsState.Range(func(key discord.Snowflake, _ bool) bool {
		userMutuals = append(userMutuals, key)

		return true
	})

	RecordStateHitWithValue(float64(len(userMutuals)))
	return userMutuals, true
}

func (s *StateProviderMemoryOptimized) AddUserMutualGuild(_ context.Context, userID, guildID discord.Snowflake) {
	userMutualsState, ok := s.UserMutuals.Load(userID)
	if !ok {
		userMutualsState = &syncmap.Map[discord.Snowflake, bool]{}

		s.UserMutuals.Store(userID, userMutualsState)
	}

	userMutualsState.Store(guildID, true)
}

func (s *StateProviderMemoryOptimized) RemoveUserMutualGuild(_ context.Context, userID, guildID discord.Snowflake) {
	userMutualsState, ok := s.UserMutuals.Load(userID)
	if !ok {
		return
	}

	userMutualsState.Delete(guildID)
}

type StateGuild struct {
	ID              discord.Snowflake `json:"id"`
	Name            string            `json:"name"`
	Icon            string            `json:"icon"`
	IconHash        string            `json:"icon_hash"`
	Splash          string            `json:"splash"`
	DiscoverySplash string            `json:"discovery_splash"`

	Owner       bool               `json:"owner"`
	OwnerID     *discord.Snowflake `json:"owner_id"`
	Permissions *discord.Int64     `json:"permissions"`
	Region      string             `json:"region"`

	AFKChannelID *discord.Snowflake `json:"afk_channel_id"`
	AFKTimeout   int32              `json:"afk_timeout"`

	WidgetEnabled   bool               `json:"widget_enabled"`
	WidgetChannelID *discord.Snowflake `json:"widget_channel_id"`

	VerificationLevel           discord.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications discord.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       discord.ExplicitContentFilterLevel `json:"explicit_content_filter"`

	MFALevel           discord.MFALevel            `json:"mfa_level"`
	ApplicationID      *discord.Snowflake          `json:"application_id"`
	SystemChannelID    *discord.Snowflake          `json:"system_channel_id"`
	SystemChannelFlags *discord.SystemChannelFlags `json:"system_channel_flags"`
	RulesChannelID     *discord.Snowflake          `json:"rules_channel_id"`

	JoinedAt    time.Time `json:"joined_at"`
	Large       bool      `json:"large"`
	Unavailable bool      `json:"unavailable"`
	MemberCount int32     `json:"member_count"`

	MaxPresences  int32  `json:"max_presences"`
	MaxMembers    int32  `json:"max_members"`
	VanityURLCode string `json:"vanity_url_code"`
	Description   string `json:"description"`
	Banner        string `json:"banner"`

	PremiumTier               *discord.PremiumTier       `json:"premium_tier"`
	PremiumSubscriptionCount  int32                      `json:"premium_subscription_count"`
	PreferredLocale           string                     `json:"preferred_locale"`
	PublicUpdatesChannelID    *discord.Snowflake         `json:"public_updates_channel_id"`
	MaxVideoChannelUsers      int32                      `json:"max_video_channel_users"`
	ApproximateMemberCount    int32                      `json:"approximate_member_count"`
	ApproximatePresenceCount  int32                      `json:"approximate_presence_count"`
	NSFWLevel                 discord.GuildNSFWLevelType `json:"nsfw_level"`
	PremiumProgressBarEnabled bool                       `json:"premium_progress_bar_enabled"`

	Features             []string                 `json:"features"`
	StageInstances       []discord.StageInstance  `json:"stage_instances"`
	GuildScheduledEvents []discord.ScheduledEvent `json:"guild_scheduled_events"`
}

func DiscordToStateGuild(guild discord.Guild) StateGuild {
	return StateGuild{
		ID:                          guild.ID,
		Name:                        guild.Name,
		Icon:                        guild.Icon,
		IconHash:                    guild.IconHash,
		Splash:                      guild.Splash,
		DiscoverySplash:             guild.DiscoverySplash,
		Owner:                       guild.Owner,
		OwnerID:                     guild.OwnerID,
		Permissions:                 guild.Permissions,
		Region:                      guild.Region,
		AFKChannelID:                guild.AFKChannelID,
		AFKTimeout:                  guild.AFKTimeout,
		WidgetEnabled:               guild.WidgetEnabled,
		WidgetChannelID:             guild.WidgetChannelID,
		VerificationLevel:           guild.VerificationLevel,
		DefaultMessageNotifications: guild.DefaultMessageNotifications,
		ExplicitContentFilter:       guild.ExplicitContentFilter,
		MFALevel:                    guild.MFALevel,
		ApplicationID:               guild.ApplicationID,
		SystemChannelID:             guild.SystemChannelID,
		SystemChannelFlags:          guild.SystemChannelFlags,
		RulesChannelID:              guild.RulesChannelID,
		JoinedAt:                    guild.JoinedAt,
		Large:                       guild.Large,
		Unavailable:                 guild.Unavailable,
		MemberCount:                 guild.MemberCount,
		MaxPresences:                guild.MaxPresences,
		MaxMembers:                  guild.MaxMembers,
		VanityURLCode:               guild.VanityURLCode,
		Description:                 guild.Description,
		Banner:                      guild.Banner,
		PremiumTier:                 guild.PremiumTier,
		PremiumSubscriptionCount:    guild.PremiumSubscriptionCount,
		PreferredLocale:             guild.PreferredLocale,
		PublicUpdatesChannelID:      guild.PublicUpdatesChannelID,
		MaxVideoChannelUsers:        guild.MaxVideoChannelUsers,
		ApproximateMemberCount:      guild.ApproximateMemberCount,
		ApproximatePresenceCount:    guild.ApproximatePresenceCount,
		NSFWLevel:                   guild.NSFWLevel,
		PremiumProgressBarEnabled:   guild.PremiumProgressBarEnabled,
		StageInstances:              guild.StageInstances,
		GuildScheduledEvents:        guild.GuildScheduledEvents,
		Features:                    guild.Features,
	}
}

func StateGuildToDiscord(v StateGuild) discord.Guild {
	return discord.Guild{
		ID:                          v.ID,
		Name:                        v.Name,
		Icon:                        v.Icon,
		IconHash:                    v.IconHash,
		Splash:                      v.Splash,
		DiscoverySplash:             v.DiscoverySplash,
		Owner:                       v.Owner,
		OwnerID:                     v.OwnerID,
		Permissions:                 v.Permissions,
		Region:                      v.Region,
		AFKChannelID:                v.AFKChannelID,
		AFKTimeout:                  v.AFKTimeout,
		WidgetEnabled:               v.WidgetEnabled,
		WidgetChannelID:             v.WidgetChannelID,
		VerificationLevel:           v.VerificationLevel,
		DefaultMessageNotifications: v.DefaultMessageNotifications,
		ExplicitContentFilter:       v.ExplicitContentFilter,
		MFALevel:                    v.MFALevel,
		ApplicationID:               v.ApplicationID,
		SystemChannelID:             v.SystemChannelID,
		SystemChannelFlags:          v.SystemChannelFlags,
		RulesChannelID:              v.RulesChannelID,
		JoinedAt:                    v.JoinedAt,
		Large:                       v.Large,
		Unavailable:                 v.Unavailable,
		MemberCount:                 v.MemberCount,
		MaxPresences:                v.MaxPresences,
		MaxMembers:                  v.MaxMembers,
		VanityURLCode:               v.VanityURLCode,
		Description:                 v.Description,
		Banner:                      v.Banner,
		PremiumTier:                 v.PremiumTier,
		PremiumSubscriptionCount:    v.PremiumSubscriptionCount,
		PreferredLocale:             v.PreferredLocale,
		PublicUpdatesChannelID:      v.PublicUpdatesChannelID,
		MaxVideoChannelUsers:        v.MaxVideoChannelUsers,
		ApproximateMemberCount:      v.ApproximateMemberCount,
		ApproximatePresenceCount:    v.ApproximatePresenceCount,
		NSFWLevel:                   v.NSFWLevel,
		PremiumProgressBarEnabled:   v.PremiumProgressBarEnabled,
		StageInstances:              v.StageInstances,
		GuildScheduledEvents:        v.GuildScheduledEvents,
		Features:                    v.Features,
		Presences:                   []discord.Activity{},
		Stickers:                    []discord.Sticker{},
		Roles:                       []discord.Role{},
		Emojis:                      []discord.Emoji{},
		VoiceStates:                 []discord.VoiceState{},
		Members:                     []discord.GuildMember{},
		Channels:                    []discord.Channel{},
	}
}

type StateChannel struct {
	ID                         discord.Snowflake          `json:"id"`
	GuildID                    *discord.Snowflake         `json:"guild_id"`
	OwnerID                    *discord.Snowflake         `json:"owner_id"`
	ApplicationID              *discord.Snowflake         `json:"application_id"`
	ParentID                   *discord.Snowflake         `json:"parent_id"`
	LastPinTimestamp           *time.Time                 `json:"last_pin_timestamp"`
	Permissions                *discord.Int64             `json:"permissions"`
	ThreadMetadata             *discord.ThreadMetadata    `json:"thread_metadata"`
	ThreadMember               *discord.ThreadMember      `json:"member"`
	VideoQualityMode           *discord.VideoQualityMode  `json:"video_quality_mode"`
	PermissionOverwrites       []discord.ChannelOverwrite `json:"permission_overwrites"`
	Recipients                 []discord.Snowflake        `json:"recipients"`
	Type                       discord.ChannelType        `json:"type"`
	Position                   int32                      `json:"position"`
	Bitrate                    int32                      `json:"bitrate"`
	UserLimit                  int32                      `json:"user_limit"`
	RateLimitPerUser           int32                      `json:"rate_limit_per_user"`
	MessageCount               int32                      `json:"message_count"`
	MemberCount                int32                      `json:"member_count"`
	DefaultAutoArchiveDuration int32                      `json:"default_auto_archive_duration"`
	NSFW                       bool                       `json:"nsfw"`
	Name                       string                     `json:"name"`
	Topic                      string                     `json:"topic"`
	LastMessageID              string                     `json:"last_message_id"`
	Icon                       string                     `json:"icon"`
	RTCRegion                  string                     `json:"rtc_region"`
}

func DiscordToStateChannel(v discord.Channel) StateChannel {
	channelState := StateChannel{
		ID:                         v.ID,
		Type:                       v.Type,
		GuildID:                    v.GuildID,
		Position:                   v.Position,
		PermissionOverwrites:       v.PermissionOverwrites,
		Name:                       v.Name,
		Topic:                      v.Topic,
		NSFW:                       v.NSFW,
		LastMessageID:              v.LastMessageID,
		Bitrate:                    v.Bitrate,
		UserLimit:                  v.UserLimit,
		RateLimitPerUser:           v.RateLimitPerUser,
		Recipients:                 make([]discord.Snowflake, len(v.Recipients)),
		Icon:                       v.Icon,
		OwnerID:                    v.OwnerID,
		ApplicationID:              v.ApplicationID,
		ParentID:                   v.ParentID,
		LastPinTimestamp:           v.LastPinTimestamp,
		RTCRegion:                  v.RTCRegion,
		VideoQualityMode:           v.VideoQualityMode,
		MessageCount:               v.MessageCount,
		MemberCount:                v.MemberCount,
		ThreadMetadata:             v.ThreadMetadata,
		ThreadMember:               v.ThreadMember,
		DefaultAutoArchiveDuration: v.DefaultAutoArchiveDuration,
		Permissions:                v.Permissions,
	}

	for i, recipient := range v.Recipients {
		channelState.Recipients[i] = recipient.ID
	}

	return channelState
}

func StateChannelToDiscord(v StateChannel) discord.Channel {
	channel := discord.Channel{
		ID:                         v.ID,
		Type:                       v.Type,
		GuildID:                    v.GuildID,
		Position:                   v.Position,
		PermissionOverwrites:       v.PermissionOverwrites,
		Name:                       v.Name,
		Topic:                      v.Topic,
		NSFW:                       v.NSFW,
		LastMessageID:              v.LastMessageID,
		Bitrate:                    v.Bitrate,
		UserLimit:                  v.UserLimit,
		RateLimitPerUser:           v.RateLimitPerUser,
		Recipients:                 make([]discord.User, len(v.Recipients)),
		Icon:                       v.Icon,
		OwnerID:                    v.OwnerID,
		ApplicationID:              v.ApplicationID,
		ParentID:                   v.ParentID,
		LastPinTimestamp:           v.LastPinTimestamp,
		RTCRegion:                  v.RTCRegion,
		VideoQualityMode:           v.VideoQualityMode,
		MessageCount:               v.MessageCount,
		MemberCount:                v.MemberCount,
		ThreadMetadata:             v.ThreadMetadata,
		ThreadMember:               v.ThreadMember,
		DefaultAutoArchiveDuration: v.DefaultAutoArchiveDuration,
		Permissions:                v.Permissions,
	}

	for i, recipient := range v.Recipients {
		channel.Recipients[i] = discord.User{ID: recipient}
	}

	return channel
}

type StateGuildMember struct {
	UserID                     discord.Snowflake   `json:"user_id"`
	Permissions                *discord.Int64      `json:"permissions"`
	JoinedAt                   time.Time           `json:"joined_at"`
	Roles                      []discord.Snowflake `json:"roles"`
	Nick                       string              `json:"nick"`
	Avatar                     string              `json:"avatar"`
	PremiumSince               *time.Time          `json:"premium_since"`
	CommunicationDisabledUntil *time.Time          `json:"communication_disabled_until"`
	Deaf                       bool                `json:"deaf"`
	Mute                       bool                `json:"mute"`
	Pending                    bool                `json:"pending"`
}

func DiscordToStateGuildMember(v discord.GuildMember) StateGuildMember {
	return StateGuildMember{
		UserID:                     v.User.ID,
		Permissions:                v.Permissions,
		JoinedAt:                   v.JoinedAt,
		Roles:                      v.Roles,
		Nick:                       v.Nick,
		Avatar:                     v.Avatar,
		PremiumSince:               v.PremiumSince,
		CommunicationDisabledUntil: v.CommunicationDisabledUntil,
		Deaf:                       v.Deaf,
		Mute:                       v.Mute,
		Pending:                    v.Pending,
	}
}

func StateGuildMemberToDiscord(v StateGuildMember) discord.GuildMember {
	return discord.GuildMember{
		User:                       &discord.User{ID: v.UserID},
		Permissions:                v.Permissions,
		JoinedAt:                   v.JoinedAt,
		Roles:                      v.Roles,
		Nick:                       v.Nick,
		GuildID:                    nil,
		Avatar:                     v.Avatar,
		PremiumSince:               v.PremiumSince,
		CommunicationDisabledUntil: v.CommunicationDisabledUntil,
		Deaf:                       v.Deaf,
		Mute:                       v.Mute,
		Pending:                    v.Pending,
	}
}

type StateRole struct {
	ID           discord.Snowflake `json:"id"`
	Permissions  discord.Int64     `json:"permissions"`
	Position     int32             `json:"position"`
	Color        int32             `json:"color"`
	Name         string            `json:"name"`
	Icon         string            `json:"icon"`
	UnicodeEmoji string            `json:"unicode_emoji"`
	Tags         *discord.RoleTag  `json:"tags"`
	Hoist        bool              `json:"hoist"`
	Managed      bool              `json:"managed"`
	Mentionable  bool              `json:"mentionable"`
}

func DiscordToStateRole(v discord.Role) StateRole {
	return StateRole{
		ID:           v.ID,
		Permissions:  v.Permissions,
		Position:     v.Position,
		Color:        v.Color,
		Name:         v.Name,
		Icon:         v.Icon,
		UnicodeEmoji: v.UnicodeEmoji,
		Tags:         v.Tags,
		Hoist:        v.Hoist,
		Managed:      v.Managed,
		Mentionable:  v.Mentionable,
	}
}

func StateRoleToDiscord(v StateRole) discord.Role {
	return discord.Role{
		ID:           v.ID,
		Permissions:  v.Permissions,
		Position:     v.Position,
		Color:        v.Color,
		Name:         v.Name,
		Icon:         v.Icon,
		UnicodeEmoji: v.UnicodeEmoji,
		Tags:         v.Tags,
		Hoist:        v.Hoist,
		Managed:      v.Managed,
		Mentionable:  v.Mentionable,
		GuildID:      nil,
	}
}

type StateEmoji struct {
	ID            discord.Snowflake   `json:"id"`
	UserID        discord.Snowflake   `json:"user"`
	Name          string              `json:"name"`
	Roles         []discord.Snowflake `json:"roles"`
	RequireColons bool                `json:"require_colons"`
	Managed       bool                `json:"managed"`
	Animated      bool                `json:"animated"`
	Available     bool                `json:"available"`
}

func DiscordToStateEmoji(v discord.Emoji) StateEmoji {
	stateEmoji := StateEmoji{
		ID:            v.ID,
		Name:          v.Name,
		Roles:         v.Roles,
		RequireColons: v.RequireColons,
		Managed:       v.Managed,
		Animated:      v.Animated,
		Available:     v.Available,
		UserID:        0,
	}

	if v.User != nil {
		stateEmoji.UserID = v.User.ID
	}

	return stateEmoji
}

func StateEmojiToDiscord(v StateEmoji) discord.Emoji {
	return discord.Emoji{
		ID:            v.ID,
		User:          &discord.User{ID: v.UserID},
		Name:          v.Name,
		Roles:         v.Roles,
		RequireColons: v.RequireColons,
		Managed:       v.Managed,
		Animated:      v.Animated,
		Available:     v.Available,
	}
}

type StateSticker struct {
	ID          discord.Snowflake         `json:"id"`
	PackID      discord.Snowflake         `json:"pack_id"`
	GuildID     discord.Snowflake         `json:"guild_id"`
	UserID      discord.Snowflake         `json:"user"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Tags        string                    `json:"tags"`
	Type        discord.StickerType       `json:"type"`
	FormatType  discord.StickerFormatType `json:"format_type"`
	SortValue   int32                     `json:"sort_value"`
	Available   bool                      `json:"available"`
}

func DiscordToStateSticker(v discord.Sticker) StateSticker {
	sticker := StateSticker{
		ID:          v.ID,
		Name:        v.Name,
		Description: v.Description,
		Tags:        v.Tags,
		Type:        v.Type,
		FormatType:  v.FormatType,
		SortValue:   v.SortValue,
		Available:   v.Available,
		PackID:      0,
		GuildID:     0,
		UserID:      0,
	}

	if v.PackID != nil {
		sticker.PackID = *v.PackID
	}

	if v.GuildID != nil {
		sticker.GuildID = *v.GuildID
	}

	if v.User != nil {
		sticker.UserID = v.User.ID
	}

	return sticker
}

func StateStickerToDiscord(v StateSticker) discord.Sticker {
	return discord.Sticker{
		ID:          v.ID,
		PackID:      &v.PackID,
		GuildID:     &v.GuildID,
		User:        &discord.User{ID: v.UserID},
		Name:        v.Name,
		Description: v.Description,
		Tags:        v.Tags,
		Type:        v.Type,
		FormatType:  v.FormatType,
		SortValue:   v.SortValue,
		Available:   v.Available,
	}
}

type StateVoiceState struct {
	RequestToSpeakTimestamp time.Time         `json:"request_to_speak_timestamp"`
	ChannelID               discord.Snowflake `json:"channel_id"`
	GuildID                 discord.Snowflake `json:"guild_id"`
	UserID                  discord.Snowflake `json:"user_id"`
	SessionID               string            `json:"session_id"`
	Deaf                    bool              `json:"deaf"`
	Mute                    bool              `json:"mute"`
	SelfDeaf                bool              `json:"self_deaf"`
	SelfMute                bool              `json:"self_mute"`
	SelfStream              bool              `json:"self_stream"`
	SelfVideo               bool              `json:"self_video"`
	Suppress                bool              `json:"suppress"`
}

func DiscordToStateVoiceState(v discord.VoiceState) StateVoiceState {
	voiceState := StateVoiceState{
		RequestToSpeakTimestamp: v.RequestToSpeakTimestamp,
		ChannelID:               v.ChannelID,
		GuildID:                 0,
		UserID:                  v.UserID,
		SessionID:               v.SessionID,
		Deaf:                    v.Deaf,
		Mute:                    v.Mute,
		SelfDeaf:                v.SelfDeaf,
		SelfMute:                v.SelfMute,
		SelfStream:              v.SelfStream,
		SelfVideo:               v.SelfVideo,
		Suppress:                v.Suppress,
	}

	if v.GuildID != nil {
		voiceState.GuildID = *v.GuildID
	}

	return voiceState
}

func StateVoiceStateToDiscord(v StateVoiceState) discord.VoiceState {
	return discord.VoiceState{
		RequestToSpeakTimestamp: v.RequestToSpeakTimestamp,
		ChannelID:               v.ChannelID,
		SessionID:               v.SessionID,
		Deaf:                    v.Deaf,
		Mute:                    v.Mute,
		SelfDeaf:                v.SelfDeaf,
		SelfMute:                v.SelfMute,
		SelfStream:              v.SelfStream,
		SelfVideo:               v.SelfVideo,
		Suppress:                v.Suppress,
		GuildID:                 &v.GuildID,
		Member:                  &discord.GuildMember{User: &discord.User{ID: v.UserID}},
		UserID:                  v.UserID,
	}
}

type StateUser struct {
	ID            discord.Snowflake       `json:"id"`
	DMChannelID   *discord.Snowflake      `json:"dm_channel_id"`
	AccentColor   int32                   `json:"accent_color"`
	Flags         discord.UserFlags       `json:"flags"`
	PublicFlags   discord.UserFlags       `json:"public_flags"`
	PremiumType   discord.UserPremiumType `json:"premium_type"`
	Username      string                  `json:"username"`
	Discriminator string                  `json:"discriminator"`
	GlobalName    string                  `json:"global_name"`
	Avatar        string                  `json:"avatar"`
	Banner        string                  `json:"banner"`
	Locale        string                  `json:"locale"`
	Email         string                  `json:"email"`
	Bot           bool                    `json:"bot"`
	System        bool                    `json:"system"`
	MFAEnabled    bool                    `json:"mfa_enabled"`
	Verified      bool                    `json:"verified"`
}

func DiscordToStateUser(v discord.User) StateUser {
	return StateUser{
		ID:            v.ID,
		DMChannelID:   v.DMChannelID,
		AccentColor:   v.AccentColor,
		Flags:         v.Flags,
		PublicFlags:   v.PublicFlags,
		PremiumType:   v.PremiumType,
		Username:      v.Username,
		Discriminator: v.Discriminator,
		GlobalName:    v.GlobalName,
		Avatar:        v.Avatar,
		Banner:        v.Banner,
		Locale:        v.Locale,
		Email:         v.Email,
		Bot:           v.Bot,
		System:        v.System,
		MFAEnabled:    v.MFAEnabled,
		Verified:      v.Verified,
	}
}

func StateUserToDiscord(v StateUser) discord.User {
	return discord.User{
		ID:            v.ID,
		DMChannelID:   v.DMChannelID,
		AccentColor:   v.AccentColor,
		Flags:         v.Flags,
		PublicFlags:   v.PublicFlags,
		PremiumType:   v.PremiumType,
		Username:      v.Username,
		Discriminator: v.Discriminator,
		GlobalName:    v.GlobalName,
		Avatar:        v.Avatar,
		Banner:        v.Banner,
		Locale:        v.Locale,
		Email:         v.Email,
		Bot:           v.Bot,
		System:        v.System,
		MFAEnabled:    v.MFAEnabled,
		Verified:      v.Verified,
	}
}
