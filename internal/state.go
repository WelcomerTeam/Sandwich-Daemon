package internal

import (
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
)

//
// Guild Operations
//

// GuildFromState converts the structs.StateGuild into a discord.Guild, for use within the application.
// Channels, Roles, Members and Emoji lists will not be populated.
func (ss *SandwichState) GuildFromState(guildState *sandwich_structs.StateGuild) (guild *discord.Guild) {
	guild = &discord.Guild{
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

		PremiumTier:               guildState.PremiumTier,
		PremiumSubscriptionCount:  guildState.PremiumSubscriptionCount,
		PreferredLocale:           guildState.PreferredLocale,
		PublicUpdatesChannelID:    guildState.PublicUpdatesChannelID,
		MaxVideoChannelUsers:      guildState.MaxVideoChannelUsers,
		ApproximateMemberCount:    guildState.ApproximateMemberCount,
		ApproximatePresenceCount:  guildState.ApproximatePresenceCount,
		NSFWLevel:                 guildState.NSFWLevel,
		PremiumProgressBarEnabled: guildState.PremiumProgressBarEnabled,

		Features:             guildState.Features,
		StageInstances:       make([]*discord.StageInstance, 0, len(guildState.StageInstances)),
		Stickers:             make([]*discord.Sticker, 0, len(guildState.Stickers)),
		GuildScheduledEvents: make([]*discord.ScheduledEvent, 0, len(guildState.GuildScheduledEvents)),
	}

	for _, stageInstance := range guildState.StageInstances {
		stageInstance := stageInstance
		guild.StageInstances = append(guild.StageInstances, &stageInstance)
	}

	for _, sticker := range guildState.Stickers {
		sticker := sticker
		guild.Stickers = append(guild.Stickers, &sticker)
	}

	for _, scheduledEvent := range guildState.GuildScheduledEvents {
		scheduledEvent := scheduledEvent
		guild.GuildScheduledEvents = append(guild.GuildScheduledEvents, &scheduledEvent)
	}

	return guild
}

// GuildFromState converts from discord.Guild to structs.StateGuild, for storing in cache.
// Does not add Channels, Roles, Members and Emojis to state.
func (ss *SandwichState) GuildToState(guild *discord.Guild) (guildState *sandwich_structs.StateGuild) {
	guildState = &sandwich_structs.StateGuild{
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
	stateGuild, ok := ss.Guilds[guildID]
	ss.guildsMu.RUnlock()

	if !ok {
		return
	}

	guild = ss.GuildFromState(stateGuild)

	return
}

// SetGuild creates or updates a guild entry in the cache.
func (ss *SandwichState) SetGuild(ctx *StateCtx, guild *discord.Guild) {
	ctx.ShardGroup.guildsMu.Lock()
	ctx.ShardGroup.Guilds[guild.ID] = true
	ctx.ShardGroup.guildsMu.Unlock()

	ss.guildsMu.Lock()
	ss.Guilds[guild.ID] = ss.GuildToState(guild)
	ss.guildsMu.Unlock()

	for _, role := range guild.Roles {
		ss.SetGuildRole(ctx, guild.ID, role)
	}

	for _, channel := range guild.Channels {
		ss.SetGuildChannel(ctx, &guild.ID, channel)
	}

	for _, emoji := range guild.Emojis {
		ss.SetGuildEmoji(ctx, guild.ID, emoji)
	}

	for _, member := range guild.Members {
		ss.SetGuildMember(ctx, guild.ID, member)
	}

	for _, voiceState := range guild.VoiceStates {
		voiceState.GuildID = &guild.ID
		ss.UpdateVoiceState(ctx, *voiceState)
	}
}

// RemoveGuild removes a guild from the cache.
func (ss *SandwichState) RemoveGuild(ctx *StateCtx, guildID discord.Snowflake) {
	ss.guildsMu.Lock()
	delete(ss.Guilds, guildID)
	ss.guildsMu.Unlock()

	if !ctx.Stateless {
		ctx.ShardGroup.guildsMu.Lock()
		delete(ctx.ShardGroup.Guilds, guildID)
		ctx.ShardGroup.guildsMu.Unlock()
	}

	ss.RemoveAllGuildRoles(guildID)
	ss.RemoveAllGuildChannels(guildID)
	ss.RemoveAllGuildEmojis(guildID)
	ss.RemoveAllGuildMembers(guildID)
}

//
// GuildMember Operations
//

// GuildMemberFromState converts the structs.StateGuildMembers into a discord.GuildMember,
// for use within the application.
// This will not populate the user object from cache, it will be an empty object with only an ID.
func (ss *SandwichState) GuildMemberFromState(guildMemberState *sandwich_structs.StateGuildMember) (guildMember *discord.GuildMember) {
	return &discord.GuildMember{
		User: &discord.User{
			ID: guildMemberState.UserID,
		},
		Nick:                       guildMemberState.Nick,
		Avatar:                     guildMemberState.Avatar,
		Roles:                      guildMemberState.Roles,
		JoinedAt:                   guildMemberState.JoinedAt,
		PremiumSince:               guildMemberState.PremiumSince,
		Deaf:                       guildMemberState.Deaf,
		Mute:                       guildMemberState.Mute,
		Pending:                    guildMemberState.Pending,
		Permissions:                guildMemberState.Permissions,
		CommunicationDisabledUntil: guildMemberState.CommunicationDisabledUntil,
	}
}

// GuildMemberFromState converts from discord.GuildMember to structs.StateGuildMembers, for storing in cache.
// This does not add the user to the cache.
func (ss *SandwichState) GuildMemberToState(guildMember *discord.GuildMember) (guildMemberState *sandwich_structs.StateGuildMember) {
	return &sandwich_structs.StateGuildMember{
		UserID:                     guildMember.User.ID,
		Nick:                       guildMember.Nick,
		Avatar:                     guildMember.Avatar,
		Roles:                      guildMember.Roles,
		JoinedAt:                   guildMember.JoinedAt,
		PremiumSince:               guildMember.PremiumSince,
		Deaf:                       guildMember.Deaf,
		Mute:                       guildMember.Mute,
		Pending:                    guildMember.Pending,
		Permissions:                guildMember.Permissions,
		CommunicationDisabledUntil: guildMember.CommunicationDisabledUntil,
	}
}

// GetGuildMember returns the guildMember with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildMember(guildID discord.Snowflake, guildMemberID discord.Snowflake) (guildMember *discord.GuildMember, ok bool) {
	ss.guildMembersMu.RLock()
	guildMembers, ok := ss.GuildMembers[guildID]
	ss.guildMembersMu.RUnlock()

	if !ok {
		return
	}

	guildMembers.MembersMu.RLock()
	stateGuildMember, ok := guildMembers.Members[guildMemberID]
	guildMembers.MembersMu.RUnlock()

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
func (ss *SandwichState) SetGuildMember(ctx *StateCtx, guildID discord.Snowflake, guildMember *discord.GuildMember) {
	// We will always cache the guild member of the bot that receives this event.
	if !ctx.CacheMembers && guildMember.User.ID != ctx.Manager.User.ID {
		return
	}

	ss.guildMembersMu.RLock()
	guildMembers, ok := ss.GuildMembers[guildID]
	ss.guildMembersMu.RUnlock()

	if !ok {
		guildMembers = &sandwich_structs.StateGuildMembers{
			MembersMu: sync.RWMutex{},
			Members:   make(map[discord.Snowflake]*sandwich_structs.StateGuildMember),
		}

		ss.guildMembersMu.Lock()
		ss.GuildMembers[guildID] = guildMembers
		ss.guildMembersMu.Unlock()
	}

	guildMembers.MembersMu.Lock()
	guildMembers.Members[guildMember.User.ID] = ss.GuildMemberToState(guildMember)
	guildMembers.MembersMu.Unlock()

	ss.SetUser(ctx, guildMember.User)
}

// RemoveGuildMember removes a guildMember from the cache.
func (ss *SandwichState) RemoveGuildMember(guildID discord.Snowflake, guildMemberID discord.Snowflake) {
	ss.guildMembersMu.RLock()
	guildMembers, ok := ss.GuildMembers[guildID]
	ss.guildMembersMu.RUnlock()

	if !ok {
		return
	}

	guildMembers.MembersMu.Lock()
	delete(guildMembers.Members, guildMemberID)
	guildMembers.MembersMu.Unlock()
}

// GetAllGuildMembers returns all guildMembers of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildMembers(guildID discord.Snowflake) (guildMembersList []*discord.GuildMember, ok bool) {
	ss.guildMembersMu.RLock()
	guildMembers, ok := ss.GuildMembers[guildID]
	ss.guildMembersMu.RUnlock()

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
	delete(ss.GuildMembers, guildID)
	ss.guildMembersMu.Unlock()
}

//
// Role Operations
//

// RoleFromState converts the structs.StateRole into a discord.Role, for use within the application.
func (ss *SandwichState) RoleFromState(guildState *sandwich_structs.StateRole) (guild *discord.Role) {
	return &discord.Role{
		ID:           guildState.ID,
		Name:         guildState.Name,
		Color:        guildState.Color,
		Hoist:        guildState.Hoist,
		Icon:         guildState.Icon,
		UnicodeEmoji: guildState.UnicodeEmoji,
		Position:     guildState.Position,
		Permissions:  guildState.Permissions,
		Managed:      guildState.Managed,
		Mentionable:  guildState.Mentionable,
		Tags:         guildState.Tags,
	}
}

// RoleFromState converts from discord.Role to structs.StateRole, for storing in cache.
func (ss *SandwichState) RoleToState(guild *discord.Role) (guildState *sandwich_structs.StateRole) {
	return &sandwich_structs.StateRole{
		ID:           guild.ID,
		Name:         guild.Name,
		Color:        guild.Color,
		Hoist:        guild.Hoist,
		Icon:         guild.Icon,
		UnicodeEmoji: guild.UnicodeEmoji,
		Position:     guild.Position,
		Permissions:  guild.Permissions,
		Managed:      guild.Managed,
		Mentionable:  guild.Mentionable,
		Tags:         guild.Tags,
	}
}

// GetGuildRole returns the role with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildRole(guildID discord.Snowflake, roleID discord.Snowflake) (role *discord.Role, ok bool) {
	ss.guildRolesMu.RLock()
	stateGuildRoles, ok := ss.GuildRoles[roleID]
	ss.guildRolesMu.RUnlock()

	if !ok {
		return
	}

	stateGuildRoles.RolesMu.RLock()
	stateGuildRole, ok := stateGuildRoles.Roles[roleID]
	stateGuildRoles.RolesMu.RUnlock()

	if !ok {
		return
	}

	role = ss.RoleFromState(stateGuildRole)

	return
}

// SetGuildRole creates or updates a role entry in the cache.
func (ss *SandwichState) SetGuildRole(ctx *StateCtx, guildID discord.Snowflake, role *discord.Role) {
	ss.guildRolesMu.RLock()
	guildRoles, ok := ss.GuildRoles[guildID]
	ss.guildRolesMu.RUnlock()

	if !ok {
		guildRoles = &sandwich_structs.StateGuildRoles{
			RolesMu: sync.RWMutex{},
			Roles:   make(map[discord.Snowflake]*sandwich_structs.StateRole),
		}

		ss.guildRolesMu.Lock()
		ss.GuildRoles[guildID] = guildRoles
		ss.guildRolesMu.Unlock()
	}

	guildRoles.RolesMu.Lock()
	guildRoles.Roles[role.ID] = ss.RoleToState(role)
	guildRoles.RolesMu.Unlock()
}

// RemoveGuildRole removes a role from the cache.
func (ss *SandwichState) RemoveGuildRole(guildID discord.Snowflake, roleID discord.Snowflake) {
	ss.guildRolesMu.RLock()
	guildRoles, ok := ss.GuildRoles[guildID]
	ss.guildRolesMu.RUnlock()

	if !ok {
		return
	}

	guildRoles.RolesMu.Lock()
	delete(guildRoles.Roles, roleID)
	guildRoles.RolesMu.Unlock()
}

// GetAllGuildRoles returns all guildRoles of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildRoles(guildID discord.Snowflake) (guildRolesList []*discord.Role, ok bool) {
	ss.guildRolesMu.RLock()
	guildRoles, ok := ss.GuildRoles[guildID]
	ss.guildRolesMu.RUnlock()

	if !ok {
		return
	}

	guildRoles.RolesMu.RLock()
	defer guildRoles.RolesMu.RUnlock()

	for _, guildRole := range guildRoles.Roles {
		guildRolesList = append(guildRolesList, ss.RoleFromState(guildRole))
	}

	return
}

// RemoveGuildRoles removes all guild roles of a specifi guild from the cache.
func (ss *SandwichState) RemoveAllGuildRoles(guildID discord.Snowflake) {
	ss.guildRolesMu.Lock()
	delete(ss.GuildRoles, guildID)
	ss.guildRolesMu.Unlock()
}

//
// Emoji Operations
//

// EmojiFromState converts the structs.StateEmoji into a discord.Emoji, for use within the application.
func (ss *SandwichState) EmojiFromState(guildState *sandwich_structs.StateEmoji) (guild *discord.Emoji) {
	return &discord.Emoji{
		ID:    guildState.ID,
		Name:  guildState.Name,
		Roles: guildState.Roles,
		User: &discord.User{
			ID: guildState.UserID,
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
func (ss *SandwichState) EmojiToState(emoji *discord.Emoji) (guildState *sandwich_structs.StateEmoji) {
	guildState = &sandwich_structs.StateEmoji{
		ID:            emoji.ID,
		Name:          emoji.Name,
		Roles:         emoji.Roles,
		RequireColons: emoji.RequireColons,
		Managed:       emoji.Managed,
		Animated:      emoji.Animated,
		Available:     emoji.Available,
	}

	if emoji.User != nil {
		guildState.UserID = emoji.User.ID
	}

	return guildState
}

// GetGuildEmoji returns the emoji with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildEmoji(guildID discord.Snowflake, emojiID discord.Snowflake) (guildEmoji *discord.Emoji, ok bool) {
	ss.guildEmojisMu.RLock()
	guildEmojis, ok := ss.GuildEmojis[guildID]
	ss.guildEmojisMu.RUnlock()

	if !ok {
		return
	}

	guildEmojis.EmojisMu.RLock()
	stateGuildEmoji, ok := guildEmojis.Emojis[emojiID]
	guildEmojis.EmojisMu.RUnlock()

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
func (ss *SandwichState) SetGuildEmoji(ctx *StateCtx, guildID discord.Snowflake, emoji *discord.Emoji) {
	ss.guildEmojisMu.RLock()
	guildEmojis, ok := ss.GuildEmojis[guildID]
	ss.guildEmojisMu.RUnlock()

	if !ok {
		guildEmojis = &sandwich_structs.StateGuildEmojis{
			EmojisMu: sync.RWMutex{},
			Emojis:   make(map[discord.Snowflake]*sandwich_structs.StateEmoji),
		}

		ss.guildEmojisMu.Lock()
		ss.GuildEmojis[guildID] = guildEmojis
		ss.guildEmojisMu.Unlock()
	}

	guildEmojis.EmojisMu.Lock()
	guildEmojis.Emojis[emoji.ID] = ss.EmojiToState(emoji)
	guildEmojis.EmojisMu.Unlock()

	if emoji.User != nil {
		ss.SetUser(ctx, emoji.User)
	}
}

// RemoveGuildEmoji removes a emoji from the cache.
func (ss *SandwichState) RemoveGuildEmoji(guildID discord.Snowflake, emojiID discord.Snowflake) {
	ss.guildEmojisMu.RLock()
	guildEmojis, ok := ss.GuildEmojis[guildID]
	ss.guildEmojisMu.RUnlock()

	if !ok {
		return
	}

	guildEmojis.EmojisMu.Lock()
	delete(guildEmojis.Emojis, emojiID)
	guildEmojis.EmojisMu.Unlock()
}

// GetAllGuildEmojis returns all guildEmojis on a specific guild from the cache.
func (ss *SandwichState) GetAllGuildEmojis(guildID discord.Snowflake) (guildEmojisList []*discord.Emoji, ok bool) {
	ss.guildEmojisMu.RLock()
	guildEmojis, ok := ss.GuildEmojis[guildID]
	ss.guildEmojisMu.RUnlock()

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
	delete(ss.GuildEmojis, guildID)
	ss.guildEmojisMu.Unlock()
}

//
// User Operations
//

// UserFromState converts the structs.StateUser into a discord.User, for use within the application.
func (ss *SandwichState) UserFromState(userState *sandwich_structs.StateUser) (user *discord.User) {
	return &discord.User{
		ID:            userState.ID,
		Username:      userState.Username,
		Discriminator: userState.Discriminator,
		GlobalName:    userState.GlobalName,
		Avatar:        userState.Avatar,
		Bot:           userState.Bot,
		System:        userState.System,
		MFAEnabled:    userState.MFAEnabled,
		Banner:        userState.Banner,
		AccentColor:   userState.AccentColour,
		Locale:        userState.Locale,
		Verified:      userState.Verified,
		Email:         userState.Email,
		Flags:         userState.Flags,
		PremiumType:   userState.PremiumType,
		PublicFlags:   userState.PublicFlags,
		DMChannelID:   userState.DMChannelID,
	}
}

// UserFromState converts from discord.User to structs.StateUser, for storing in cache.
func (ss *SandwichState) UserToState(user *discord.User) (userState *sandwich_structs.StateUser) {
	return &sandwich_structs.StateUser{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		GlobalName:    user.GlobalName,
		Avatar:        user.Avatar,
		Bot:           user.Bot,
		System:        user.System,
		MFAEnabled:    user.MFAEnabled,
		Banner:        user.Banner,
		AccentColour:  user.AccentColor,
		Locale:        user.Locale,
		Verified:      user.Verified,
		Email:         user.Email,
		Flags:         user.Flags,
		PremiumType:   user.PremiumType,
		PublicFlags:   user.PublicFlags,
		DMChannelID:   user.DMChannelID,
	}
}

// GetUser returns the user with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetUser(userID discord.Snowflake) (user *discord.User, ok bool) {
	ss.usersMu.RLock()
	stateUser, ok := ss.Users[userID]
	ss.usersMu.RUnlock()

	if !ok {
		return
	}

	user = ss.UserFromState(stateUser)

	return
}

// SetUser creates or updates a user entry in the cache.
func (ss *SandwichState) SetUser(ctx *StateCtx, user *discord.User) {
	// We will always cache the user of the bot that receives this event.
	if !ctx.CacheUsers && user.ID != ctx.Manager.User.ID {
		return
	}

	ss.usersMu.Lock()
	ss.Users[user.ID] = ss.UserToState(user)
	ss.usersMu.Unlock()
}

// RemoveUser removes a user from the cache.
func (ss *SandwichState) RemoveUser(userID discord.Snowflake) {
	ss.usersMu.Lock()
	delete(ss.Users, userID)
	ss.usersMu.Unlock()
}

//
// Channel Operations
//

// ChannelFromState converts the structs.StateChannel into a discord.Channel, for use within the application.
// This will not populate the recipient user object from cache.
func (ss *SandwichState) ChannelFromState(guildChannelState *sandwich_structs.StateChannel) (guildChannel *discord.Channel) {
	guildChannel = &discord.Channel{
		ID:                         guildChannelState.ID,
		Type:                       guildChannelState.Type,
		GuildID:                    guildChannelState.GuildID,
		Position:                   guildChannelState.Position,
		PermissionOverwrites:       make([]*discord.ChannelOverwrite, 0, len(guildChannelState.PermissionOverwrites)),
		Name:                       guildChannelState.Name,
		Topic:                      guildChannelState.Topic,
		NSFW:                       guildChannelState.NSFW,
		LastMessageID:              guildChannelState.LastMessageID,
		Bitrate:                    guildChannelState.Bitrate,
		UserLimit:                  guildChannelState.UserLimit,
		RateLimitPerUser:           guildChannelState.RateLimitPerUser,
		Recipients:                 make([]*discord.User, 0, len(guildChannelState.Recipients)),
		Icon:                       guildChannelState.Icon,
		OwnerID:                    guildChannelState.OwnerID,
		ApplicationID:              guildChannelState.ApplicationID,
		ParentID:                   guildChannelState.ParentID,
		LastPinTimestamp:           guildChannelState.LastPinTimestamp,
		RTCRegion:                  guildChannelState.RTCRegion,
		VideoQualityMode:           guildChannelState.VideoQualityMode,
		MessageCount:               guildChannelState.MessageCount,
		MemberCount:                guildChannelState.MemberCount,
		ThreadMetadata:             guildChannelState.ThreadMetadata,
		ThreadMember:               guildChannelState.ThreadMember,
		DefaultAutoArchiveDuration: guildChannelState.DefaultAutoArchiveDuration,
		Permissions:                guildChannelState.Permissions,
	}

	for _, permissionOverride := range guildChannelState.PermissionOverwrites {
		permissionOverride := permissionOverride
		guildChannel.PermissionOverwrites = append(guildChannel.PermissionOverwrites, &permissionOverride)
	}

	for _, recipientID := range guildChannelState.Recipients {
		guildChannel.Recipients = append(guildChannel.Recipients, &discord.User{
			ID: recipientID,
		})
	}

	return guildChannel
}

// ChannelFromState converts from discord.Channel to structs.StateChannel, for storing in cache.
// This does not add the recipients to the cache.
func (ss *SandwichState) ChannelToState(guildChannel *discord.Channel) (guildChannelState *sandwich_structs.StateChannel) {
	guildChannelState = &sandwich_structs.StateChannel{
		ID:                   guildChannel.ID,
		Type:                 guildChannel.Type,
		GuildID:              guildChannel.GuildID,
		Position:             guildChannel.Position,
		PermissionOverwrites: make([]discord.ChannelOverwrite, 0),
		Name:                 guildChannel.Name,
		Topic:                guildChannel.Topic,
		NSFW:                 guildChannel.NSFW,
		LastMessageID:        guildChannel.LastMessageID,
		Bitrate:              guildChannel.Bitrate,
		UserLimit:            guildChannel.UserLimit,
		RateLimitPerUser:     guildChannel.RateLimitPerUser,
		Recipients:           make([]discord.Snowflake, 0, len(guildChannel.Recipients)),
		Icon:                 guildChannel.Icon,
		OwnerID:              guildChannel.OwnerID,
		ApplicationID:        guildChannel.ApplicationID,
		ParentID:             guildChannel.ParentID,
		LastPinTimestamp:     guildChannel.LastPinTimestamp,

		RTCRegion:        guildChannel.RTCRegion,
		VideoQualityMode: guildChannel.VideoQualityMode,

		MessageCount:               guildChannel.MessageCount,
		MemberCount:                guildChannel.MemberCount,
		ThreadMetadata:             guildChannel.ThreadMetadata,
		ThreadMember:               guildChannel.ThreadMember,
		DefaultAutoArchiveDuration: guildChannel.DefaultAutoArchiveDuration,

		Permissions: guildChannel.Permissions,
	}

	for _, permissionOverride := range guildChannel.PermissionOverwrites {
		permissionOverride := permissionOverride
		guildChannelState.PermissionOverwrites = append(guildChannelState.PermissionOverwrites, *permissionOverride)
	}

	for _, recipient := range guildChannel.Recipients {
		guildChannelState.Recipients = append(guildChannelState.Recipients, recipient.ID)
	}

	return guildChannelState
}

// GetGuildChannel returns the channel with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildChannel(guildIDPtr *discord.Snowflake, channelID discord.Snowflake) (guildChannel *discord.Channel, ok bool) {
	var guildID discord.Snowflake

	if guildIDPtr != nil {
		guildID = *guildIDPtr
	} else {
		guildID = discord.Snowflake(0)
	}

	ss.guildChannelsMu.RLock()
	stateChannels, ok := ss.GuildChannels[guildID]
	ss.guildChannelsMu.RUnlock()

	if !ok {
		return guildChannel, false
	}

	stateChannels.ChannelsMu.RLock()
	defer stateChannels.ChannelsMu.RUnlock()

	stateGuildChannel, ok := stateChannels.Channels[channelID]
	if !ok {
		return guildChannel, false
	}

	guildChannel = ss.ChannelFromState(stateGuildChannel)

	newRecipients := make([]*discord.User, 0, len(guildChannel.Recipients))

	for _, recipient := range guildChannel.Recipients {
		recipientUser, ok := ss.GetUser(recipient.ID)
		if ok {
			recipient = recipientUser
		}

		newRecipients = append(newRecipients, recipient)
	}

	guildChannel.Recipients = newRecipients

	return guildChannel, ok
}

// SetGuildChannel creates or updates a channel entry in the cache.
func (ss *SandwichState) SetGuildChannel(ctx *StateCtx, guildIDPtr *discord.Snowflake, channel *discord.Channel) {
	var guildID discord.Snowflake

	if guildIDPtr != nil {
		guildID = *guildIDPtr
	} else {
		guildID = discord.Snowflake(0)
	}

	ss.guildChannelsMu.RLock()
	guildChannels, ok := ss.GuildChannels[guildID]
	ss.guildChannelsMu.RUnlock()

	if !ok {
		guildChannels = &sandwich_structs.StateGuildChannels{
			ChannelsMu: sync.RWMutex{},
			Channels:   make(map[discord.Snowflake]*sandwich_structs.StateChannel),
		}

		ss.guildChannelsMu.Lock()
		ss.GuildChannels[guildID] = guildChannels
		ss.guildChannelsMu.Unlock()
	}

	guildChannels.ChannelsMu.Lock()
	defer guildChannels.ChannelsMu.Unlock()

	guildChannels.Channels[channel.ID] = ss.ChannelToState(channel)

	for _, recipient := range channel.Recipients {
		recipient := recipient
		ss.SetUser(ctx, recipient)
	}
}

// RemoveGuildChannel removes a channel from the cache.
func (ss *SandwichState) RemoveGuildChannel(guildIDPtr *discord.Snowflake, channelID discord.Snowflake) {
	var guildID discord.Snowflake

	if guildIDPtr != nil {
		guildID = *guildIDPtr
	} else {
		guildID = discord.Snowflake(0)
	}

	ss.guildChannelsMu.RLock()
	guildChannels, ok := ss.GuildChannels[guildID]
	ss.guildChannelsMu.RUnlock()

	if !ok {
		return
	}

	guildChannels.ChannelsMu.Lock()
	delete(guildChannels.Channels, channelID)
	guildChannels.ChannelsMu.Unlock()
}

// GetAllGuildChannels returns all guildChannels of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildChannels(guildID discord.Snowflake) (guildChannelsList []*discord.Channel, ok bool) {
	ss.guildChannelsMu.RLock()
	guildChannels, ok := ss.GuildChannels[guildID]
	ss.guildChannelsMu.RUnlock()

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
	ss.guildChannelsMu.Lock()
	delete(ss.GuildChannels, guildID)
	ss.guildChannelsMu.Unlock()
}

// GetDMChannel returns the DM channel of a user.
func (ss *SandwichState) GetDMChannel(userID discord.Snowflake) (channel *discord.Channel, ok bool) {
	ss.dmChannelsMu.RLock()
	dmChannel, ok := ss.dmChannels[userID]
	ss.dmChannelsMu.RUnlock()

	if !ok || int64(dmChannel.ExpiresAt) < time.Now().Unix() {
		ok = false

		return
	}

	channel = dmChannel.Channel
	dmChannel.ExpiresAt = discord.Int64(time.Now().Add(memberDMExpiration).Unix())

	ss.dmChannelsMu.Lock()
	ss.dmChannels[userID] = dmChannel
	ss.dmChannelsMu.Unlock()

	return
}

// AddDMChannel adds a DM channel to a user.
func (ss *SandwichState) AddDMChannel(userID discord.Snowflake, channel *discord.Channel) {
	dmChannel := &sandwich_structs.StateDMChannel{
		Channel:   channel,
		ExpiresAt: discord.Int64(time.Now().Add(memberDMExpiration).Unix()),
	}

	ss.dmChannelsMu.Lock()
	ss.dmChannels[userID] = dmChannel
	ss.dmChannelsMu.Unlock()
}

// RemoveDMChannel removes a DM channel from a user.
func (ss *SandwichState) RemoveDMChannel(userID discord.Snowflake) {
	ss.dmChannelsMu.Lock()
	delete(ss.dmChannels, userID)
	ss.dmChannelsMu.Unlock()
}

// GetUserMutualGuilds returns a list of snowflakes of mutual guilds a member is seen on.
func (ss *SandwichState) GetUserMutualGuilds(userID discord.Snowflake) (guildIDs []discord.Snowflake, ok bool) {
	ss.mutualsMu.RLock()
	mutualGuilds, ok := ss.Mutuals[userID]
	ss.mutualsMu.RUnlock()

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
func (ss *SandwichState) AddUserMutualGuild(ctx *StateCtx, userID discord.Snowflake, guildID discord.Snowflake) {
	if !ctx.StoreMutuals {
		return
	}

	ss.mutualsMu.RLock()
	mutualGuilds, ok := ss.Mutuals[userID]
	ss.mutualsMu.RUnlock()

	if !ok {
		mutualGuilds = &sandwich_structs.StateMutualGuilds{
			GuildsMu: sync.RWMutex{},
			Guilds:   make(map[discord.Snowflake]bool),
		}

		ss.mutualsMu.Lock()
		ss.Mutuals[userID] = mutualGuilds
		ss.mutualsMu.Unlock()
	}

	mutualGuilds.GuildsMu.Lock()
	mutualGuilds.Guilds[guildID] = true
	mutualGuilds.GuildsMu.Unlock()
}

// RemoveUserMutualGuild removes a mutual guild from a user.
func (ss *SandwichState) RemoveUserMutualGuild(userID discord.Snowflake, guildID discord.Snowflake) {
	ss.mutualsMu.RLock()
	mutualGuilds, ok := ss.Mutuals[userID]
	ss.mutualsMu.RUnlock()

	if !ok {
		return
	}

	mutualGuilds.GuildsMu.Lock()
	delete(mutualGuilds.Guilds, guildID)
	mutualGuilds.GuildsMu.Unlock()
}

//
// VoiceState Operations
//

func (ss *SandwichState) VoiceStateFromState(guildID discord.Snowflake, userID discord.Snowflake, voiceStateState *sandwich_structs.StateVoiceState) (voiceState *discord.VoiceState) {
	guildMember, _ := ss.GetGuildMember(guildID, userID)

	return &discord.VoiceState{
		GuildID:   &guildID,
		ChannelID: voiceStateState.ChannelID,
		UserID:    userID,
		Member:    guildMember,
		SessionID: voiceStateState.SessionID,
		Deaf:      voiceStateState.Deaf,
		Mute:      voiceStateState.Mute,
		SelfDeaf:  voiceStateState.SelfDeaf,
		SelfMute:  voiceStateState.SelfMute,
		SelfVideo: voiceStateState.SelfVideo,
		Suppress:  voiceStateState.Suppress,
	}
}

func (ss *SandwichState) VoiceStateToState(voiceState *discord.VoiceState) (voiceStateState *sandwich_structs.StateVoiceState) {
	return &sandwich_structs.StateVoiceState{
		ChannelID: voiceState.ChannelID,
		SessionID: voiceState.SessionID,
		Deaf:      voiceState.Deaf,
		Mute:      voiceState.Mute,
		SelfDeaf:  voiceState.SelfDeaf,
		SelfMute:  voiceState.SelfMute,
		SelfVideo: voiceState.SelfVideo,
		Suppress:  voiceState.Suppress,
	}
}

func (ss *SandwichState) GetVoiceState(guildID discord.Snowflake, userID discord.Snowflake) (voiceState *discord.VoiceState, ok bool) {
	ss.guildVoiceStatesMu.RLock()
	guildVoiceStates, ok := ss.GuildVoiceStates[guildID]
	ss.guildVoiceStatesMu.RUnlock()

	if !ok {
		return
	}

	guildVoiceStates.VoiceStatesMu.RLock()
	stateVoiceState, ok := guildVoiceStates.VoiceStates[userID]
	guildVoiceStates.VoiceStatesMu.RUnlock()

	if !ok {
		return
	}

	voiceState = ss.VoiceStateFromState(guildID, userID, stateVoiceState)

	return
}

func (ss *SandwichState) UpdateVoiceState(ctx *StateCtx, voiceState discord.VoiceState) {
	if voiceState.GuildID == nil {
		return
	}

	ss.guildVoiceStatesMu.RLock()
	guildVoiceStates, ok := ss.GuildVoiceStates[*voiceState.GuildID]
	ss.guildVoiceStatesMu.RUnlock()

	if !ok {
		guildVoiceStates = &sandwich_structs.StateGuildVoiceStates{
			VoiceStatesMu: sync.RWMutex{},
			VoiceStates:   make(map[discord.Snowflake]*sandwich_structs.StateVoiceState),
		}

		ss.guildVoiceStatesMu.Lock()
		ss.GuildVoiceStates[*voiceState.GuildID] = guildVoiceStates
		ss.guildVoiceStatesMu.Unlock()
	}

	beforeVoiceState, _ := ctx.Sandwich.State.GetVoiceState(*voiceState.GuildID, voiceState.UserID)

	guildVoiceStates.VoiceStatesMu.Lock()
	if voiceState.ChannelID == 0 {
		// Remove from voice states if leaving voice channel.
		delete(guildVoiceStates.VoiceStates, voiceState.UserID)
	} else {
		guildVoiceStates.VoiceStates[voiceState.UserID] = ss.VoiceStateToState(&voiceState)
	}
	guildVoiceStates.VoiceStatesMu.Unlock()

	if voiceState.Member != nil {
		ss.SetGuildMember(ctx, *voiceState.GuildID, voiceState.Member)
	}

	// Update channel counts

	if beforeVoiceState == nil || beforeVoiceState.ChannelID != voiceState.ChannelID {
		if beforeVoiceState != nil {
			voiceChannel, ok := ctx.Sandwich.State.GetGuildChannel(beforeVoiceState.GuildID, beforeVoiceState.ChannelID)
			if ok {
				voiceChannel.MemberCount = ss.CountMembersForVoiceChannel(*beforeVoiceState.GuildID, voiceChannel.ID)

				ctx.Sandwich.State.SetGuildChannel(ctx, beforeVoiceState.GuildID, voiceChannel)
			}
		}

		voiceChannel, ok := ctx.Sandwich.State.GetGuildChannel(voiceState.GuildID, voiceState.ChannelID)
		if ok {
			voiceChannel.MemberCount = ss.CountMembersForVoiceChannel(*voiceState.GuildID, voiceChannel.ID)

			ctx.Sandwich.State.SetGuildChannel(ctx, voiceState.GuildID, voiceChannel)
		}
	}
}

func (ss *SandwichState) CountMembersForVoiceChannel(guildID discord.Snowflake, channelID discord.Snowflake) int32 {
	ss.guildVoiceStatesMu.RLock()
	guildVoiceStates, ok := ss.GuildVoiceStates[guildID]
	ss.guildVoiceStatesMu.RUnlock()

	if !ok {
		return 0
	}

	guildVoiceStates.VoiceStatesMu.RLock()
	defer guildVoiceStates.VoiceStatesMu.RUnlock()

	var count int32

	for _, voiceState := range guildVoiceStates.VoiceStates {
		if voiceState.ChannelID == channelID {
			count++
		}
	}

	return count
}
