package internal

import (
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

// GetGuild returns the guild with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuild(guildID discord.Snowflake) (guild *discord.Guild, ok bool) {
	guild, ok = ss.Guilds.Load(guildID)

	if !ok {
		return
	}

	return
}

// SetGuild creates or updates a guild entry in the cache.
func (ss *SandwichState) SetGuild(ctx *StateCtx, guild *discord.Guild) {
	ctx.ShardGroup.Guilds.Store(guild.ID, true)
	ss.Guilds.Store(guild.ID, guild)

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
	ss.Guilds.Delete(guildID)

	if !ctx.Stateless {
		ctx.ShardGroup.Guilds.Delete(guildID)
	}

	ss.RemoveAllGuildRoles(guildID)
	ss.RemoveAllGuildChannels(guildID)
	ss.RemoveAllGuildEmojis(guildID)
	ss.RemoveAllGuildMembers(guildID)
}

// GetGuildMember returns the guildMember with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildMember(guildID discord.Snowflake, guildMemberID discord.Snowflake) (guildMember *discord.GuildMember, ok bool) {
	guildMembers, ok := ss.GuildMembers.Load(guildID)

	if !ok {
		return
	}

	guildMember, ok = guildMembers.Members.Load(guildMemberID)

	if !ok {
		return
	}

	// FIX: Ensure that joined_at is set correctly, it tends to get corrupted for some reason
	//
	// This is common enough to not warrning a log message for it.
	if guildMember.JoinedAt != "" {
		if _, err := time.Parse(time.RFC3339, string(guildMember.JoinedAt)); err != nil {
			guildMember.JoinedAt = ""
		}
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

	guildMembers, ok := ss.GuildMembers.Load(guildID)

	if !ok {
		// Only set if its not already set.
		ss.GuildMembers.SetIfAbsent(guildID, sandwich_structs.StateGuildMembers{
			Members: csmap.Create(
				csmap.WithSize[discord.Snowflake, *discord.GuildMember](100),
			),
		})

		guildMembers, _ = ss.GuildMembers.Load(guildID)
	}

	guildMembers.Members.Store(guildMember.User.ID, guildMember)

	ss.SetUser(ctx, guildMember.User)
}

// RemoveGuildMember removes a guildMember from the cache.
func (ss *SandwichState) RemoveGuildMember(guildID discord.Snowflake, guildMemberID discord.Snowflake) {
	guildMembers, ok := ss.GuildMembers.Load(guildID)

	if !ok {
		return
	}

	guildMembers.Members.Delete(guildMemberID)
}

// GetAllGuildMembers returns all guildMembers of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildMembers(guildID discord.Snowflake) (guildMembersList []*discord.GuildMember, ok bool) {
	guildMembers, ok := ss.GuildMembers.Load(guildID)

	if !ok {
		return
	}

	guildMembers.Members.Range(func(_ discord.Snowflake, guildMember *discord.GuildMember) bool {
		guildMembersList = append(guildMembersList, guildMember)
		return false
	})

	return
}

// RemoveAllGuildMembers removes all guildMembers of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildMembers(guildID discord.Snowflake) {
	ss.GuildMembers.Delete(guildID)
}

// GetGuildRole returns the role with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildRole(guildID discord.Snowflake, roleID discord.Snowflake) (role *discord.Role, ok bool) {
	stateGuildRoles, ok := ss.GuildRoles.Load(roleID)

	if !ok {
		return
	}

	role, ok = stateGuildRoles.Roles.Load(roleID)

	if !ok {
		return
	}

	return
}

// SetGuildRole creates or updates a role entry in the cache.
func (ss *SandwichState) SetGuildRole(ctx *StateCtx, guildID discord.Snowflake, role *discord.Role) {
	guildRoles, ok := ss.GuildRoles.Load(guildID)

	if !ok {
		ss.GuildRoles.SetIfAbsent(guildID, sandwich_structs.StateGuildRoles{
			Roles: csmap.Create(
				csmap.WithSize[discord.Snowflake, *discord.Role](100),
			),
		})

		guildRoles, _ = ss.GuildRoles.Load(guildID)
	}

	guildRoles.Roles.Store(role.ID, role)
}

// RemoveGuildRole removes a role from the cache.
func (ss *SandwichState) RemoveGuildRole(guildID discord.Snowflake, roleID discord.Snowflake) {
	guildRoles, ok := ss.GuildRoles.Load(guildID)

	if !ok {
		return
	}

	guildRoles.Roles.Delete(roleID)
}

// GetAllGuildRoles returns all guildRoles of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildRoles(guildID discord.Snowflake) (guildRolesList []*discord.Role, ok bool) {
	guildRoles, ok := ss.GuildRoles.Load(guildID)

	if !ok {
		return
	}

	guildRoles.Roles.Range(func(_ discord.Snowflake, role *discord.Role) bool {
		guildRolesList = append(guildRolesList, role)
		return false
	})

	return
}

// RemoveGuildRoles removes all guild roles of a specifi guild from the cache.
func (ss *SandwichState) RemoveAllGuildRoles(guildID discord.Snowflake) {
	ss.GuildRoles.Delete(guildID)
}

//
// Emoji Operations
//

// GetGuildEmoji returns the emoji with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildEmoji(guildID discord.Snowflake, emojiID discord.Snowflake) (guildEmoji *discord.Emoji, ok bool) {
	guildEmojis, ok := ss.GuildEmojis.Load(guildID)

	if !ok {
		return
	}

	guildEmoji, ok = guildEmojis.Emojis.Load(emojiID)

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
	guildEmojis, ok := ss.GuildEmojis.Load(guildID)

	if !ok {
		ss.GuildEmojis.SetIfAbsent(guildID, sandwich_structs.StateGuildEmojis{
			Emojis: csmap.Create(
				csmap.WithSize[discord.Snowflake, *discord.Emoji](50),
			),
		})

		guildEmojis, _ = ss.GuildEmojis.Load(guildID)
	}

	guildEmojis.Emojis.Store(emoji.ID, emoji)

	if emoji.User != nil {
		ss.SetUser(ctx, emoji.User)
	}
}

// RemoveGuildEmoji removes a emoji from the cache.
func (ss *SandwichState) RemoveGuildEmoji(guildID discord.Snowflake, emojiID discord.Snowflake) {
	guildEmojis, ok := ss.GuildEmojis.Load(guildID)

	if !ok {
		return
	}

	guildEmojis.Emojis.Delete(emojiID)
}

// GetAllGuildEmojis returns all guildEmojis on a specific guild from the cache.
func (ss *SandwichState) GetAllGuildEmojis(guildID discord.Snowflake) (guildEmojisList []*discord.Emoji, ok bool) {
	guildEmojis, ok := ss.GuildEmojis.Load(guildID)

	if !ok {
		return
	}

	guildEmojis.Emojis.Range(func(_ discord.Snowflake, guildEmoji *discord.Emoji) bool {
		guildEmojisList = append(guildEmojisList, guildEmoji)
		return false
	})

	return
}

// RemoveGuildEmojis removes all guildEmojis of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildEmojis(guildID discord.Snowflake) {
	ss.GuildEmojis.Delete(guildID)
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
	stateUser, ok := ss.Users.Load(userID)

	if !ok {
		return
	}

	user = ss.UserFromState(&stateUser)

	return
}

// SetUser creates or updates a user entry in the cache.
func (ss *SandwichState) SetUser(ctx *StateCtx, user *discord.User) {
	// We will always cache the user of the bot that receives this event.
	if !ctx.CacheUsers && user.ID != ctx.Manager.User.ID {
		return
	}

	ss.Users.Store(user.ID, *ss.UserToState(user))
}

// RemoveUser removes a user from the cache.
func (ss *SandwichState) RemoveUser(userID discord.Snowflake) {
	ss.Users.Delete(userID)
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

	stateChannels, ok := ss.GuildChannels.Load(guildID)

	if !ok {
		return guildChannel, false
	}

	guildChannel, ok = stateChannels.Channels.Load(channelID)
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

	guildChannels, ok := ss.GuildChannels.Load(guildID)

	if !ok {
		ss.GuildChannels.SetIfAbsent(guildID, sandwich_structs.StateGuildChannels{
			Channels: csmap.Create(
				csmap.WithSize[discord.Snowflake, *discord.Channel](50),
			),
		})

		guildChannels, _ = ss.GuildChannels.Load(guildID)
	}

	guildChannels.Channels.Store(channel.ID, channel)

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

	guildChannels, ok := ss.GuildChannels.Load(guildID)

	if !ok {
		return
	}

	guildChannels.Channels.Delete(channelID)
}

// GetAllGuildChannels returns all guildChannels of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildChannels(guildID discord.Snowflake) (guildChannelsList []*discord.Channel, ok bool) {
	guildChannels, ok := ss.GuildChannels.Load(guildID)

	if !ok {
		return
	}

	guildChannels.Channels.Range(func(_ discord.Snowflake, guildChannel *discord.Channel) bool {
		guildChannelsList = append(guildChannelsList, guildChannel)
		return false
	})

	return
}

// RemoveAllGuildChannels removes all guildChannels of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildChannels(guildID discord.Snowflake) {
	ss.GuildChannels.Delete(guildID)
}

// GetDMChannel returns the DM channel of a user.
func (ss *SandwichState) GetDMChannel(userID discord.Snowflake) (channel *discord.Channel, ok bool) {
	dmChannel, ok := ss.DmChannels.Load(userID)

	if !ok || int64(dmChannel.ExpiresAt) < time.Now().Unix() {
		ok = false

		return
	}

	channel = dmChannel.Channel
	dmChannel.ExpiresAt = discord.Int64(time.Now().Add(memberDMExpiration).Unix())

	ss.DmChannels.Store(userID, dmChannel)

	return
}

// AddDMChannel adds a DM channel to a user.
func (ss *SandwichState) AddDMChannel(userID discord.Snowflake, channel *discord.Channel) {
	ss.DmChannels.Store(userID, sandwich_structs.StateDMChannel{
		Channel:   channel,
		ExpiresAt: discord.Int64(time.Now().Add(memberDMExpiration).Unix()),
	})
}

// RemoveDMChannel removes a DM channel from a user.
func (ss *SandwichState) RemoveDMChannel(userID discord.Snowflake) {
	ss.DmChannels.Delete(userID)
}

// GetUserMutualGuilds returns a list of snowflakes of mutual guilds a member is seen on.
func (ss *SandwichState) GetUserMutualGuilds(userID discord.Snowflake) (guildIDs []discord.Snowflake, ok bool) {
	mutualGuilds, ok := ss.Mutuals.Load(userID)

	if !ok {
		return
	}

	mutualGuilds.Guilds.Range(func(guildID discord.Snowflake, _ bool) bool {
		guildIDs = append(guildIDs, guildID)
		return false
	})

	return
}

// AddUserMutualGuild adds a mutual guild to a user.
func (ss *SandwichState) AddUserMutualGuild(ctx *StateCtx, userID discord.Snowflake, guildID discord.Snowflake) {
	if !ctx.StoreMutuals {
		return
	}

	mutualGuilds, ok := ss.Mutuals.Load(userID)

	if !ok {
		ss.Mutuals.SetIfAbsent(userID, sandwich_structs.StateMutualGuilds{
			Guilds: csmap.Create(
				csmap.WithSize[discord.Snowflake, bool](20),
			),
		})

		mutualGuilds, _ = ss.Mutuals.Load(userID)
	}

	mutualGuilds.Guilds.Store(guildID, true)
}

// RemoveUserMutualGuild removes a mutual guild from a user.
func (ss *SandwichState) RemoveUserMutualGuild(userID discord.Snowflake, guildID discord.Snowflake) {
	mutualGuilds, ok := ss.Mutuals.Load(userID)

	if !ok {
		return
	}

	mutualGuilds.Guilds.Delete(guildID)
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
	guildVoiceStates, ok := ss.GuildVoiceStates.Load(guildID)

	if !ok {
		return
	}

	stateVoiceState, ok := guildVoiceStates.VoiceStates.Load(userID)

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

	guildVoiceStates, ok := ss.GuildVoiceStates.Load(*voiceState.GuildID)

	if !ok {
		ss.GuildVoiceStates.SetIfAbsent(*voiceState.GuildID, sandwich_structs.StateGuildVoiceStates{
			VoiceStates: csmap.Create(
				csmap.WithSize[discord.Snowflake, *discord.VoiceState](50),
			),
		})

		guildVoiceStates, _ = ss.GuildVoiceStates.Load(*voiceState.GuildID)
	}

	beforeVoiceState, _ := ctx.Sandwich.State.GetVoiceState(*voiceState.GuildID, voiceState.UserID)

	if voiceState.ChannelID == 0 {
		// Remove from voice states if leaving voice channel.
		guildVoiceStates.VoiceStates.Delete(voiceState.UserID)
	} else {
		guildVoiceStates.VoiceStates.Store(voiceState.UserID, ss.ParseVoiceState(*voiceState.GuildID, voiceState.UserID, &voiceState))
	}

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
	guildVoiceStates, ok := ss.GuildVoiceStates.Load(guildID)

	if !ok {
		return 0
	}

	var count int32

	guildVoiceStates.VoiceStates.Range(func(_ discord.Snowflake, voiceState *discord.VoiceState) bool {
		if voiceState.ChannelID == channelID {
			count++
		}
		return false
	})

	return count
}
