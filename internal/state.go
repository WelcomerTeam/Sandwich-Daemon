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
// TODO: Implement.
func (ss *SandwichState) GuildFromState(guildState *structs.StateGuild) (guild *discord.Guild) {
	return
}

// GuildFromState converts from discord.Guild to structs.StateGuild, for storing in cache.
// TODO: Implement.
func (ss *SandwichState) GuildToState(guild *discord.Guild) (guildState *structs.StateGuild) {
	return
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
// TODO: Implement.
func (ss *SandwichState) SetGuild(guild *discord.Guild) {
	ss.guildsMu.Lock()
	defer ss.guildsMu.Unlock()

	// Create roles, channels, members, emojis.

	ss.Guilds[guild.ID] = ss.GuildToState(guild)

	return
}

// RemoveGuild removes a guild from the cache.
// TODO: Implement.
func (ss *SandwichState) RemoveGuild(guildID discord.Snowflake) {
	ss.guildsMu.Lock()

	// Cleanup roles, channels, members, emojis.

	delete(ss.Guilds, guildID)
	ss.guildsMu.RUnlock()

	return
}

//
// GuildMember Operations
//

// GuildMemberFromState converts the structs.StateGuildMembers into a discord.GuildMember, for use within the application.
// TODO: Implement.
func (ss *SandwichState) GuildMemberFromState(guildState *structs.StateGuildMember) (guild *discord.GuildMember) {
	return
}

// GuildMemberFromState converts from discord.GuildMember to structs.StateGuildMembers, for storing in cache.
// TODO: Implement.
func (ss *SandwichState) GuildMemberToState(guild *discord.GuildMember) (guildState *structs.StateGuildMember) {
	return
}

// GetGuildMember returns the guildMember with the same ID from the cache.
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

	return
}

// SetGuildMember creates or updates a guildMember entry in the cache.
func (ss *SandwichState) SetGuildMember(guildID discord.Snowflake, guildMember *discord.GuildMember) {
	ss.guildMembersMu.Lock()
	defer ss.guildMembersMu.Lock()

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

//
// Role Operations
//

// RoleFromState converts the structs.StateRole into a discord.Role, for use within the application.
// TODO: Implement.
func (ss *SandwichState) RoleFromState(guildState *structs.StateRole) (guild *discord.Role) {
	return
}

// RoleFromState converts from discord.Role to structs.StateRole, for storing in cache.
// TODO: Implement.
func (ss *SandwichState) RoleToState(guild *discord.Role) (guildState *structs.StateRole) {
	return
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
// TODO: Implement.
func (ss *SandwichState) EmojiFromState(guildState *structs.StateEmoji) (guild *discord.Emoji) {
	return
}

// EmojiFromState converts from discord.Emoji to structs.StateEmoji, for storing in cache.
// TODO: Implement.
func (ss *SandwichState) EmojiToState(guild *discord.Emoji) (guildState *structs.StateEmoji) {
	return
}

// GetEmoji returns the emoji with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetEmoji(emojiID discord.Snowflake) (emoji *discord.Emoji, ok bool) {
	ss.emojisMu.RLock()
	defer ss.emojisMu.RUnlock()

	stateEmoji, ok := ss.Emojis[emojiID]
	if !ok {
		return
	}

	emoji = ss.EmojiFromState(stateEmoji)

	return
}

// SetEmoji creates or updates a emoji entry in the cache.
func (ss *SandwichState) SetEmoji(emoji *discord.Emoji) {
	ss.emojisMu.Lock()
	defer ss.emojisMu.Unlock()

	ss.Emojis[emoji.ID] = ss.EmojiToState(emoji)

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
// TODO: Implement.
func (ss *SandwichState) UserFromState(guildState *structs.StateUser) (guild *discord.User) {
	return
}

// UserFromState converts from discord.User to structs.StateUser, for storing in cache.
// TODO: Implement.
func (ss *SandwichState) UserToState(guild *discord.User) (guildState *structs.StateUser) {
	return
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
// TODO: Implement.
func (ss *SandwichState) ChannelFromState(guildState *structs.StateChannel) (guild *discord.Channel) {
	return
}

// ChannelFromState converts from discord.Channel to structs.StateChannel, for storing in cache.
// TODO: Implement.
func (ss *SandwichState) ChannelToState(guild *discord.Channel) (guildState *structs.StateChannel) {
	return
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

	return
}

// SetChannel creates or updates a channel entry in the cache.
func (ss *SandwichState) SetChannel(channel *discord.Channel) {
	ss.channelsMu.Lock()
	defer ss.channelsMu.Unlock()

	ss.Channels[channel.ID] = ss.ChannelToState(channel)

	return
}

// RemoveChannel removes a channel from the cache.
func (ss *SandwichState) RemoveChannel(channelID discord.Snowflake) {
	ss.channelsMu.Lock()
	defer ss.channelsMu.RUnlock()

	delete(ss.Channels, channelID)

	return
}
