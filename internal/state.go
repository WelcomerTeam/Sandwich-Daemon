package internal

import (
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
)

// GetGuild returns the guild with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuild(guildID discord.Snowflake) (guild *discord.Guild, ok bool) {
	ss.guildsMu.RLock()
	guild, ok = ss.Guilds[guildID]
	ss.guildsMu.RUnlock()

	if !ok {
		return
	}

	return
}

// SetGuild creates or updates a guild entry in the cache.
func (ss *SandwichState) SetGuild(ctx *StateCtx, guild *discord.Guild) {
	ctx.ShardGroup.guildsMu.Lock()
	ctx.ShardGroup.Guilds[guild.ID] = true
	ctx.ShardGroup.guildsMu.Unlock()

	ss.guildsMu.Lock()
	ss.Guilds[guild.ID] = guild
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
	guildMember, ok = guildMembers.Members[guildMemberID]
	guildMembers.MembersMu.RUnlock()

	if !ok {
		return
	}

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
			Members:   make(map[discord.Snowflake]*discord.GuildMember),
		}

		ss.guildMembersMu.Lock()
		ss.GuildMembers[guildID] = guildMembers
		ss.guildMembersMu.Unlock()
	}

	guildMembers.MembersMu.Lock()
	guildMembers.Members[guildMember.User.ID] = guildMember
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
		guildMembersList = append(guildMembersList, guildMember)
	}

	return
}

// RemoveAllGuildMembers removes all guildMembers of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildMembers(guildID discord.Snowflake) {
	ss.guildMembersMu.Lock()
	delete(ss.GuildMembers, guildID)
	ss.guildMembersMu.Unlock()
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
	role, ok = stateGuildRoles.Roles[roleID]
	stateGuildRoles.RolesMu.RUnlock()

	if !ok {
		return
	}

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
			Roles:   make(map[discord.Snowflake]*discord.Role),
		}

		ss.guildRolesMu.Lock()
		ss.GuildRoles[guildID] = guildRoles
		ss.guildRolesMu.Unlock()
	}

	guildRoles.RolesMu.Lock()
	guildRoles.Roles[role.ID] = role
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
		guildRolesList = append(guildRolesList, guildRole)
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
	guildEmoji, ok = guildEmojis.Emojis[emojiID]
	guildEmojis.EmojisMu.RUnlock()

	if !ok {
		return
	}

	if guildEmoji.User != nil {
		user, ok := ss.GetUser(guildEmoji.User.ID)
		if ok {
			guildEmoji.User = user
		}
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
			Emojis:   make(map[discord.Snowflake]*discord.Emoji),
		}

		ss.guildEmojisMu.Lock()
		ss.GuildEmojis[guildID] = guildEmojis
		ss.guildEmojisMu.Unlock()
	}

	guildEmojis.EmojisMu.Lock()
	guildEmojis.Emojis[emoji.ID] = emoji
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
		guildEmojisList = append(guildEmojisList, guildEmoji)
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
	return userState.User
}

// UserFromState converts from discord.User to structs.StateUser, for storing in cache.
func (ss *SandwichState) UserToState(user *discord.User) (userState *sandwich_structs.StateUser) {
	return &sandwich_structs.StateUser{
		User:        user,
		LastUpdated: time.Now(),
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

	guildChannel, ok = stateChannels.Channels[channelID]
	if !ok {
		return guildChannel, false
	}

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

	// Ensure channel has guild id set
	channel.GuildID = &guildID

	ss.guildChannelsMu.RLock()
	guildChannels, ok := ss.GuildChannels[guildID]
	ss.guildChannelsMu.RUnlock()

	if !ok {
		guildChannels = &sandwich_structs.StateGuildChannels{
			ChannelsMu: sync.RWMutex{},
			Channels:   make(map[discord.Snowflake]*discord.Channel),
		}

		ss.guildChannelsMu.Lock()
		ss.GuildChannels[guildID] = guildChannels
		ss.guildChannelsMu.Unlock()
	}

	guildChannels.ChannelsMu.Lock()
	defer guildChannels.ChannelsMu.Unlock()

	guildChannels.Channels[channel.ID] = channel

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
		guildChannelsList = append(guildChannelsList, guildRole)
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

// ParseVoiceState parses a voice state info populating it from cache
func (ss *SandwichState) ParseVoiceState(guildID discord.Snowflake, userID discord.Snowflake, voiceStateState *discord.VoiceState) (voiceState *discord.VoiceState) {
	if voiceStateState.Member == nil {
		gm, _ := ss.GetGuildMember(guildID, userID)

		voiceStateState.Member = gm
	}

	voiceStateState.UserID = userID

	return voiceStateState
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

	voiceState = ss.ParseVoiceState(guildID, userID, stateVoiceState)

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
			VoiceStates:   make(map[discord.Snowflake]*discord.VoiceState),
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
		guildVoiceStates.VoiceStates[voiceState.UserID] = ss.ParseVoiceState(*voiceState.GuildID, voiceState.UserID, &voiceState)
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
