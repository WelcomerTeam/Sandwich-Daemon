package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
)

type StateCtx struct {
	context context.Context
	*Shard
	CacheUsers   bool
	CacheMembers bool
	Stateless    bool
	StoreMutuals bool
}

func NewFakeCtx(mg *Manager) StateCtx {
	return StateCtx{
		CacheUsers:   true,
		CacheMembers: true,
		StoreMutuals: true,
		Shard: &Shard{
			ctx:     mg.ctx,
			Manager: mg,
			Logger:  mg.Logger,
		},
	}
}

// SandwichState stores the collective state of all ShardGroups
// across all Managers.
type SandwichState struct {
	Guilds Cache[discord.GuildID, discord.Guild]

	GuildMembers DoubleCache[discord.GuildID, discord.UserID, discord.GuildMember]

	GuildChannels DoubleCache[discord.GuildID, discord.ChannelID, discord.Channel]

	GuildRoles DoubleCache[discord.GuildID, discord.RoleID, discord.Role]

	GuildEmojis Cache[discord.GuildID, []discord.Emoji]

	Users Cache[discord.UserID, StateUser]

	DmChannels Cache[discord.ChannelID, StateDMChannel]

	Mutuals DoubleCache[discord.UserID, discord.GuildID, struct{}]

	GuildVoiceStates DoubleCache[discord.GuildID, discord.UserID, discord.VoiceState]
}

func NewSandwichState() *SandwichState {
	state := &SandwichState{
		Guilds: NewCache[discord.GuildID, discord.Guild](100),

		GuildMembers: NewDoubleCache[discord.GuildID, discord.UserID, discord.GuildMember](0, 50),

		GuildChannels: NewDoubleCache[discord.GuildID, discord.ChannelID, discord.Channel](0, 50),

		GuildRoles: NewDoubleCache[discord.GuildID, discord.RoleID, discord.Role](0, 50),

		GuildEmojis: NewCache[discord.GuildID, []discord.Emoji](50),

		Users: NewCache[discord.UserID, StateUser](100),

		DmChannels: NewCache[discord.ChannelID, StateDMChannel](50),

		Mutuals: NewDoubleCache[discord.UserID, discord.GuildID, struct{}](0, 50),

		GuildVoiceStates: NewDoubleCache[discord.GuildID, discord.UserID, discord.VoiceState](0, 50),
	}

	return state
}

// GetGuild returns the guild with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuild(guildID discord.GuildID) (guild discord.Guild, ok bool) {
	guild, ok = ss.Guilds.Load(guildID)

	if !ok {
		return
	}

	// Get list of roles
	roles, ok := ss.GetAllGuildRoles(guildID)

	if !ok {
		fmt.Println("Failed to get roles")
		return
	}

	guild.Roles = roles

	// Get list of channels
	guildChannels, ok := ss.GetAllGuildChannels(guildID)

	if ok {
		guild.Channels = guildChannels
	} else {
		guild.Channels = make([]discord.Channel, 0)
	}

	// Get list of voice states, if any
	voiceStates, ok := ss.GuildVoiceStates.Inner(guildID)

	if ok {
		// Pre-allocate the list
		guild.VoiceStates = make([]discord.VoiceState, 0, voiceStates.Count())

		voiceStates.Range(func(_ discord.UserID, voiceState discord.VoiceState) bool {
			guild.VoiceStates = append(guild.VoiceStates, voiceState)
			return false
		})
	} else {
		guild.VoiceStates = make([]discord.VoiceState, 0)
	}

	// Get list of emojis
	emojis, ok := ss.GetAllGuildEmojis(guildID)

	if !ok {
		fmt.Println("Failed to get emojis")
		return
	}

	guild.Emojis = emojis

	// Get list of members
	members, ok := ss.GetAllGuildMembers(guildID)

	if !ok {
		fmt.Println("Failed to get members")
		return
	}

	// Fix AFK channel
	if guild.AFKChannelID == nil {
		cid := discord.ChannelID(guild.ID) // Channel ID of AFK channel id is the guild id
		guild.AFKChannelID = &cid
	}

	guild.Members = members
	guild.ApproximateMemberCount = guild.MemberCount
	ok = true

	return
}

// SetGuild creates or updates a guild entry in the cache.
//
// NOT fake-ctx-safe UNLESS
func (ss *SandwichState) SetGuild(ctx StateCtx, guild discord.Guild) {
	ctx.ShardGroup.Guilds.Store(guild.ID, struct{}{})
	ss.Guilds.Store(guild.ID, guild)
	ctx.Guilds.Store(guild.ID, struct{}{})

	// Safety: there is guaranteed to be at least one role
	for _, role := range guild.Roles {
		ss.SetGuildRole(guild.ID, role)
	}

	for _, channel := range guild.Channels {
		ss.SetGuildChannel(ctx, guild.ID, channel)
	}

	ss.SetGuildEmojis(ctx, guild.ID, guild.Emojis)

	for _, member := range guild.Members {
		ss.SetGuildMember(ctx, guild.ID, member)
	}

	for _, voiceState := range guild.VoiceStates {
		voiceState.GuildID = &guild.ID
		ss.UpdateVoiceState(ctx, voiceState)
	}

	// Clear out some data that we don't need to cache in guild
	guild.Roles = nil
	guild.Channels = nil
	guild.VoiceStates = nil
	guild.Members = nil // No need to duplicate this data.
	guild.Emojis = nil  // No need to duplicate this data.
}

// RemoveGuild removes a guild from the cache.
//
// NOT fake-ctx-safe
func (ss *SandwichState) RemoveGuild(ctx StateCtx, guildID discord.GuildID) {
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
func (ss *SandwichState) GetGuildMember(guildID discord.GuildID, guildMemberID discord.UserID) (guildMember discord.GuildMember, ok bool) {
	guildMembers, ok := ss.GuildMembers.Inner(guildID)

	if !ok {
		return
	}

	guildMember, ok = guildMembers.Load(guildMemberID)

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
		guildMember.User = &user
	}

	return
}

// SetGuildMember creates or updates a guildMember entry in the cache. Adds user in guildMember object to cache.
//
// fake-ctx-safe
func (ss *SandwichState) SetGuildMember(ctx StateCtx, guildID discord.GuildID, guildMember discord.GuildMember) {
	// We will always cache the guild member of the bot that receives this event.
	if !ctx.CacheMembers && guildMember.User.ID != ctx.Manager.User.ID {
		return
	}

	ss.GuildMembers.Store(guildID, guildMember.User.ID, guildMember)

	if guildMember.User != nil {
		ss.SetUser(ctx, *guildMember.User)
	}
}

// RemoveGuildMember removes a guildMember from the cache.
func (ss *SandwichState) RemoveGuildMember(guildID discord.GuildID, guildMemberID discord.UserID) {
	ss.GuildMembers.Delete(guildID, guildMemberID)
}

// GetAllGuildMembers returns all guildMembers of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildMembers(guildID discord.GuildID) (guildMembersList []discord.GuildMember, ok bool) {
	guildMembers, ok := ss.GuildMembers.Inner(guildID)

	if !ok {
		return
	}

	// Pre-allocate the list
	guildMembersList = make([]discord.GuildMember, 0, guildMembers.Count())

	guildMembers.Range(func(_ discord.UserID, guildMember discord.GuildMember) bool {
		guildMembersList = append(guildMembersList, guildMember)
		return false
	})

	return
}

// RemoveAllGuildMembers removes all guildMembers of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildMembers(guildID discord.GuildID) {
	ss.GuildMembers.ClearKey(guildID)
}

// GetGuildRole returns the role with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildRole(guildID discord.GuildID, roleID discord.RoleID) (role discord.Role, ok bool) {
	return ss.GuildRoles.Load(guildID, roleID)
}

// SetGuildRole creates or updates a role entry in the cache.
func (ss *SandwichState) SetGuildRole(guildID discord.GuildID, role discord.Role) {
	ss.GuildRoles.Store(guildID, role.ID, role)
}

// RemoveGuildRole removes a role from the cache.
func (ss *SandwichState) RemoveGuildRole(guildID discord.GuildID, roleID discord.RoleID) {
	ss.GuildRoles.Delete(guildID, roleID)
}

// GetAllGuildRoles returns all guildRoles of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildRoles(guildID discord.GuildID) (guildRolesList []discord.Role, ok bool) {
	guildRoles, ok := ss.GuildRoles.Inner(guildID)

	if !ok {
		return
	}

	// Pre-allocate the list
	guildRolesList = make([]discord.Role, 0, guildRoles.Count())

	guildRoles.Range(func(id discord.RoleID, role discord.Role) bool {
		if role.ID == 0 {
			role.ID = id
		}

		guildRolesList = append(guildRolesList, role)
		return false
	})

	return
}

// RemoveGuildRoles removes all guild roles of a specifi guild from the cache.
func (ss *SandwichState) RemoveAllGuildRoles(guildID discord.GuildID) {
	ss.GuildRoles.ClearKey(guildID)
}

//
// Emoji Operations
//

// GetGuildEmoji returns the emoji with the same ID from the cache. Populated user field from cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildEmoji(guildID discord.GuildID, emojiID discord.EmojiID) (guildEmoji discord.Emoji, ok bool) {
	guildEmojis, ok := ss.GuildEmojis.Load(guildID)

	if !ok {
		return
	}

	for _, emoji := range guildEmojis {
		if emoji.ID == emojiID {
			guildEmoji = emoji
			ok = true
			break
		}
	}

	if guildEmoji.User != nil {
		user, ok := ss.GetUser(guildEmoji.User.ID)
		if ok {
			guildEmoji.User = &user
		}
	}

	return
}

// SetGuildEmoji sets the list of emoji entries in the cache. Adds user in user object to cache.
//
// fake-ctx-safe
func (ss *SandwichState) SetGuildEmojis(ctx StateCtx, guildID discord.GuildID, emojis []discord.Emoji) {
	ss.GuildEmojis.Store(guildID, emojis)

	for _, emoji := range emojis {
		if emoji.User != nil {
			ss.SetUser(ctx, *emoji.User)
		}
	}
}

// GetAllGuildEmojis returns all guildEmojis on a specific guild from the cache.
func (ss *SandwichState) GetAllGuildEmojis(guildID discord.GuildID) (guildEmojisList []discord.Emoji, ok bool) {
	return ss.GuildEmojis.Load(guildID)
}

// RemoveGuildEmojis removes all guildEmojis of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildEmojis(guildID discord.GuildID) {
	ss.GuildEmojis.Delete(guildID)
}

//
// User Operations
//

// UserFromState converts the structs.StateUser into a discord.User, for use within the application.
func (ss *SandwichState) UserFromState(userState StateUser) discord.User {
	return userState.User
}

// UserFromState converts from discord.User to structs.StateUser, for storing in cache.
func (ss *SandwichState) UserToState(user discord.User) StateUser {
	return StateUser{
		User:        user,
		LastUpdated: time.Now(),
	}
}

// GetUser returns the user with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetUser(userID discord.UserID) (user discord.User, ok bool) {
	stateUser, ok := ss.Users.Load(userID)

	if !ok {
		return
	}

	user = ss.UserFromState(stateUser)

	return
}

// SetUser creates or updates a user entry in the cache.
//
// fake-ctx-safe
func (ss *SandwichState) SetUser(ctx StateCtx, user discord.User) {
	// We will always cache the user of the bot that receives this event.
	if !ctx.CacheUsers && user.ID != ctx.Manager.User.ID {
		return
	}

	ss.Users.Store(user.ID, ss.UserToState(user))
}

// RemoveUser removes a user from the cache.
func (ss *SandwichState) RemoveUser(userID discord.UserID) {
	ss.Users.Delete(userID)
}

//
// Channel Operations
//

// GetGuildChannel returns the channel with the same ID from the cache.
// Returns a boolean to signify a match or not.
func (ss *SandwichState) GetGuildChannel(guildID discord.GuildID, channelID discord.ChannelID) (guildChannel discord.Channel, ok bool) {
	guildChannel, ok = ss.GuildChannels.Load(guildID, channelID)

	if !ok {
		return guildChannel, false
	}

	newRecipients := make([]discord.User, 0, len(guildChannel.Recipients))

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
//
// fake-ctx-safe
func (ss *SandwichState) SetGuildChannel(ctx StateCtx, guildID discord.GuildID, channel discord.Channel) {
	// Ensure channel has guild id set
	channel.GuildID = &guildID

	ss.GuildChannels.Store(guildID, channel.ID, channel)

	for _, recipient := range channel.Recipients {
		recipient := recipient
		ss.SetUser(ctx, recipient)
	}
}

// RemoveGuildChannel removes a channel from the cache.
func (ss *SandwichState) RemoveGuildChannel(guildID discord.GuildID, channelID discord.ChannelID) {
	ss.GuildChannels.Delete(guildID, channelID)
}

// Runs a function on a guild channel in the cache, updating the value in cache based on returned value.
func (ss *SandwichState) UpdateGuildChannel(guildID discord.GuildID, channelID discord.ChannelID, fn func(channel discord.Channel) discord.Channel) (channel discord.Channel, ok bool) {
	return ss.GuildChannels.Update(guildID, channelID, fn)
}

// GetChannel returns a channel from its ID searching both DMs and guild channels.
//
// Note that guildIdHint must be provided if the channel is not a DM channel otherwise no result will be returned.
func (ss *SandwichState) GetChannel(guildIdHint *discord.GuildID, channelID discord.ChannelID) (channel *discord.Channel, ok bool) {
	dmChannel, ok := ss.GetDMChannel(channelID)

	if ok {
		return &dmChannel, true
	}

	if guildIdHint != nil {
		_channel, ok := ss.GetGuildChannel(*guildIdHint, channelID)
		return &_channel, ok
	} else {
		return nil, false
	}
}

// GetAllGuildChannels returns all guildChannels of a specific guild from the cache.
func (ss *SandwichState) GetAllGuildChannels(guildID discord.GuildID) (guildChannelsList []discord.Channel, ok bool) {
	guildChannels, ok := ss.GuildChannels.Inner(guildID)

	if !ok {
		return
	}

	// Pre-allocate the list
	guildChannelsList = make([]discord.Channel, 0, guildChannels.Count())

	guildChannels.Range(func(_ discord.ChannelID, guildChannel discord.Channel) bool {
		guildChannelsList = append(guildChannelsList, guildChannel)
		return false
	})

	return
}

// RemoveAllGuildChannels removes all guildChannels of a specific guild from the cache.
func (ss *SandwichState) RemoveAllGuildChannels(guildID discord.GuildID) {
	ss.GuildChannels.ClearKey(guildID)
}

// GetDMChannel returns the DM channel of a user.
func (ss *SandwichState) GetDMChannel(channelID discord.ChannelID) (channel discord.Channel, ok bool) {
	dmChannel, ok := ss.DmChannels.Load(channelID)

	if !ok || int64(dmChannel.ExpiresAt) < time.Now().Unix() {
		ok = false

		return
	}

	channel = dmChannel.Channel
	dmChannel.ExpiresAt = discord.Int64(time.Now().Add(memberDMExpiration).Unix())

	ss.DmChannels.Store(channelID, dmChannel)

	return
}

// AddDMChannel adds a DM channel to a user.
func (ss *SandwichState) AddDMChannel(userID discord.UserID, channel discord.Channel) {
	ss.DmChannels.Store(channel.ID, StateDMChannel{
		Channel:   channel,
		UserID:    userID,
		ExpiresAt: discord.Int64(time.Now().Add(memberDMExpiration).Unix()),
	})
}

// RemoveDMChannel removes a DM channel given channel id.
func (ss *SandwichState) RemoveDMChannelByChannelID(channelID discord.ChannelID) {
	ss.DmChannels.Delete(channelID)
}

// RemoveDMChannel removes a DM channel given user id.
func (ss *SandwichState) RemoveDMChannelByUserID(userID discord.UserID) {
	var channelID discord.ChannelID

	ss.DmChannels.Range(func(id discord.ChannelID, dmChannel StateDMChannel) bool {
		if dmChannel.UserID == userID {
			channelID = id
			return true
		}

		return false
	})

	if channelID != 0 {
		ss.DmChannels.Delete(channelID)
	}
}

// Runs a function on a DM channel in the cache, updating the value in cache based on returned value.
func (ss *SandwichState) UpdateDMChannelByChannelID(channelID discord.ChannelID, fn func(old StateDMChannel) StateDMChannel) (channel StateDMChannel, ok bool) {
	return ss.DmChannels.Update(channelID, fn)
}

// GetUserMutualGuilds returns a list of snowflakes of mutual guilds a member is seen on.
func (ss *SandwichState) GetUserMutualGuilds(userID discord.UserID) (guildIDs []discord.GuildID, ok bool) {
	mutualGuilds, ok := ss.Mutuals.Inner(userID)

	if !ok {
		return
	}

	// Pre-allocate the list
	guildIDs = make([]discord.GuildID, 0, mutualGuilds.Count())

	mutualGuilds.Range(func(guildID discord.GuildID, _ struct{}) bool {
		guildIDs = append(guildIDs, guildID)
		return false
	})

	return
}

// AddUserMutualGuild adds a mutual guild to a user.
//
// fake-ctx-safe
func (ss *SandwichState) AddUserMutualGuild(ctx StateCtx, userID discord.UserID, guildID discord.GuildID) {
	if !ctx.StoreMutuals {
		return
	}

	ss.Mutuals.Store(userID, guildID, struct{}{})
}

// RemoveUserMutualGuild removes a mutual guild from a user.
func (ss *SandwichState) RemoveUserMutualGuild(userID discord.UserID, guildID discord.GuildID) {
	ss.Mutuals.Delete(userID, guildID)
}

//
// VoiceState Operations
//

// ParseVoiceState parses a voice state info populating it from cache
func (ss *SandwichState) ParseVoiceState(guildID discord.GuildID, userID discord.UserID, voiceStateState discord.VoiceState) (voiceState discord.VoiceState) {
	if voiceStateState.Member == nil {
		gm, _ := ss.GetGuildMember(guildID, userID)

		voiceStateState.Member = &gm
	}

	voiceStateState.UserID = userID

	return voiceStateState
}

func (ss *SandwichState) GetVoiceState(guildID discord.GuildID, userID discord.UserID) (voiceState discord.VoiceState, ok bool) {
	stateVoiceState, ok := ss.GuildVoiceStates.Load(guildID, userID)

	if !ok {
		return
	}

	voiceState = ss.ParseVoiceState(guildID, userID, stateVoiceState)

	return
}

// UpdateVoiceState updates the voice state of a user in a guild.
//
// fake-ctx-safe
func (ss *SandwichState) UpdateVoiceState(ctx StateCtx, voiceState discord.VoiceState) {
	if voiceState.GuildID == nil {
		return
	}

	guildVoiceStates := ss.GuildVoiceStates.LoadOrNew(*voiceState.GuildID)

	beforeVoiceState, _ := ss.GetVoiceState(*voiceState.GuildID, voiceState.UserID)

	if voiceState.ChannelID == 0 {
		// Remove from voice states if leaving voice channel.
		guildVoiceStates.Delete(voiceState.UserID)
	} else {
		guildVoiceStates.Store(voiceState.UserID, ss.ParseVoiceState(*voiceState.GuildID, voiceState.UserID, voiceState))
	}

	if voiceState.Member != nil {
		ss.SetGuildMember(ctx, *voiceState.GuildID, *voiceState.Member)
	}

	// Update channel counts

	if !beforeVoiceState.ChannelID.IsNil() {
		voiceChannel, ok := ctx.Sandwich.State.GetGuildChannel(*beforeVoiceState.GuildID, beforeVoiceState.ChannelID)
		if ok {
			voiceChannel.MemberCount = ss.CountMembersForVoiceChannel(*beforeVoiceState.GuildID, voiceChannel.ID)

			if beforeVoiceState.GuildID != nil && beforeVoiceState.GuildID.IsNil() {
				ctx.Sandwich.State.SetGuildChannel(ctx, *beforeVoiceState.GuildID, voiceChannel)
			} else {
				ctx.Sandwich.State.UpdateDMChannelByChannelID(voiceChannel.ID, func(old StateDMChannel) StateDMChannel {
					old.Channel = voiceChannel
					return old
				})
			}
		}
	}

	if !voiceState.ChannelID.IsNil() {
		voiceChannel, ok := ctx.Sandwich.State.GetGuildChannel(*voiceState.GuildID, voiceState.ChannelID)
		if ok {
			voiceChannel.MemberCount = ss.CountMembersForVoiceChannel(*voiceState.GuildID, voiceChannel.ID)

			if voiceState.GuildID != nil && voiceState.GuildID.IsNil() {
				ctx.Sandwich.State.SetGuildChannel(ctx, *voiceState.GuildID, voiceChannel)
			} else {
				ctx.Sandwich.State.UpdateDMChannelByChannelID(voiceChannel.ID, func(old StateDMChannel) StateDMChannel {
					old.Channel = voiceChannel
					return old
				})
			}
		}
	}
}

func (ss *SandwichState) RemoveVoiceState(ctx StateCtx, guildID discord.GuildID, userID discord.UserID) {
	svs, ok := ss.GuildVoiceStates.Load(guildID, userID)

	if !ok {
		return
	}

	ss.GuildVoiceStates.Delete(guildID, userID)

	// Update channel counts.

	voiceChannel, ok := ss.GetGuildChannel(guildID, svs.ChannelID)
	if ok {
		voiceChannel.MemberCount = ss.CountMembersForVoiceChannel(guildID, voiceChannel.ID)

		ss.SetGuildChannel(ctx, guildID, voiceChannel)
	}
}

func (ss *SandwichState) CountMembersForVoiceChannel(guildID discord.GuildID, channelID discord.ChannelID) int32 {
	guildVoiceStates, ok := ss.GuildVoiceStates.Inner(guildID)

	if !ok {
		return 0
	}

	var count int32

	guildVoiceStates.Range(func(_ discord.UserID, voiceState discord.VoiceState) bool {
		if voiceState.ChannelID == channelID {
			count++
		}
		return false
	})

	return count
}

// Special state structs

type StateDMChannel struct {
	discord.Channel
	UserID    discord.UserID `json:"user_id"`
	ExpiresAt discord.Int64  `json:"expires_at"`
}

type StateUser struct {
	LastUpdated time.Time `json:"__sandwich_last_updated,omitempty"`
	discord.User
}
