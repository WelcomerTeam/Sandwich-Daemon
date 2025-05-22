package sandwich

import (
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/proto"
)

// Conversion functions for protobuf types to Discord types

func PBToGuild(pbGuild *pb.Guild) *discord.Guild {
	if pbGuild == nil {
		return nil
	}

	guild := &discord.Guild{
		ID:                          discord.Snowflake(pbGuild.ID),
		Name:                        pbGuild.Name,
		Icon:                        pbGuild.Icon,
		IconHash:                    pbGuild.IconHash,
		Splash:                      pbGuild.Splash,
		DiscoverySplash:             pbGuild.DiscoverySplash,
		Owner:                       pbGuild.Owner,
		Region:                      pbGuild.Region,
		AFKTimeout:                  int32(pbGuild.AFKTimeout),
		WidgetEnabled:               pbGuild.WidgetEnabled,
		VerificationLevel:           discord.VerificationLevel(pbGuild.VerificationLevel),
		DefaultMessageNotifications: discord.MessageNotificationLevel(pbGuild.DefaultMessageNotifications),
		ExplicitContentFilter:       discord.ExplicitContentFilterLevel(pbGuild.ExplicitContentFilter),
		Roles:                       PBToRoles(pbGuild.Roles),
		Emojis:                      PBToEmojis(pbGuild.Emojis),
		Features:                    pbGuild.Features,
		MFALevel:                    discord.MFALevel(pbGuild.MFALevel),
		Large:                       pbGuild.Large,
		Unavailable:                 pbGuild.Unavailable,
		MemberCount:                 int32(pbGuild.MemberCount),
		VoiceStates:                 PBToVoiceStates(pbGuild.VoiceStates),
		Members:                     PBToGuildMembers(pbGuild.Members),
		Channels:                    PBToChannels(pbGuild.Channels),
		Presences:                   PBToActivities(pbGuild.Presences),
		MaxPresences:                int32(pbGuild.MaxPresences),
		MaxMembers:                  int32(pbGuild.MaxMembers),
		VanityURLCode:               pbGuild.VanityURLCode,
		Description:                 pbGuild.Description,
		Banner:                      pbGuild.Banner,
		PremiumSubscriptionCount:    int32(pbGuild.PremiumSubscriptionCount),
		PreferredLocale:             pbGuild.PreferredLocale,
		MaxVideoChannelUsers:        int32(pbGuild.MaxVideoChannelUsers),
		ApproximateMemberCount:      int32(pbGuild.ApproximateMemberCount),
		ApproximatePresenceCount:    int32(pbGuild.ApproximatePresenceCount),
		NSFWLevel:                   discord.GuildNSFWLevelType(pbGuild.NSFWLevel),
		StageInstances:              PBToStageInstances(pbGuild.StageInstances),
		Stickers:                    PBToStickers(pbGuild.Stickers),
		GuildScheduledEvents:        PBToScheduledEvents(pbGuild.GuildScheduledEvents),
		PremiumProgressBarEnabled:   pbGuild.PremiumProgressBarEnabled,
	}

	// Handle optional fields
	if pbGuild.OwnerID != 0 {
		ownerID := discord.Snowflake(pbGuild.OwnerID)
		guild.OwnerID = &ownerID
	}

	if pbGuild.Permissions != 0 {
		permissions := discord.Int64(pbGuild.Permissions)
		guild.Permissions = &permissions
	}

	if pbGuild.AFKChannelID != 0 {
		afkChannelID := discord.Snowflake(pbGuild.AFKChannelID)
		guild.AFKChannelID = &afkChannelID
	}

	if pbGuild.WidgetChannelID != 0 {
		widgetChannelID := discord.Snowflake(pbGuild.WidgetChannelID)
		guild.WidgetChannelID = &widgetChannelID
	}

	if pbGuild.ApplicationID != 0 {
		applicationID := discord.Snowflake(pbGuild.ApplicationID)
		guild.ApplicationID = &applicationID
	}

	if pbGuild.SystemChannelID != 0 {
		systemChannelID := discord.Snowflake(pbGuild.SystemChannelID)
		guild.SystemChannelID = &systemChannelID
	}

	if pbGuild.SystemChannelFlags != 0 {
		systemChannelFlags := discord.SystemChannelFlags(pbGuild.SystemChannelFlags)
		guild.SystemChannelFlags = &systemChannelFlags
	}

	if pbGuild.PremiumTier != 0 {
		premiumTier := discord.PremiumTier(pbGuild.PremiumTier)
		guild.PremiumTier = &premiumTier
	}

	if pbGuild.PublicUpdatesChannelID != 0 {
		publicUpdatesChannelID := discord.Snowflake(pbGuild.PublicUpdatesChannelID)
		guild.PublicUpdatesChannelID = &publicUpdatesChannelID
	}

	if pbGuild.RulesChannelID != 0 {
		rulesChannelID := discord.Snowflake(pbGuild.RulesChannelID)
		guild.RulesChannelID = &rulesChannelID
	}

	// Parse JoinedAt time
	if pbGuild.JoinedAt != "" {
		if joinedAt, err := time.Parse(time.RFC3339, pbGuild.JoinedAt); err == nil {
			guild.JoinedAt = joinedAt
		}
	}

	return guild
}

func PBToChannel(pbChannel *pb.Channel) *discord.Channel {
	if pbChannel == nil {
		return nil
	}

	channel := &discord.Channel{
		ID:                         discord.Snowflake(pbChannel.ID),
		Type:                       discord.ChannelType(pbChannel.Type),
		Position:                   int32(pbChannel.Position),
		PermissionOverwrites:       PBToChannelOverwrites(pbChannel.PermissionOverwrites),
		Name:                       pbChannel.Name,
		Topic:                      pbChannel.Topic,
		NSFW:                       pbChannel.NSFW,
		LastMessageID:              pbChannel.LastMessageID,
		Bitrate:                    int32(pbChannel.Bitrate),
		UserLimit:                  int32(pbChannel.UserLimit),
		RateLimitPerUser:           int32(pbChannel.RateLimitPerUser),
		Recipients:                 PBToUsers(pbChannel.Recipients),
		Icon:                       pbChannel.Icon,
		RTCRegion:                  pbChannel.RTCRegion,
		MessageCount:               int32(pbChannel.MessageCount),
		MemberCount:                int32(pbChannel.MemberCount),
		ThreadMetadata:             PBToThreadMetadata(pbChannel.ThreadMetadata),
		ThreadMember:               PBToThreadMember(pbChannel.ThreadMember),
		DefaultAutoArchiveDuration: int32(pbChannel.DefaultAutoArchiveDuration),
	}

	// Handle optional fields
	if pbChannel.GuildID != 0 {
		guildID := discord.Snowflake(pbChannel.GuildID)
		channel.GuildID = &guildID
	}

	if pbChannel.OwnerID != 0 {
		ownerID := discord.Snowflake(pbChannel.OwnerID)
		channel.OwnerID = &ownerID
	}

	if pbChannel.ApplicationID != 0 {
		applicationID := discord.Snowflake(pbChannel.ApplicationID)
		channel.ApplicationID = &applicationID
	}

	if pbChannel.ParentID != 0 {
		parentID := discord.Snowflake(pbChannel.ParentID)
		channel.ParentID = &parentID
	}

	if pbChannel.LastPinTimestamp != "" {
		if lastPinTimestamp, err := time.Parse(time.RFC3339, pbChannel.LastPinTimestamp); err == nil {
			channel.LastPinTimestamp = &lastPinTimestamp
		}
	}

	if pbChannel.VideoQualityMode != 0 {
		videoQualityMode := discord.VideoQualityMode(pbChannel.VideoQualityMode)
		channel.VideoQualityMode = &videoQualityMode
	}

	if pbChannel.Permissions != 0 {
		permissions := discord.Int64(pbChannel.Permissions)
		channel.Permissions = &permissions
	}

	return channel
}

func PBToChannelOverwrites(pbOverwrites []*pb.ChannelOverwrite) []discord.ChannelOverwrite {
	if pbOverwrites == nil {
		return nil
	}

	overwrites := make([]discord.ChannelOverwrite, len(pbOverwrites))
	for i, overwrite := range pbOverwrites {
		overwrites[i] = discord.ChannelOverwrite{
			ID:    discord.Snowflake(overwrite.ID),
			Type:  discord.ChannelOverrideType(overwrite.Type),
			Allow: discord.Int64(overwrite.Allow),
			Deny:  discord.Int64(overwrite.Deny),
		}
	}
	return overwrites
}

func PBToUsers(pbUsers []*pb.User) []discord.User {
	if pbUsers == nil {
		return nil
	}

	users := make([]discord.User, len(pbUsers))
	for i, user := range pbUsers {
		users[i] = *PBToUser(user)
	}
	return users
}

func PBToUser(pbUser *pb.User) *discord.User {
	if pbUser == nil {
		return nil
	}

	user := &discord.User{
		ID:            discord.Snowflake(pbUser.ID),
		Username:      pbUser.Username,
		Discriminator: pbUser.Discriminator,
		Avatar:        pbUser.Avatar,
		Bot:           pbUser.Bot,
		System:        pbUser.System,
		MFAEnabled:    pbUser.MFAEnabled,
		Banner:        pbUser.Banner,
		AccentColor:   pbUser.AccentColour,
		Locale:        pbUser.Locale,
		Verified:      pbUser.Verified,
		Email:         pbUser.Email,
		Flags:         discord.UserFlags(pbUser.Flags),
		PremiumType:   discord.UserPremiumType(pbUser.PremiumType),
		PublicFlags:   discord.UserFlags(pbUser.PublicFlags),
	}

	return user
}

func PBToRoles(pbRoles []*pb.Role) []discord.Role {
	if pbRoles == nil {
		return nil
	}

	roles := make([]discord.Role, len(pbRoles))
	for i, role := range pbRoles {
		roles[i] = *PBToRole(role)
	}
	return roles
}

func PBToRole(pbRole *pb.Role) *discord.Role {
	if pbRole == nil {
		return nil
	}

	role := &discord.Role{
		ID:           discord.Snowflake(pbRole.ID),
		Name:         pbRole.Name,
		Color:        pbRole.Color,
		Hoist:        pbRole.Hoist,
		Icon:         pbRole.Icon,
		UnicodeEmoji: pbRole.UnicodeEmoji,
		Position:     int32(pbRole.Position),
		Permissions:  discord.Int64(pbRole.Permissions),
		Managed:      pbRole.Managed,
		Mentionable:  pbRole.Mentionable,
		Tags:         PBToRoleTags(pbRole.Tags),
	}

	if pbRole.GuildID != 0 {
		guildID := discord.Snowflake(pbRole.GuildID)
		role.GuildID = &guildID
	}

	return role
}

func PBToRoleTags(pbTags *pb.RoleTag) *discord.RoleTag {
	if pbTags == nil {
		return nil
	}

	tags := &discord.RoleTag{
		PremiumSubscriber: pbTags.PremiumSubscriber,
	}

	if pbTags.BotID != 0 {
		botID := discord.Snowflake(pbTags.BotID)
		tags.BotID = &botID
	}

	if pbTags.IntegrationID != 0 {
		integrationID := discord.Snowflake(pbTags.IntegrationID)
		tags.IntegrationID = &integrationID
	}

	return tags
}

func PBToEmojis(pbEmojis []*pb.Emoji) []discord.Emoji {
	if pbEmojis == nil {
		return nil
	}

	emojis := make([]discord.Emoji, len(pbEmojis))
	for i, emoji := range pbEmojis {
		emojis[i] = *PBToEmoji(emoji)
	}
	return emojis
}

func PBToEmoji(pbEmoji *pb.Emoji) *discord.Emoji {
	if pbEmoji == nil {
		return nil
	}

	emoji := &discord.Emoji{
		ID:            discord.Snowflake(pbEmoji.ID),
		Name:          pbEmoji.Name,
		Roles:         PBToSnowflakes(pbEmoji.Roles),
		User:          PBToUser(pbEmoji.User),
		RequireColons: pbEmoji.RequireColons,
		Managed:       pbEmoji.Managed,
		Animated:      pbEmoji.Animated,
		Available:     pbEmoji.Available,
	}

	if pbEmoji.GuildID != 0 {
		guildID := discord.Snowflake(pbEmoji.GuildID)
		emoji.GuildID = &guildID
	}

	return emoji
}

func PBToSnowflakes(pbSnowflakes []int64) []discord.Snowflake {
	if pbSnowflakes == nil {
		return nil
	}

	snowflakes := make([]discord.Snowflake, len(pbSnowflakes))
	for i, snowflake := range pbSnowflakes {
		snowflakes[i] = discord.Snowflake(snowflake)
	}
	return snowflakes
}

func PBToVoiceStates(pbStates []*pb.VoiceState) []discord.VoiceState {
	if pbStates == nil {
		return nil
	}

	states := make([]discord.VoiceState, len(pbStates))
	for i, state := range pbStates {
		states[i] = *PBToVoiceState(state)
	}
	return states
}

func PBToVoiceState(pbState *pb.VoiceState) *discord.VoiceState {
	if pbState == nil {
		return nil
	}

	state := &discord.VoiceState{
		UserID:     discord.Snowflake(pbState.UserID),
		ChannelID:  discord.Snowflake(pbState.ChannelID),
		Member:     PBToGuildMember(pbState.Member),
		SessionID:  pbState.SessionID,
		Deaf:       pbState.Deaf,
		Mute:       pbState.Mute,
		SelfDeaf:   pbState.SelfDeaf,
		SelfMute:   pbState.SelfMute,
		SelfStream: pbState.SelfStream,
		SelfVideo:  pbState.SelfVideo,
		Suppress:   pbState.Suppress,
	}

	if pbState.GuildID != 0 {
		guildID := discord.Snowflake(pbState.GuildID)
		state.GuildID = &guildID
	}

	if pbState.RequestToSpeakTimestamp != "" {
		if timestamp, err := time.Parse(time.RFC3339, pbState.RequestToSpeakTimestamp); err == nil {
			state.RequestToSpeakTimestamp = timestamp
		}
	}

	return state
}

func PBToGuildMembers(pbMembers []*pb.GuildMember) []discord.GuildMember {
	if pbMembers == nil {
		return nil
	}

	members := make([]discord.GuildMember, len(pbMembers))
	for i, member := range pbMembers {
		members[i] = *PBToGuildMember(member)
	}
	return members
}

func PBToGuildMember(pbMember *pb.GuildMember) *discord.GuildMember {
	if pbMember == nil {
		return nil
	}

	member := &discord.GuildMember{
		User:    PBToUser(pbMember.User),
		GuildID: nil,
		Nick:    pbMember.Nick,
		Avatar:  pbMember.Avatar,
		Roles:   PBToSnowflakes(pbMember.Roles),
		Deaf:    pbMember.Deaf,
		Mute:    pbMember.Mute,
		Pending: pbMember.Pending,
	}

	guildIDSnowflake := discord.Snowflake(pbMember.GuildID)
	member.GuildID = &guildIDSnowflake

	if pbMember.JoinedAt != "" {
		if joinedAt, err := time.Parse(time.RFC3339, pbMember.JoinedAt); err == nil {
			member.JoinedAt = joinedAt
		}
	}

	if pbMember.PremiumSince != "" {
		member.PremiumSince = pbMember.PremiumSince
		// if premiumSince, err := time.Parse(time.RFC3339, pbMember.PremiumSince); err == nil {
		// 	member.PremiumSince = &premiumSince
		// }
	}

	if pbMember.Permissions != 0 {
		permissions := discord.Int64(pbMember.Permissions)
		member.Permissions = &permissions
	}

	if pbMember.CommunicationDisabledUntil != "" {
		member.CommunicationDisabledUntil = pbMember.CommunicationDisabledUntil
		// if disabledUntil, err := time.Parse(time.RFC3339, pbMember.CommunicationDisabledUntil); err == nil {
		// 	member.CommunicationDisabledUntil = &disabledUntil
		// }
	}

	return member
}

func PBToChannels(pbChannels []*pb.Channel) []discord.Channel {
	if pbChannels == nil {
		return nil
	}

	channels := make([]discord.Channel, len(pbChannels))
	for i, channel := range pbChannels {
		channels[i] = *PBToChannel(channel)
	}
	return channels
}

func PBToActivities(pbActivities []*pb.Activity) []discord.Activity {
	if pbActivities == nil {
		return nil
	}

	activities := make([]discord.Activity, len(pbActivities))
	for i, activity := range pbActivities {
		activities[i] = *PBToActivity(activity)
	}
	return activities
}

func PBToActivity(pbActivity *pb.Activity) *discord.Activity {
	if pbActivity == nil {
		return nil
	}

	activity := &discord.Activity{
		Name:          pbActivity.Name,
		Type:          discord.ActivityType(pbActivity.Type),
		URL:           pbActivity.URL,
		Timestamps:    PBToTimestamps(pbActivity.Timestamps),
		ApplicationID: discord.Snowflake(pbActivity.ApplicationID),
		Details:       pbActivity.Details,
		State:         pbActivity.State,
		Party:         PBToParty(pbActivity.Party),
		Assets:        PBToAssets(pbActivity.Assets),
		Secrets:       PBToSecrets(pbActivity.Secrets),
		Instance:      pbActivity.Instance,
	}

	if pbActivity.Flags != 0 {
		flags := discord.ActivityFlag(pbActivity.Flags)
		activity.Flags = &flags
	}

	return activity
}

func PBToTimestamps(pbTimestamps *pb.Timestamps) *discord.Timestamps {
	if pbTimestamps == nil {
		return nil
	}

	return &discord.Timestamps{
		Start: int32(pbTimestamps.Start),
		End:   int32(pbTimestamps.End),
	}
}

func PBToParty(pbParty *pb.Party) *discord.Party {
	if pbParty == nil {
		return nil
	}

	return &discord.Party{
		ID:   pbParty.ID,
		Size: pbParty.Size,
	}
}

func PBToAssets(pbAssets *pb.Assets) *discord.Assets {
	if pbAssets == nil {
		return nil
	}

	return &discord.Assets{
		LargeImage: pbAssets.LargeImage,
		LargeText:  pbAssets.LargeText,
		SmallImage: pbAssets.SmallImage,
		SmallText:  pbAssets.SmallText,
	}
}

func PBToSecrets(pbSecrets *pb.Secrets) *discord.Secrets {
	if pbSecrets == nil {
		return nil
	}

	return &discord.Secrets{
		Join:     pbSecrets.Join,
		Spectate: pbSecrets.Spectate,
		Match:    pbSecrets.Match,
	}
}

func PBToStageInstances(pbInstances []*pb.StageInstance) []discord.StageInstance {
	if pbInstances == nil {
		return nil
	}

	instances := make([]discord.StageInstance, len(pbInstances))
	for i, instance := range pbInstances {
		instances[i] = discord.StageInstance{
			ID:                   discord.Snowflake(instance.ID),
			GuildID:              discord.Snowflake(instance.GuildID),
			ChannelID:            discord.Snowflake(instance.ChannelID),
			Topic:                instance.Topic,
			PrivacyLabel:         discord.StageChannelPrivacyLevel(instance.PrivacyLabel),
			DiscoverableDisabled: instance.DiscoverableDisabled,
		}
	}
	return instances
}

func PBToStickers(pbStickers []*pb.Sticker) []discord.Sticker {
	if pbStickers == nil {
		return nil
	}

	stickers := make([]discord.Sticker, len(pbStickers))
	for i, sticker := range pbStickers {
		stickers[i] = *PBToSticker(sticker)
	}
	return stickers
}

func PBToSticker(pbSticker *pb.Sticker) *discord.Sticker {
	if pbSticker == nil {
		return nil
	}

	sticker := &discord.Sticker{
		ID:          discord.Snowflake(pbSticker.ID),
		Name:        pbSticker.Name,
		Description: pbSticker.Description,
		Tags:        pbSticker.Tags,
		Type:        discord.StickerType(pbSticker.Type),
		FormatType:  discord.StickerFormatType(pbSticker.FormatType),
		Available:   pbSticker.Available,
		User:        PBToUser(pbSticker.User),
		SortValue:   int32(pbSticker.SortValue),
	}

	if pbSticker.GuildID != 0 {
		guildID := discord.Snowflake(pbSticker.GuildID)
		sticker.GuildID = &guildID
	}

	if pbSticker.PackID != 0 {
		packID := discord.Snowflake(pbSticker.PackID)
		sticker.PackID = &packID
	}

	return sticker
}

func PBToScheduledEvents(pbEvents []*pb.ScheduledEvent) []discord.ScheduledEvent {
	if pbEvents == nil {
		return nil
	}

	events := make([]discord.ScheduledEvent, len(pbEvents))
	for i, event := range pbEvents {
		events[i] = *PBToScheduledEvent(event)
	}
	return events
}

func PBToScheduledEvent(pbEvent *pb.ScheduledEvent) *discord.ScheduledEvent {
	if pbEvent == nil {
		return nil
	}

	event := &discord.ScheduledEvent{
		ID:                 discord.Snowflake(pbEvent.ID),
		GuildID:            discord.Snowflake(pbEvent.GuildID),
		Name:               pbEvent.Name,
		Description:        pbEvent.Description,
		ScheduledStartTime: pbEvent.ScheduledStartTime,
		ScheduledEndTime:   pbEvent.ScheduledEndTime,
		PrivacyLevel:       discord.StageChannelPrivacyLevel(pbEvent.PrivacyLevel),
		Status:             discord.EventStatus(pbEvent.Status),
		EntityType:         discord.ScheduledEntityType(pbEvent.EntityType),
		EntityMetadata:     PBToEventMetadata(pbEvent.EntityMetadata),
		Creator:            PBToUser(pbEvent.Creator),
		UserCount:          int32(pbEvent.UserCount),
	}

	if pbEvent.ChannelID != 0 {
		channelID := discord.Snowflake(pbEvent.ChannelID)
		event.ChannelID = &channelID
	}

	if pbEvent.CreatorID != 0 {
		creatorID := discord.Snowflake(pbEvent.CreatorID)
		event.CreatorID = &creatorID
	}

	if pbEvent.EntityID != 0 {
		entityID := discord.Snowflake(pbEvent.EntityID)
		event.EntityID = &entityID
	}

	return event
}

func PBToEventMetadata(pbMetadata *pb.EventMetadata) *discord.EventMetadata {
	if pbMetadata == nil {
		return nil
	}

	return &discord.EventMetadata{
		Location: pbMetadata.Location,
	}
}

func PBToThreadMetadata(pbMetadata *pb.ThreadMetadata) *discord.ThreadMetadata {
	if pbMetadata == nil {
		return nil
	}

	metadata := &discord.ThreadMetadata{
		Archived:            pbMetadata.Archived,
		AutoArchiveDuration: int32(pbMetadata.AutoArchiveDuration),
		Locked:              pbMetadata.Locked,
	}

	if pbMetadata.ArchiveTimestamp != "" {
		if timestamp, err := time.Parse(time.RFC3339, pbMetadata.ArchiveTimestamp); err == nil {
			metadata.ArchiveTimestamp = timestamp
		}
	}

	return metadata
}

func PBToThreadMember(pbMember *pb.ThreadMember) *discord.ThreadMember {
	if pbMember == nil {
		return nil
	}

	member := &discord.ThreadMember{
		Flags: int32(pbMember.Flags),
	}

	if pbMember.ID != 0 {
		id := discord.Snowflake(pbMember.ID)
		member.ID = &id
	}

	if pbMember.UserID != 0 {
		userID := discord.Snowflake(pbMember.UserID)
		member.UserID = &userID
	}

	if pbMember.GuildID != 0 {
		guildID := discord.Snowflake(pbMember.GuildID)
		member.GuildID = &guildID
	}

	if pbMember.JoinTimestamp != "" {
		if timestamp, err := time.Parse(time.RFC3339, pbMember.JoinTimestamp); err == nil {
			member.JoinTimestamp = timestamp
		}
	}

	return member
}
