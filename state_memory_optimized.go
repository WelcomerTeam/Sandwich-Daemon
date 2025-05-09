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
	Guildroles    *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateRole]]
	GuildEmojis   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateEmoji]]
	VoiceStates   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateVoiceState]]
	Users         *csmap.CsMap[discord.Snowflake, discord.User]
	UserMutuals   *csmap.CsMap[discord.Snowflake, *syncmap.Map[discord.Snowflake, bool]]
}

func NewStateProviderMemoryOptimized() *StateProviderMemoryOptimized {
	return &StateProviderMemoryOptimized{
		Guilds:        csmap.Create[discord.Snowflake, StateGuild](),
		GuildMembers:  csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateGuildMember]](),
		GuildChannels: csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateChannel]](),
		Guildroles:    csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateRole]](),
		GuildEmojis:   csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateEmoji]](),
		VoiceStates:   csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, StateVoiceState]](),
		Users:         csmap.Create[discord.Snowflake, discord.User](),
		UserMutuals:   csmap.Create[discord.Snowflake, *syncmap.Map[discord.Snowflake, bool]](),
	}
}

func (s *StateProviderMemoryOptimized) GetGuild(_ context.Context, guildID discord.Snowflake) (discord.Guild, bool) {
	guildState, guildExists := s.Guilds.Load(guildID)
	guild := StateGuildToDiscord(guildState)

	if guildChannels, exists := s.GuildChannels.Load(guildID); exists {
		guildChannels.Range(func(_ discord.Snowflake, value StateChannel) bool {
			guild.Channels = append(guild.Channels, StateChannelToDiscord(value))

			return true
		})
	}

	if guildRoles, exists := s.Guildroles.Load(guildID); exists {
		guildRoles.Range(func(_ discord.Snowflake, value StateRole) bool {
			guild.Roles = append(guild.Roles, StateRoleToDiscord(value))

			return true
		})
	}

	if guildEmojis, exists := s.GuildEmojis.Load(guildID); exists {
		guildEmojis.Range(func(_ discord.Snowflake, value StateEmoji) bool {
			guild.Emojis = append(guild.Emojis, StateEmojiToDiscord(value))

			return true
		})
	}

	return guild, guildExists
}

func (s *StateProviderMemoryOptimized) SetGuild(ctx context.Context, guildID discord.Snowflake, guild discord.Guild) {
	s.SetGuildMembers(ctx, guildID, guild.Members)
	clear(guild.Members)

	s.SetGuildChannels(ctx, guildID, guild.Channels)
	clear(guild.Channels)

	s.SetGuildRoles(ctx, guildID, guild.Roles)
	clear(guild.Roles)

	s.SetGuildEmojis(ctx, guildID, guild.Emojis)
	clear(guild.Emojis)

	s.Guilds.Store(guildID, DiscordToStateGuild(guild))
}

func (s *StateProviderMemoryOptimized) GetGuildMembers(_ context.Context, guildID discord.Snowflake) ([]discord.GuildMember, bool) {
	guildMembersState, exists := s.GuildMembers.Load(guildID)
	if !exists {
		return nil, false
	}

	var guildMembers []discord.GuildMember

	guildMembersState.Range(func(_ discord.Snowflake, value StateGuildMember) bool {
		guildMembers = append(guildMembers, StateGuildMemberToDiscord(value))

		return true
	})

	return guildMembers, exists
}

func (s *StateProviderMemoryOptimized) SetGuildMembers(_ context.Context, guildID discord.Snowflake, guildMembers []discord.GuildMember) {
	guildMembersState, ok := s.GuildMembers.Load(guildID)
	if !ok {
		guildMembersState = &syncmap.Map[discord.Snowflake, StateGuildMember]{}

		s.GuildMembers.Store(guildID, guildMembersState)
	}

	for _, member := range guildMembers {
		guildMembersState.Store(member.User.ID, DiscordToStateGuildMember(member))
	}
}

func (s *StateProviderMemoryOptimized) GetGuildMember(_ context.Context, guildID, userID discord.Snowflake) (discord.GuildMember, bool) {
	members, ok := s.GuildMembers.Load(guildID)
	if !ok {
		return discord.GuildMember{}, false
	}

	member, ok := members.Load(userID)
	if !ok {
		return discord.GuildMember{}, false
	}

	return StateGuildMemberToDiscord(member), true
}

func (s *StateProviderMemoryOptimized) SetGuildMember(_ context.Context, guildID discord.Snowflake, member discord.GuildMember) {
	guildMembersState, ok := s.GuildMembers.Load(guildID)
	if !ok {
		guildMembersState = &syncmap.Map[discord.Snowflake, StateGuildMember]{}

		s.GuildMembers.Store(guildID, guildMembersState)
	}

	guildMembersState.Store(member.User.ID, DiscordToStateGuildMember(member))
}

func (s *StateProviderMemoryOptimized) RemoveGuildMember(_ context.Context, guildID, userID discord.Snowflake) {
	guildMembersState, ok := s.GuildMembers.Load(guildID)
	if !ok {
		return
	}

	guildMembersState.Delete(userID)
}

func (s *StateProviderMemoryOptimized) GetGuildChannels(_ context.Context, guildID discord.Snowflake) ([]discord.Channel, bool) {
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		return nil, false
	}

	var guildChannels []discord.Channel

	guildChannelsState.Range(func(_ discord.Snowflake, value StateChannel) bool {
		guildChannels = append(guildChannels, StateChannelToDiscord(value))

		return true
	})

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

func (s *StateProviderMemoryOptimized) GetGuildChannel(_ context.Context, guildID, channelID discord.Snowflake) (discord.Channel, bool) {
	guildChannelsState, ok := s.GuildChannels.Load(guildID)
	if !ok {
		return discord.Channel{}, false
	}

	channel, ok := guildChannelsState.Load(channelID)
	if !ok {
		return discord.Channel{}, false
	}

	return StateChannelToDiscord(channel), true
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

func (s *StateProviderMemoryOptimized) GetGuildRoles(_ context.Context, guildID discord.Snowflake) ([]discord.Role, bool) {
	guildRolesState, ok := s.Guildroles.Load(guildID)
	if !ok {
		return nil, false
	}

	var guildRoles []discord.Role

	guildRolesState.Range(func(_ discord.Snowflake, value StateRole) bool {
		guildRoles = append(guildRoles, StateRoleToDiscord(value))

		return true
	})

	return guildRoles, true
}

func (s *StateProviderMemoryOptimized) SetGuildRoles(_ context.Context, guildID discord.Snowflake, roles []discord.Role) {
	guildRolesState, ok := s.Guildroles.Load(guildID)
	if !ok {
		guildRolesState = &syncmap.Map[discord.Snowflake, StateRole]{}

		s.Guildroles.Store(guildID, guildRolesState)
	}

	for _, role := range roles {
		guildRolesState.Store(role.ID, DiscordToStateRole(role))
	}
}

func (s *StateProviderMemoryOptimized) GetGuildRole(_ context.Context, guildID, roleID discord.Snowflake) (discord.Role, bool) {
	guildRolesState, ok := s.Guildroles.Load(guildID)
	if !ok {
		return discord.Role{}, false
	}

	role, ok := guildRolesState.Load(roleID)
	if !ok {
		return discord.Role{}, false
	}

	return StateRoleToDiscord(role), true
}

func (s *StateProviderMemoryOptimized) SetGuildRole(_ context.Context, guildID discord.Snowflake, role discord.Role) {
	guildRolesState, ok := s.Guildroles.Load(guildID)
	if !ok {
		guildRolesState = &syncmap.Map[discord.Snowflake, StateRole]{}

		s.Guildroles.Store(guildID, guildRolesState)
	}

	guildRolesState.Store(role.ID, DiscordToStateRole(role))
}

func (s *StateProviderMemoryOptimized) RemoveGuildRole(_ context.Context, guildID, roleID discord.Snowflake) {
	guildRolesState, ok := s.Guildroles.Load(guildID)
	if !ok {
		return
	}

	guildRolesState.Delete(roleID)
}

func (s *StateProviderMemoryOptimized) GetGuildEmojis(_ context.Context, guildID discord.Snowflake) ([]discord.Emoji, bool) {
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		return nil, false
	}

	var guildEmojis []discord.Emoji

	guildEmojisState.Range(func(_ discord.Snowflake, value StateEmoji) bool {
		guildEmojis = append(guildEmojis, StateEmojiToDiscord(value))

		return true
	})

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

func (s *StateProviderMemoryOptimized) GetGuildEmoji(_ context.Context, guildID, emojiID discord.Snowflake) (discord.Emoji, bool) {
	guildEmojisState, ok := s.GuildEmojis.Load(guildID)
	if !ok {
		return discord.Emoji{}, false
	}

	emoji, ok := guildEmojisState.Load(emojiID)
	if !ok {
		return discord.Emoji{}, false
	}

	return StateEmojiToDiscord(emoji), true
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

func (s *StateProviderMemoryOptimized) GetVoiceStates(_ context.Context, guildID discord.Snowflake) ([]discord.VoiceState, bool) {
	voiceStatesState, ok := s.VoiceStates.Load(guildID)
	if !ok {
		return nil, false
	}

	var voiceStates []discord.VoiceState

	voiceStatesState.Range(func(_ discord.Snowflake, value StateVoiceState) bool {
		voiceStates = append(voiceStates, StateVoiceStateToDiscord(value))

		return true
	})

	return voiceStates, true
}

func (s *StateProviderMemoryOptimized) GetVoiceState(_ context.Context, guildID, userID discord.Snowflake) (discord.VoiceState, bool) {
	voiceStatesState, ok := s.VoiceStates.Load(guildID)
	if !ok {
		return discord.VoiceState{}, false
	}

	voiceState, ok := voiceStatesState.Load(userID)
	if !ok {
		return discord.VoiceState{}, false
	}

	return StateVoiceStateToDiscord(voiceState), true
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

func (s *StateProviderMemoryOptimized) GetUser(_ context.Context, userID discord.Snowflake) (discord.User, bool) {
	user, ok := s.Users.Load(userID)

	return user, ok
}

func (s *StateProviderMemoryOptimized) SetUser(_ context.Context, userID discord.Snowflake, user discord.User) {
	s.Users.Store(userID, user)
}

func (s *StateProviderMemoryOptimized) GetUserMutualGuilds(_ context.Context, userID discord.Snowflake) ([]discord.Snowflake, bool) {
	userMutualsState, ok := s.UserMutuals.Load(userID)
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
	OwnerID     *discord.Snowflake `json:"owner_id,omitempty"`
	Permissions *discord.Int64     `json:"permissions,omitempty"`
	Region      string             `json:"region"`

	AFKChannelID *discord.Snowflake `json:"afk_channel_id,omitempty"`
	AFKTimeout   int32              `json:"afk_timeout"`

	WidgetEnabled   bool               `json:"widget_enabled"`
	WidgetChannelID *discord.Snowflake `json:"widget_channel_id,omitempty"`

	VerificationLevel           discord.VerificationLevel          `json:"verification_level"`
	DefaultMessageNotifications discord.MessageNotificationLevel   `json:"default_message_notifications"`
	ExplicitContentFilter       discord.ExplicitContentFilterLevel `json:"explicit_content_filter"`

	MFALevel           discord.MFALevel            `json:"mfa_level"`
	ApplicationID      *discord.Snowflake          `json:"application_id,omitempty"`
	SystemChannelID    *discord.Snowflake          `json:"system_channel_id,omitempty"`
	SystemChannelFlags *discord.SystemChannelFlags `json:"system_channel_flags,omitempty"`
	RulesChannelID     *discord.Snowflake          `json:"rules_channel_id,omitempty"`

	JoinedAt    time.Time `json:"joined_at"`
	Large       bool      `json:"large"`
	Unavailable bool      `json:"unavailable"`
	MemberCount int32     `json:"member_count"`

	MaxPresences  int32  `json:"max_presences"`
	MaxMembers    int32  `json:"max_members"`
	VanityURLCode string `json:"vanity_url_code"`
	Description   string `json:"description"`
	Banner        string `json:"banner"`

	PremiumTier               *discord.PremiumTier       `json:"premium_tier,omitempty"`
	PremiumSubscriptionCount  int32                      `json:"premium_subscription_count"`
	PreferredLocale           string                     `json:"preferred_locale"`
	PublicUpdatesChannelID    *discord.Snowflake         `json:"public_updates_channel_id,omitempty"`
	MaxVideoChannelUsers      int32                      `json:"max_video_channel_users"`
	ApproximateMemberCount    int32                      `json:"approximate_member_count"`
	ApproximatePresenceCount  int32                      `json:"approximate_presence_count"`
	NSFWLevel                 discord.GuildNSFWLevelType `json:"nsfw_level"`
	PremiumProgressBarEnabled bool                       `json:"premium_progress_bar_enabled"`

	RoleIDs    []discord.Snowflake `json:"role_ids"`
	EmojiIDs   []discord.Snowflake `json:"emoji_ids"`
	ChannelIDs []discord.Snowflake `json:"channel_ids"`

	Features             []string                 `json:"features"`
	StageInstances       []discord.StageInstance  `json:"stage_instances"`
	Stickers             []discord.Sticker        `json:"stickers"`
	GuildScheduledEvents []discord.ScheduledEvent `json:"guild_scheduled_events"`
}

func DiscordToStateGuild(guild discord.Guild) StateGuild {
	return StateGuild{
		ID:              guild.ID,
		Name:            guild.Name,
		Icon:            guild.Icon,
		IconHash:        guild.IconHash,
		Splash:          guild.Splash,
		DiscoverySplash: guild.DiscoverySplash,

		Owner:       guild.Owner,
		OwnerID:     guild.OwnerID,
		Permissions: guild.Permissions,
		Region:      guild.Region,

		AFKChannelID: guild.AFKChannelID,
		AFKTimeout:   guild.AFKTimeout,

		WidgetEnabled:   guild.WidgetEnabled,
		WidgetChannelID: guild.WidgetChannelID,

		VerificationLevel:           guild.VerificationLevel,
		DefaultMessageNotifications: guild.DefaultMessageNotifications,
		ExplicitContentFilter:       guild.ExplicitContentFilter,

		MFALevel:           guild.MFALevel,
		ApplicationID:      guild.ApplicationID,
		SystemChannelID:    guild.SystemChannelID,
		SystemChannelFlags: guild.SystemChannelFlags,
		RulesChannelID:     guild.RulesChannelID,

		JoinedAt:    guild.JoinedAt,
		Large:       guild.Large,
		Unavailable: guild.Unavailable,
		MemberCount: guild.MemberCount,

		MaxPresences:  guild.MaxPresences,
		MaxMembers:    guild.MaxMembers,
		VanityURLCode: guild.VanityURLCode,
		Description:   guild.Description,
		Banner:        guild.Banner,

		PremiumTier:               guild.PremiumTier,
		PremiumSubscriptionCount:  guild.PremiumSubscriptionCount,
		PreferredLocale:           guild.PreferredLocale,
		PublicUpdatesChannelID:    guild.PublicUpdatesChannelID,
		MaxVideoChannelUsers:      guild.MaxVideoChannelUsers,
		ApproximateMemberCount:    guild.ApproximateMemberCount,
		ApproximatePresenceCount:  guild.ApproximatePresenceCount,
		NSFWLevel:                 guild.NSFWLevel,
		PremiumProgressBarEnabled: guild.PremiumProgressBarEnabled,

		StageInstances: guild.StageInstances,
		Stickers:       guild.Stickers,
	}
}

func StateGuildToDiscord(v StateGuild) discord.Guild {
	return discord.Guild{
		ID:              v.ID,
		Name:            v.Name,
		Icon:            v.Icon,
		IconHash:        v.IconHash,
		Splash:          v.Splash,
		DiscoverySplash: v.DiscoverySplash,

		Owner:       v.Owner,
		OwnerID:     v.OwnerID,
		Permissions: v.Permissions,
		Region:      v.Region,

		AFKChannelID: v.AFKChannelID,
		AFKTimeout:   v.AFKTimeout,

		WidgetEnabled:   v.WidgetEnabled,
		WidgetChannelID: v.WidgetChannelID,

		VerificationLevel:           v.VerificationLevel,
		DefaultMessageNotifications: v.DefaultMessageNotifications,
		ExplicitContentFilter:       v.ExplicitContentFilter,

		MFALevel:           v.MFALevel,
		ApplicationID:      v.ApplicationID,
		SystemChannelID:    v.SystemChannelID,
		SystemChannelFlags: v.SystemChannelFlags,
		RulesChannelID:     v.RulesChannelID,

		JoinedAt:    v.JoinedAt,
		Large:       v.Large,
		Unavailable: v.Unavailable,
		MemberCount: v.MemberCount,

		MaxPresences:  v.MaxPresences,
		MaxMembers:    v.MaxMembers,
		VanityURLCode: v.VanityURLCode,
		Description:   v.Description,
		Banner:        v.Banner,

		PremiumTier:               v.PremiumTier,
		PremiumSubscriptionCount:  v.PremiumSubscriptionCount,
		PreferredLocale:           v.PreferredLocale,
		PublicUpdatesChannelID:    v.PublicUpdatesChannelID,
		MaxVideoChannelUsers:      v.MaxVideoChannelUsers,
		ApproximateMemberCount:    v.ApproximateMemberCount,
		ApproximatePresenceCount:  v.ApproximatePresenceCount,
		NSFWLevel:                 v.NSFWLevel,
		PremiumProgressBarEnabled: v.PremiumProgressBarEnabled,

		StageInstances:       v.StageInstances,
		Stickers:             v.Stickers,
		GuildScheduledEvents: v.GuildScheduledEvents,
	}
}

type StateChannel struct {
	ID                         discord.Snowflake          `json:"id"`
	GuildID                    *discord.Snowflake         `json:"guild_id,omitempty"`
	OwnerID                    *discord.Snowflake         `json:"owner_id,omitempty"`
	ApplicationID              *discord.Snowflake         `json:"application_id,omitempty"`
	ParentID                   *discord.Snowflake         `json:"parent_id,omitempty"`
	LastPinTimestamp           *time.Time                 `json:"last_pin_timestamp,omitempty"`
	Permissions                *discord.Int64             `json:"permissions,omitempty"`
	ThreadMetadata             *discord.ThreadMetadata    `json:"thread_metadata,omitempty"`
	ThreadMember               *discord.ThreadMember      `json:"member,omitempty"`
	VideoQualityMode           *discord.VideoQualityMode  `json:"video_quality_mode,omitempty"`
	PermissionOverwrites       []discord.ChannelOverwrite `json:"permission_overwrites,omitempty"`
	Recipients                 []discord.Snowflake        `json:"recipients,omitempty"`
	Type                       discord.ChannelType        `json:"type"`
	Position                   int32                      `json:"position,omitempty"`
	Bitrate                    int32                      `json:"bitrate,omitempty"`
	UserLimit                  int32                      `json:"user_limit,omitempty"`
	RateLimitPerUser           int32                      `json:"rate_limit_per_user,omitempty"`
	MessageCount               int32                      `json:"message_count,omitempty"`
	MemberCount                int32                      `json:"member_count,omitempty"`
	DefaultAutoArchiveDuration int32                      `json:"default_auto_archive_duration,omitempty"`
	NSFW                       bool                       `json:"nsfw"`
	Name                       string                     `json:"name,omitempty"`
	Topic                      string                     `json:"topic,omitempty"`
	LastMessageID              string                     `json:"last_message_id,omitempty"`
	Icon                       string                     `json:"icon,omitempty"`
	RTCRegion                  string                     `json:"rtc_region,omitempty"`
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
	Avatar                     string              `json:"avatar,omitempty"`
	PremiumSince               string              `json:"premium_since"`
	CommunicationDisabledUntil string              `json:"communication_disabled_until,omitempty"`
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
		User:        &discord.User{ID: v.UserID},
		Permissions: v.Permissions,
		JoinedAt:    v.JoinedAt,
		Roles:       v.Roles,
		Nick:        v.Nick,
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
	}
}

type StateEmoji struct {
	ID            discord.Snowflake   `json:"id"`
	UserID        discord.Snowflake   `json:"user"`
	Name          string              `json:"name"`
	Roles         []discord.Snowflake `json:"roles,omitempty"`
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

type StateVoiceState struct {
	RequestToSpeakTimestamp time.Time         `json:"request_to_speak_timestamp"`
	ChannelID               discord.Snowflake `json:"channel_id"`
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
	return StateVoiceState{
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
	}
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
	}
}

type StateUser struct {
	ID            discord.Snowflake       `json:"id"`
	DMChannelID   *discord.Snowflake      `json:"dm_channel_id,omitempty"`
	AccentColor   int32                   `json:"accent_color"`
	Flags         discord.UserFlags       `json:"flags,omitempty"`
	PublicFlags   discord.UserFlags       `json:"public_flags,omitempty"`
	PremiumType   discord.UserPremiumType `json:"premium_type,omitempty"`
	Username      string                  `json:"username"`
	Discriminator string                  `json:"discriminator"`
	GlobalName    string                  `json:"global_name"`
	Avatar        string                  `json:"avatar"`
	Banner        string                  `json:"banner,omitempty"`
	Locale        string                  `json:"locale,omitempty"`
	Email         string                  `json:"email,omitempty"`
	Bot           bool                    `json:"bot"`
	System        bool                    `json:"system,omitempty"`
	MFAEnabled    bool                    `json:"mfa_enabled,omitempty"`
	Verified      bool                    `json:"verified,omitempty"`
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
	}
}
