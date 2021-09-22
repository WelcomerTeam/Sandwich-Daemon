package internal

import (
	"sync"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
)

//
// Guild Operations
//

// GuildFromState converts the structs.StateGuild into a discord.Guild, for use within the application.
// Channels, Roles, Members and Emoji lists will not be populated.
func (ss *SandwichState) GuildFromState(guildState *structs.StateGuild) (guild *discord.Guild) {
	return &discord.Guild{
		ID:              guildState.ID,
		Name:            guildState.Name,
		Icon:            guildState.Icon,
		IconHash:        guildState.IconHash,
		Splash:          guildState.Splash,
		DiscoverySplash: guildState.DiscoverySplash,

		Owner:       guildState.Owner,
		OwnerID:     guildState.OwnerID,
		Permissions: guildState.Permissions,
		Region:      guildState.Region,

		AFKChannelID: guildState.AFKChannelID,
		AFKTimeout:   guildState.AFKTimeout,

		WidgetEnabled:   guildState.WidgetEnabled,
		WidgetChannelID: guildState.WidgetChannelID,

		VerificationLevel:           guildState.VerificationLevel,
		DefaultMessageNotifications: guildState.DefaultMessageNotifications,
		ExplicitContentFilter:       guildState.ExplicitContentFilter,

		Features: guildState.Features,

		MFALevel:           guildState.MFALevel,
		ApplicationID:      guildState.ApplicationID,
		SystemChannelID:    guildState.SystemChannelID,
		SystemChannelFlags: guildState.SystemChannelFlags,
		RulesChannelID:     guildState.RulesChannelID,

		JoinedAt:    guildState.JoinedAt,
		Large:       guildState.Large,
		Unavailable: guildState.Unavailable,
		MemberCount: guildState.MemberCount,

		MaxPresences:  guildState.MaxPresences,
		MaxMembers:    guildState.MaxMembers,
		VanityURLCode: guildState.VanityURLCode,
		Description:   guildState.Description,
		Banner:        guildState.Banner,
		PremiumTier:   guildState.PremiumTier,

		PremiumSubscriptionCount: guildState.PremiumSubscriptionCount,
		PreferredLocale:          guildState.PreferredLocale,
		PublicUpdatesChannelID:   guildState.PublicUpdatesChannelID,
		MaxVideoChannelUsers:     guildState.MaxVideoChannelUsers,
		ApproximateMemberCount:   guildState.ApproximateMemberCount,
		ApproximatePresenceCount: guildState.ApproximatePresenceCount,

		NSFWLevel:      guildState.NSFWLevel,
		StageInstances: guildState.StageInstances,
		Stickers:       guildState.Stickers,
	}
}

// GuildFromState converts from discord.Guild to structs.StateGuild, for storing in cache.
// Does not add Channels, Roles, Members and Emojis to state.
func (ss *SandwichState) GuildToState(guild *discord.Guild) (guildState *structs.StateGuild) {
	return &structs.StateGuild{
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

		Features: guild.Features,

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
		PremiumTier:   guild.PremiumTier,

		PremiumSubscriptionCount: guild.PremiumSubscriptionCount,
		PreferredLocale:          guild.PreferredLocale,
		PublicUpdatesChannelID:   guild.PublicUpdatesChannelID,
		MaxVideoChannelUsers:     guild.MaxVideoChannelUsers,
		ApproximateMemberCount:   guild.ApproximateMemberCount,
		ApproximatePresenceCount: guild.ApproximatePresenceCount,

		NSFWLevel:      guild.NSFWLevel,
		StageInstances: guild.StageInstances,
		Stickers:       guild.Stickers,
	}
}

// GetGuild returns the guild with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuild(guildID discord.Snowflake) (guild *discord.Guild, ok bool) {
	ss.guildsMu.RLock()
	defer ss.guildsMu.RUnlock()

	stateGuild, ok := ss.Guilds[guildID]
	if !ok {
		return
	}

	guild = ss.GuildFromState(stateGuild)

	return
}

// SetGuild creates or updates a guild entry in the cache.
func (ss *SandwichState) SetGuild(guild *discord.Guild) {
	ss.guildsMu.Lock()
	defer ss.guildsMu.Unlock()

	for _, role := range guild.Roles {
		ss.SetRole(role)
	}

	for _, channel := range guild.Channels {
		ss.SetChannel(channel)
	}

	for _, emoji := range guild.Emojis {
		ss.SetEmoji(emoji)
	}

	for _, member := range guild.Members {
		ss.SetGuildMember(guild.ID, member)
	}

	ss.Guilds[guild.ID] = ss.GuildToState(guild)

	return
}

// RemoveGuild removes a guild from the cache.
func (ss *SandwichState) RemoveGuild(guildID discord.Snowflake) {
	ss.guildsMu.Lock()

	guild, ok := ss.Guilds[guildID]
	if ok {
		for _, roleID := range guild.RoleIDs {
			ss.RemoveRole(*roleID)
		}

		for _, channelID := range guild.ChannelIDs {
			ss.RemoveChannel(*channelID)
		}

		for _, emojiID := range guild.EmojiIDs {
			ss.RemoveEmoji(*emojiID)
		}

		ss.RemoveAllGuildMembers(guildID)
	}

	delete(ss.Guilds, guildID)
	ss.guildsMu.RUnlock()

	return
}

//
// GuildMember Operations
//

// GuildMemberFromState converts the structs.StateGuildMembers into a discord.GuildMember,
// for use within the application.
// This will not populate the user object from cache, it will be an empty object with only an ID.
func (ss *SandwichState) GuildMemberFromState(guildState *structs.StateGuildMember) (guild *discord.GuildMember) {
	return &discord.GuildMember{
		User: &discord.User{
			ID: *guildState.UserID,
		},
		Nick: guildState.Nick,

		Roles:    guildState.Roles,
		JoinedAt: guildState.JoinedAt,
		Deaf:     guildState.Deaf,
		Mute:     guildState.Mute,
	}
}

// GuildMemberFromState converts from discord.GuildMember to structs.StateGuildMembers, for storing in cache.
// This does not add the user to the cache.
func (ss *SandwichState) GuildMemberToState(guild *discord.GuildMember) (guildState *structs.StateGuildMember) {
	return &structs.StateGuildMember{
		UserID: &guild.User.ID,
		Nick:   guild.Nick,

		Roles:    guild.Roles,
		JoinedAt: guild.JoinedAt,
		Deaf:     guild.Deaf,
		Mute:     guild.Mute,
	}
}

// GetGuildMember returns the guildMember with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildMember(guildID discord.Snowflake, guildMemberID discord.Snowflake) (guildMember *discord.GuildMember, ok bool) {
	ss.guildMembersMu.RLock()
	defer ss.guildMembersMu.RUnlock()

	guildMembers, ok := ss.GuildMembers[guildID]
	if !ok {
		return
	}

	guildMembers.MembersMu.RLock()
	defer guildMembers.MembersMu.RUnlock()

	stateGuildMember, ok := guildMembers.Members[guildMemberID]
	if !ok {
		return
	}

	guildMember = ss.GuildMemberFromState(stateGuildMember)

	user, ok := ss.GetUser(guildMember.User.ID)
	if ok {
		guildMember.User = user
	}

	return
}

// SetGuildMember creates or updates a guildMember entry in the cache. Adds user in guildMember object to cache.
func (ss *SandwichState) SetGuildMember(guildID discord.Snowflake, guildMember *discord.GuildMember) {
	ss.guildMembersMu.Lock()
	defer ss.guildMembersMu.Unlock()

	guildMembers, ok := ss.GuildMembers[guildID]
	if !ok {
		guildMembers = &structs.StateGuildMembers{
			MembersMu: sync.RWMutex{},
			Members:   make(map[discord.Snowflake]*structs.StateGuildMember),
		}

		ss.GuildMembers[guildID] = guildMembers
	}

	guildMembers.MembersMu.Lock()
	defer guildMembers.MembersMu.Unlock()

	guildMembers.Members[guildMember.User.ID] = ss.GuildMemberToState(guildMember)

	ss.SetUser(guildMember.User)

	return
}

// RemoveGuildMember removes a guildMember from the cache.
func (ss *SandwichState) RemoveGuildMember(guildID discord.Snowflake, guildMemberID discord.Snowflake) {
	ss.guildMembersMu.RLock()
	defer ss.guildMembersMu.RUnlock()

	guildMembers, ok := ss.GuildMembers[guildID]
	if !ok {
		return
	}

	guildMembers.MembersMu.RLock()
	defer guildMembers.MembersMu.RUnlock()

	delete(guildMembers.Members, guildMemberID)

	return
}

// RemoveAllGuildMembers removes all guildMembers of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildMembers(guildID discord.Snowflake) {
	ss.guildMembersMu.RLock()
	defer ss.guildMembersMu.RUnlock()

	delete(ss.GuildMembers, guildID)

	return
}

//
// Role Operations
//

// RoleFromState converts the structs.StateRole into a discord.Role, for use within the application.
func (ss *SandwichState) RoleFromState(guildState *structs.StateRole) (guild *discord.Role) {
	return &discord.Role{
		ID:          guildState.ID,
		Name:        guildState.Name,
		Color:       guildState.Color,
		Hoist:       guildState.Hoist,
		Position:    guildState.Position,
		Permissions: guildState.Permissions,
		Managed:     guildState.Managed,
		Mentionable: guildState.Mentionable,
		Tags:        guildState.Tags,
	}
}

// RoleFromState converts from discord.Role to structs.StateRole, for storing in cache.
func (ss *SandwichState) RoleToState(guild *discord.Role) (guildState *structs.StateRole) {
	return &structs.StateRole{
		ID:          guild.ID,
		Name:        guild.Name,
		Color:       guild.Color,
		Hoist:       guild.Hoist,
		Position:    guild.Position,
		Permissions: guild.Permissions,
		Managed:     guild.Managed,
		Mentionable: guild.Mentionable,
		Tags:        guild.Tags,
	}
}

// GetRole returns the role with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetRole(roleID discord.Snowflake) (role *discord.Role, ok bool) {
	ss.rolesMu.RLock()
	defer ss.rolesMu.RUnlock()

	stateRole, ok := ss.Roles[roleID]
	if !ok {
		return
	}

	role = ss.RoleFromState(stateRole)

	return
}

// SetRole creates or updates a role entry in the cache.
func (ss *SandwichState) SetRole(role *discord.Role) {
	ss.rolesMu.Lock()
	defer ss.rolesMu.Unlock()

	ss.Roles[role.ID] = ss.RoleToState(role)

	return
}

// RemoveRole removes a role from the cache.
func (ss *SandwichState) RemoveRole(roleID discord.Snowflake) {
	ss.rolesMu.Lock()
	defer ss.rolesMu.RUnlock()

	delete(ss.Roles, roleID)

	return
}

//
// Emoji Operations
//

// EmojiFromState converts the structs.StateEmoji into a discord.Emoji, for use within the application.
func (ss *SandwichState) EmojiFromState(guildState *structs.StateEmoji) (guild *discord.Emoji) {
	return &discord.Emoji{
		ID:    guildState.ID,
		Name:  guildState.Name,
		Roles: guildState.Roles,
		User: &discord.User{
			ID: *guildState.UserID,
		},
		RequireColons: guildState.RequireColons,
		Managed:       guildState.Managed,
		Animated:      guildState.Animated,
		Available:     guildState.Available,
	}
}

// EmojiFromState converts from discord.Emoji to structs.StateEmoji, for storing in cache.
// This does not add the user to the cache.
// This will not populate the user object from cache, it will be an empty object with only an ID.
func (ss *SandwichState) EmojiToState(guild *discord.Emoji) (guildState *structs.StateEmoji) {
	guildState = &structs.StateEmoji{
		ID:            guild.ID,
		Name:          guild.Name,
		Roles:         guild.Roles,
		RequireColons: guild.RequireColons,
		Managed:       guild.Managed,
		Animated:      guild.Animated,
		Available:     guild.Available,
	}

	if guild.User != nil {
		guildState.UserID = &guild.User.ID
	}

	return guildState
}

// GetEmoji returns the emoji with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetEmoji(emojiID discord.Snowflake) (emoji *discord.Emoji, ok bool) {
	ss.emojisMu.RLock()
	defer ss.emojisMu.RUnlock()

	stateEmoji, ok := ss.Emojis[emojiID]
	if !ok {
		return
	}

	emoji = ss.EmojiFromState(stateEmoji)

	user, ok := ss.GetUser(emoji.User.ID)
	if ok {
		emoji.User = user
	}

	return
}

// SetEmoji creates or updates a emoji entry in the cache. Adds user in user object to cache.
func (ss *SandwichState) SetEmoji(emoji *discord.Emoji) {
	ss.emojisMu.Lock()
	defer ss.emojisMu.Unlock()

	ss.Emojis[emoji.ID] = ss.EmojiToState(emoji)

	if emoji.User != nil {
		ss.SetUser(emoji.User)
	}

	return
}

// RemoveEmoji removes a emoji from the cache.
func (ss *SandwichState) RemoveEmoji(emojiID discord.Snowflake) {
	ss.emojisMu.Lock()
	defer ss.emojisMu.RUnlock()

	delete(ss.Emojis, emojiID)

	return
}

//
// User Operations
//

// UserFromState converts the structs.StateUser into a discord.User, for use within the application.
func (ss *SandwichState) UserFromState(userState *structs.StateUser) (user *discord.User) {
	return &discord.User{
		ID:            userState.ID,
		Username:      userState.Username,
		Discriminator: userState.Discriminator,
		Avatar:        userState.Avatar,
		Bot:           userState.Bot,
		System:        userState.System,
		MFAEnabled:    userState.MFAEnabled,
		Banner:        userState.Banner,
		Locale:        userState.Locale,
		Verified:      userState.Verified,
		Email:         userState.Email,
		Flags:         userState.Flags,
		PremiumType:   userState.PremiumType,
		PublicFlags:   userState.PublicFlags,
	}
}

// UserFromState converts from discord.User to structs.StateUser, for storing in cache.
func (ss *SandwichState) UserToState(user *discord.User) (userState *structs.StateUser) {
	return &structs.StateUser{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		Avatar:        user.Avatar,
		Bot:           user.Bot,
		System:        user.System,
		MFAEnabled:    user.MFAEnabled,
		Banner:        user.Banner,
		Locale:        user.Locale,
		Verified:      user.Verified,
		Email:         user.Email,
		Flags:         user.Flags,
		PremiumType:   user.PremiumType,
		PublicFlags:   user.PublicFlags,
	}
}

// GetUser returns the user with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetUser(userID discord.Snowflake) (user *discord.User, ok bool) {
	ss.usersMu.RLock()
	defer ss.usersMu.RUnlock()

	stateUser, ok := ss.Users[userID]
	if !ok {
		return
	}

	user = ss.UserFromState(stateUser)

	return
}

// SetUser creates or updates a user entry in the cache.
func (ss *SandwichState) SetUser(user *discord.User) {
	ss.usersMu.Lock()
	defer ss.usersMu.Unlock()

	ss.Users[user.ID] = ss.UserToState(user)

	return
}

// RemoveUser removes a user from the cache.
func (ss *SandwichState) RemoveUser(userID discord.Snowflake) {
	ss.usersMu.Lock()
	defer ss.usersMu.RUnlock()

	delete(ss.Users, userID)

	return
}

//
// Channel Operations
//

// ChannelFromState converts the structs.StateChannel into a discord.Channel, for use within the application.
// This will not populate the recipient user object from cache.
func (ss *SandwichState) ChannelFromState(guildState *structs.StateChannel) (guild *discord.Channel) {
	guild = &discord.Channel{
		ID:                   guildState.ID,
		Type:                 guildState.Type,
		GuildID:              guildState.GuildID,
		Position:             guildState.Position,
		PermissionOverwrites: guildState.PermissionOverwrites,
		Name:                 guildState.Name,
		Topic:                guildState.Topic,
		NSFW:                 guildState.NSFW,
		LastMessageID:        guildState.LastMessageID,
		Bitrate:              guildState.Bitrate,
		UserLimit:            guildState.UserLimit,
		RateLimitPerUser:     guildState.RateLimitPerUser,
		Recipients:           make([]*discord.User, 0),
		Icon:                 guildState.Icon,
		OwnerID:              guildState.OwnerID,
		ApplicationID:        guildState.ApplicationID,
		ParentID:             guildState.ParentID,
		LastPinTimestamp:     guildState.LastPinTimestamp,

		RTCRegion:        guildState.RTCRegion,
		VideoQualityMode: guildState.VideoQualityMode,

		MessageCount:               guildState.MessageCount,
		MemberCount:                guildState.MemberCount,
		ThreadMetadata:             guildState.ThreadMetadata,
		ThreadMember:               guildState.ThreadMember,
		DefaultAutoArchiveDuration: guildState.DefaultAutoArchiveDuration,

		Permissions: guildState.Permissions,
	}

	for _, recepientID := range guildState.RecipientIDs {
		guild.Recipients = append(guild.Recipients, &discord.User{
			ID: *recepientID,
		})
	}

	return guild
}

// ChannelFromState converts from discord.Channel to structs.StateChannel, for storing in cache.
// This does not add the recipients to the cache.
func (ss *SandwichState) ChannelToState(guild *discord.Channel) (guildState *structs.StateChannel) {
	guildState = &structs.StateChannel{
		ID:                   guild.ID,
		Type:                 guild.Type,
		GuildID:              guild.GuildID,
		Position:             guild.Position,
		PermissionOverwrites: guild.PermissionOverwrites,
		Name:                 guild.Name,
		Topic:                guild.Topic,
		NSFW:                 guild.NSFW,
		LastMessageID:        guild.LastMessageID,
		Bitrate:              guild.Bitrate,
		UserLimit:            guild.UserLimit,
		RateLimitPerUser:     guild.RateLimitPerUser,
		RecipientIDs:         make([]*discord.Snowflake, 0),
		Icon:                 guild.Icon,
		OwnerID:              guild.OwnerID,
		ApplicationID:        guild.ApplicationID,
		ParentID:             guild.ParentID,
		LastPinTimestamp:     guild.LastPinTimestamp,

		RTCRegion:        guild.RTCRegion,
		VideoQualityMode: guild.VideoQualityMode,

		MessageCount:               guild.MessageCount,
		MemberCount:                guild.MemberCount,
		ThreadMetadata:             guild.ThreadMetadata,
		ThreadMember:               guild.ThreadMember,
		DefaultAutoArchiveDuration: guild.DefaultAutoArchiveDuration,

		Permissions: guild.Permissions,
	}

	for _, recipient := range guild.Recipients {
		guildState.RecipientIDs = append(guildState.RecipientIDs, &recipient.ID)
	}

	return guildState
}

// GetChannel returns the channel with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetChannel(channelID discord.Snowflake) (channel *discord.Channel, ok bool) {
	ss.channelsMu.RLock()
	defer ss.channelsMu.RUnlock()

	stateChannel, ok := ss.Channels[channelID]
	if !ok {
		return
	}

	channel = ss.ChannelFromState(stateChannel)

	for _, recipient := range channel.Recipients {
		recipientUser, ok := ss.GetUser(recipient.ID)
		if ok {
			recipient = recipientUser
		}
	}

	return
}

// SetChannel creates or updates a channel entry in the cache.
func (ss *SandwichState) SetChannel(channel *discord.Channel) {
	ss.channelsMu.Lock()
	defer ss.channelsMu.Unlock()

	ss.Channels[channel.ID] = ss.ChannelToState(channel)

	for _, recipient := range channel.Recipients {
		ss.SetUser(recipient)
	}

	return
}

// RemoveChannel removes a channel from the cache.
func (ss *SandwichState) RemoveChannel(channelID discord.Snowflake) {
	ss.channelsMu.Lock()
	defer ss.channelsMu.RUnlock()

	delete(ss.Channels, channelID)

	return
}
