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
	guild = &discord.Guild{
		ID:              guildState.ID,
		Name:            guildState.Name,
		Icon:            guildState.Icon,
		IconHash:        guildState.IconHash,
		Splash:          guildState.Splash,
		DiscoverySplash: guildState.DiscoverySplash,

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
		StageInstances: make([]*discord.StageInstance, 0),
		Stickers:       make([]*discord.Sticker, 0),
	}

	for _, stageInstance := range guildState.StageInstances {
		stageInstance := stageInstance
		guild.StageInstances = append(guild.StageInstances, &stageInstance)
	}

	for _, sticker := range guildState.Stickers {
		sticker := sticker
		guild.Stickers = append(guild.Stickers, &sticker)
	}

	return guild
}

// GuildFromState converts from discord.Guild to structs.StateGuild, for storing in cache.
// Does not add Channels, Roles, Members and Emojis to state.
func (ss *SandwichState) GuildToState(guild *discord.Guild) (guildState *structs.StateGuild) {
	guildState = &structs.StateGuild{
		ID:              guild.ID,
		Name:            guild.Name,
		Icon:            guild.Icon,
		IconHash:        guild.IconHash,
		Splash:          guild.Splash,
		DiscoverySplash: guild.DiscoverySplash,

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
		StageInstances: make([]discord.StageInstance, 0),
		Stickers:       make([]discord.Sticker, 0),
	}

	for _, stageInstance := range guild.StageInstances {
		guildState.StageInstances = append(guildState.StageInstances, *stageInstance)
	}

	for _, sticker := range guild.Stickers {
		guildState.Stickers = append(guildState.Stickers, *sticker)
	}

	return guildState
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
		ss.SetGuildRole(guild.ID, role)
	}

	for _, channel := range guild.Channels {
		ss.SetGuildChannel(&guild.ID, channel)
	}

	for _, emoji := range guild.Emojis {
		ss.SetGuildEmoji(guild.ID, emoji)
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
	defer ss.guildsMu.Unlock()

	if _, ok := ss.Guilds[guildID]; ok {
		ss.RemoveAllGuildRoles(guildID)
		ss.RemoveAllGuildChannels(guildID)
		ss.RemoveAllGuildEmojis(guildID)
		ss.RemoveAllGuildMembers(guildID)
	}

	delete(ss.Guilds, guildID)

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
			ID: guildState.UserID,
		},
		Nick: guildState.Nick,

		Roles:        guildState.Roles,
		JoinedAt:     guildState.JoinedAt,
		PremiumSince: guildState.PremiumSince,
		Deaf:         guildState.Deaf,
		Mute:         guildState.Mute,
		Pending:      guildState.Pending,
		Permissions:  guildState.Permissions,
	}
}

// GuildMemberFromState converts from discord.GuildMember to structs.StateGuildMembers, for storing in cache.
// This does not add the user to the cache.
func (ss *SandwichState) GuildMemberToState(guild *discord.GuildMember) (guildState *structs.StateGuildMember) {
	return &structs.StateGuildMember{
		UserID: guild.User.ID,
		Nick:   guild.Nick,

		Roles:        guild.Roles,
		JoinedAt:     guild.JoinedAt,
		PremiumSince: guild.PremiumSince,
		Deaf:         guild.Deaf,
		Mute:         guild.Mute,
		Pending:      guild.Pending,
		Permissions:  guild.Permissions,
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

	guildMembers.MembersMu.Lock()
	defer guildMembers.MembersMu.Unlock()

	delete(guildMembers.Members, guildMemberID)

	return
}

// GetAllGuildMembers returns all guildMembers of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildMembers(guildID discord.Snowflake) (guildMembersList []*discord.GuildMember, ok bool) {
	ss.guildMembersMu.RLock()
	defer ss.guildMembersMu.RUnlock()

	guildMembers, ok := ss.GuildMembers[guildID]
	if !ok {
		return
	}

	guildMembers.MembersMu.RLock()
	defer guildMembers.MembersMu.RUnlock()

	for _, guildMember := range guildMembers.Members {
		guildMembersList = append(guildMembersList, ss.GuildMemberFromState(guildMember))
	}

	return
}

// RemoveAllGuildMembers removes all guildMembers of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildMembers(guildID discord.Snowflake) {
	ss.guildMembersMu.Lock()
	defer ss.guildMembersMu.Unlock()

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

// GetGuildRole returns the role with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildRole(guildID discord.Snowflake, roleID discord.Snowflake) (role *discord.Role, ok bool) {
	ss.guildRolesMu.RLock()
	defer ss.guildRolesMu.RUnlock()

	stateGuildRoles, ok := ss.GuildRoles[roleID]
	if !ok {
		return
	}

	stateGuildRoles.RolesMu.RLock()
	defer stateGuildRoles.RolesMu.RUnlock()

	stateGuildRole, ok := stateGuildRoles.Roles[roleID]
	if !ok {
		return
	}

	role = ss.RoleFromState(stateGuildRole)

	return
}

// SetGuildRole creates or updates a role entry in the cache.
func (ss *SandwichState) SetGuildRole(guildID discord.Snowflake, role *discord.Role) {
	ss.guildRolesMu.Lock()
	defer ss.guildRolesMu.Unlock()

	guildRoles, ok := ss.GuildRoles[guildID]
	if !ok {
		guildRoles = &structs.StateGuildRoles{
			RolesMu: sync.RWMutex{},
			Roles:   make(map[discord.Snowflake]*structs.StateRole),
		}

		ss.GuildRoles[guildID] = guildRoles
	}

	guildRoles.RolesMu.Lock()
	defer guildRoles.RolesMu.Unlock()

	guildRoles.Roles[role.ID] = ss.RoleToState(role)

	return
}

// RemoveGuildRole removes a role from the cache.
func (ss *SandwichState) RemoveGuildRole(guildID discord.Snowflake, roleID discord.Snowflake) {
	ss.guildRolesMu.RLock()
	defer ss.guildRolesMu.RUnlock()

	guildRoles, ok := ss.GuildRoles[guildID]
	if !ok {
		return
	}

	guildRoles.RolesMu.Lock()
	defer guildRoles.RolesMu.Unlock()

	delete(guildRoles.Roles, roleID)

	return
}

// GetAllGuildRoles returns all guildRoles of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildRoles(guildID discord.Snowflake) (guildRolesList []*discord.Role, ok bool) {
	ss.guildRolesMu.RLock()
	defer ss.guildRolesMu.RUnlock()

	guildRoles, ok := ss.GuildRoles[guildID]
	if !ok {
		return
	}

	guildRoles.RolesMu.RLock()
	guildRoles.RolesMu.RUnlock()

	for _, guildRole := range guildRoles.Roles {
		guildRolesList = append(guildRolesList, ss.RoleFromState(guildRole))
	}

	return
}

// RemoveGuildRoles removes all guild roles of a specifi guild from the cache.
func (ss *SandwichState) RemoveAllGuildRoles(guildID discord.Snowflake) {
	ss.guildRolesMu.Lock()
	defer ss.guildRolesMu.Unlock()

	delete(ss.GuildRoles, guildID)

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
			ID: guildState.UserID,
		},
		RequireColons: guildState.RequireColons,
		Managed:       &guildState.Managed,
		Animated:      &guildState.Animated,
		Available:     &guildState.Available,
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
		Managed:       *guild.Managed,
		Animated:      *guild.Animated,
		Available:     *guild.Available,
	}

	if guild.User != nil {
		guildState.UserID = guild.User.ID
	}

	return guildState
}

// GetGuildEmoji returns the emoji with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildEmoji(guildID discord.Snowflake, emojiID discord.Snowflake) (guildEmoji *discord.Emoji, ok bool) {
	ss.guildEmojisMu.RLock()
	defer ss.guildEmojisMu.RUnlock()

	guildEmojis, ok := ss.GuildEmojis[guildID]
	if !ok {
		return
	}

	guildEmojis.EmojisMu.RLock()
	defer guildEmojis.EmojisMu.RUnlock()

	stateGuildEmoji, ok := guildEmojis.Emojis[emojiID]
	if !ok {
		return
	}

	guildEmoji = ss.EmojiFromState(stateGuildEmoji)

	user, ok := ss.GetUser(guildEmoji.User.ID)
	if ok {
		guildEmoji.User = user
	}

	return
}

// SetGuildEmoji creates or updates a emoji entry in the cache. Adds user in user object to cache.
func (ss *SandwichState) SetGuildEmoji(guildID discord.Snowflake, emoji *discord.Emoji) {
	ss.guildEmojisMu.Lock()
	defer ss.guildEmojisMu.Unlock()

	guildEmojis, ok := ss.GuildEmojis[guildID]
	if !ok {
		guildEmojis = &structs.StateGuildEmojis{
			EmojisMu: sync.RWMutex{},
			Emojis:   make(map[discord.Snowflake]*structs.StateEmoji),
		}

		ss.GuildEmojis[guildID] = guildEmojis
	}

	guildEmojis.EmojisMu.Lock()
	defer guildEmojis.EmojisMu.Unlock()

	guildEmojis.Emojis[emoji.ID] = ss.EmojiToState(emoji)

	if emoji.User != nil {
		ss.SetUser(emoji.User)
	}

	return
}

// RemoveGuildEmoji removes a emoji from the cache.
func (ss *SandwichState) RemoveGuildEmoji(guildID discord.Snowflake, emojiID discord.Snowflake) {
	ss.guildEmojisMu.RLock()
	defer ss.guildEmojisMu.RUnlock()

	guildEmojis, ok := ss.GuildEmojis[guildID]
	if !ok {
		return
	}

	guildEmojis.EmojisMu.Lock()
	defer guildEmojis.EmojisMu.Unlock()

	delete(guildEmojis.Emojis, emojiID)

	return
}

// GetAllGuildEmojis returns all guildEmojis on a specific guild from the cache.
func (ss *SandwichState) GetAllGuildEmojis(guildID discord.Snowflake) (guildEmojisList []*discord.Emoji, ok bool) {
	ss.guildEmojisMu.RLock()
	defer ss.guildEmojisMu.RUnlock()

	guildEmojis, ok := ss.GuildEmojis[guildID]
	if !ok {
		return
	}

	guildEmojis.EmojisMu.RLock()
	defer guildEmojis.EmojisMu.RUnlock()

	for _, guildEmoji := range guildEmojis.Emojis {
		guildEmojisList = append(guildEmojisList, ss.EmojiFromState(guildEmoji))
	}

	return
}

// RemoveGuildEmojis removes all guildEmojis of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildEmojis(guildID discord.Snowflake) {
	ss.guildEmojisMu.Lock()
	defer ss.guildEmojisMu.Unlock()

	delete(ss.GuildEmojis, guildID)

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
	defer ss.usersMu.Unlock()

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
		PermissionOverwrites: make([]*discord.ChannelOverwrite, 0),
		Name:                 guildState.Name,
		Topic:                guildState.Topic,
		NSFW:                 guildState.NSFW,
		// LastMessageID:        guildState.LastMessageID,
		Bitrate:          guildState.Bitrate,
		UserLimit:        guildState.UserLimit,
		RateLimitPerUser: guildState.RateLimitPerUser,
		Recipients:       make([]discord.User, 0),
		Icon:             guildState.Icon,
		OwnerID:          guildState.OwnerID,
		// ApplicationID:        guildState.ApplicationID,
		ParentID: guildState.ParentID,
		// LastPinTimestamp:     guildState.LastPinTimestamp,

		// RTCRegion: guildState.RTCRegion,
		// VideoQualityMode: guildState.VideoQualityMode,

		// MessageCount:               guildState.MessageCount,
		// MemberCount:                guildState.MemberCount,
		ThreadMetadata: guildState.ThreadMetadata,
		// ThreadMember:               guildState.ThreadMember,
		// DefaultAutoArchiveDuration: guildState.DefaultAutoArchiveDuration,

		Permissions: guildState.Permissions,
	}

	for _, permissionOverride := range guildState.PermissionOverwrites {
		permissionOverride := permissionOverride
		guild.PermissionOverwrites = append(guild.PermissionOverwrites, &permissionOverride)
	}

	// for _, recepientID := range guildState.RecipientIDs {
	// 	guild.Recipients = append(guild.Recipients, &discord.User{
	// 		ID: *recepientID,
	// 	})
	// }

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
		PermissionOverwrites: make([]discord.ChannelOverwrite, 0),
		Name:                 guild.Name,
		Topic:                guild.Topic,
		NSFW:                 guild.NSFW,
		// LastMessageID:        guild.LastMessageID,
		Bitrate:          guild.Bitrate,
		UserLimit:        guild.UserLimit,
		RateLimitPerUser: guild.RateLimitPerUser,
		// RecipientIDs:         make([]*discord.Snowflake, 0),
		Icon:    guild.Icon,
		OwnerID: guild.OwnerID,
		// ApplicationID:        guild.ApplicationID,
		ParentID: guild.ParentID,
		// LastPinTimestamp:     guild.LastPinTimestamp,

		// RTCRegion: guild.RTCRegion,
		// VideoQualityMode: guild.VideoQualityMode,

		// MessageCount:               guild.MessageCount,
		// MemberCount:                guild.MemberCount,
		ThreadMetadata: guild.ThreadMetadata,
		// ThreadMember:               guild.ThreadMember,
		// DefaultAutoArchiveDuration: guild.DefaultAutoArchiveDuration,

		Permissions: guild.Permissions,
	}

	for _, permissionOverride := range guild.PermissionOverwrites {
		permissionOverride := permissionOverride
		guildState.PermissionOverwrites = append(guildState.PermissionOverwrites, *permissionOverride)
	}

	// for _, recipient := range guild.Recipients {
	// 	guildState.RecipientIDs = append(guildState.RecipientIDs, &recipient.ID)
	// }

	return guildState
}

// GetGuildChannel returns the channel with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildChannel(guildIDPtr *discord.Snowflake, channelID discord.Snowflake) (guildChannel *discord.Channel, ok bool) {
	ss.guildChannelsMu.RLock()
	defer ss.guildChannelsMu.RUnlock()

	var guildID discord.Snowflake

	if guildIDPtr != nil {
		guildID = *guildIDPtr
	} else {
		guildID = discord.Snowflake(0)
	}

	stateChannels, ok := ss.GuildChannels[guildID]
	if !ok {
		return
	}

	stateChannels.ChannelsMu.RLock()
	defer stateChannels.ChannelsMu.RUnlock()

	stateGuildChannel, ok := stateChannels.Channels[channelID]
	if !ok {
		return
	}

	guildChannel = ss.ChannelFromState(stateGuildChannel)

	for _, recipient := range guildChannel.Recipients {
		recipientUser, ok := ss.GetUser(recipient.ID)
		if ok {
			recipient = *recipientUser
		}
	}

	return guildChannel, ok
}

// SetGuildChannel creates or updates a channel entry in the cache.
func (ss *SandwichState) SetGuildChannel(guildIDPtr *discord.Snowflake, channel *discord.Channel) {
	ss.guildChannelsMu.Lock()
	defer ss.guildChannelsMu.Unlock()

	var guildID discord.Snowflake

	if guildIDPtr != nil {
		guildID = *guildIDPtr
	} else {
		guildID = discord.Snowflake(0)
	}

	guildChannels, ok := ss.GuildChannels[guildID]
	if !ok {
		guildChannels = &structs.StateGuildChannels{
			ChannelsMu: sync.RWMutex{},
			Channels:   make(map[discord.Snowflake]*structs.StateChannel),
		}

		ss.GuildChannels[guildID] = guildChannels
	}

	guildChannels.ChannelsMu.Lock()
	defer guildChannels.ChannelsMu.Unlock()

	guildChannels.Channels[channel.ID] = ss.ChannelToState(channel)

	for _, recipient := range channel.Recipients {
		recipient := recipient
		ss.SetUser(&recipient)
	}

	return
}

// RemoveGuildChannel removes a channel from the cache.
func (ss *SandwichState) RemoveGuildChannel(guildIDPtr *discord.Snowflake, channelID discord.Snowflake) {
	ss.guildChannelsMu.RLock()
	defer ss.guildChannelsMu.RUnlock()

	var guildID discord.Snowflake

	if guildIDPtr != nil {
		guildID = *guildIDPtr
	} else {
		guildID = discord.Snowflake(0)
	}

	guildChannels, ok := ss.GuildChannels[guildID]
	if !ok {
		return
	}

	guildChannels.ChannelsMu.Lock()
	defer guildChannels.ChannelsMu.Unlock()

	delete(guildChannels.Channels, channelID)

	return
}

// GetAllGuildChannels returns all guildChannels of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildChannels(guildID discord.Snowflake) (guildChannelsList []*discord.Channel, ok bool) {
	ss.guildChannelsMu.RLock()
	defer ss.guildChannelsMu.RUnlock()

	guildChannels, ok := ss.GuildChannels[guildID]
	if !ok {
		return
	}

	guildChannels.ChannelsMu.RLock()
	defer guildChannels.ChannelsMu.RUnlock()

	for _, guildRole := range guildChannels.Channels {
		guildChannelsList = append(guildChannelsList, ss.ChannelFromState(guildRole))
	}

	return
}

// RemoveAllGuildChannels removes all guildChannels of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildChannels(guildID discord.Snowflake) {
	ss.guildEmojisMu.Lock()
	defer ss.guildEmojisMu.Unlock()

	delete(ss.GuildEmojis, guildID)

	return
}

// GetUserMutualGuilds returns a list of snowflakes of mutual guilds a member is seen on.
func (ss *SandwichState) GetUserMutualGuilds(userID discord.Snowflake) (guildIDs []discord.Snowflake, ok bool) {
	ss.mutualsMu.RLock()
	defer ss.mutualsMu.RUnlock()

	mutualGuilds, ok := ss.Mutuals[userID]
	if !ok {
		return
	}

	mutualGuilds.GuildsMu.RLock()
	defer mutualGuilds.GuildsMu.RUnlock()

	for guildID := range mutualGuilds.Guilds {
		guildIDs = append(guildIDs, guildID)
	}

	return
}

// AddUserMutualGuild adds a mutual guild to a user.
func (ss *SandwichState) AddUserMutualGuild(userID discord.Snowflake, guildID discord.Snowflake) {
	ss.mutualsMu.Lock()
	defer ss.mutualsMu.Unlock()

	mutualGuilds, ok := ss.Mutuals[userID]
	if !ok {
		mutualGuilds = &structs.StateMutualGuilds{
			GuildsMu: sync.RWMutex{},
			Guilds:   make(map[discord.Snowflake]bool),
		}

		ss.Mutuals[userID] = mutualGuilds
	}

	mutualGuilds.GuildsMu.Lock()
	defer mutualGuilds.GuildsMu.Unlock()

	mutualGuilds.Guilds[guildID] = true

	return
}

// RemoveUserMutualGuild removes a mutual guild from a user.
func (ss *SandwichState) RemoveUserMutualGuild(userID discord.Snowflake, guildID discord.Snowflake) {
	ss.mutualsMu.RLock()
	defer ss.mutualsMu.RUnlock()

	mutualGuilds, ok := ss.Mutuals[userID]
	if !ok {
		return
	}

	mutualGuilds.GuildsMu.Lock()
	defer mutualGuilds.GuildsMu.Unlock()

	delete(mutualGuilds.Guilds, guildID)

	return
}
