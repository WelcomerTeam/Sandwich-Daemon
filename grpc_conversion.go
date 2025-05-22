package sandwich

import (
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/proto"
)

// Conversion functions for Discord types to protobuf types

func GuildToPB(guild *discord.Guild) *pb.Guild {
	if guild == nil {
		return nil
	}

	pbGuild := &pb.Guild{
		ID:                          int64(guild.ID),
		Name:                        guild.Name,
		Icon:                        guild.Icon,
		IconHash:                    guild.IconHash,
		Splash:                      guild.Splash,
		DiscoverySplash:             guild.DiscoverySplash,
		Owner:                       guild.Owner,
		Region:                      guild.Region,
		AFKTimeout:                  int32(guild.AFKTimeout),
		WidgetEnabled:               guild.WidgetEnabled,
		VerificationLevel:           uint32(guild.VerificationLevel),
		DefaultMessageNotifications: int32(guild.DefaultMessageNotifications),
		ExplicitContentFilter:       int32(guild.ExplicitContentFilter),
		Roles:                       RolesToPB(guild.Roles),
		Emojis:                      EmojisToPB(guild.Emojis),
		Features:                    guild.Features,
		MFALevel:                    uint32(guild.MFALevel),
		JoinedAt:                    guild.JoinedAt.Format(time.RFC3339),
		Large:                       guild.Large,
		Unavailable:                 guild.Unavailable,
		MemberCount:                 int32(guild.MemberCount),
		VoiceStates:                 VoiceStatesToPB(guild.VoiceStates),
		Members:                     GuildMembersToPB(guild.Members),
		Channels:                    ChannelsToPB(guild.Channels),
		Presences:                   ActivitiesToPB(guild.Presences),
		MaxPresences:                int32(guild.MaxPresences),
		MaxMembers:                  int32(guild.MaxMembers),
		VanityURLCode:               guild.VanityURLCode,
		Description:                 guild.Description,
		Banner:                      guild.Banner,
		PremiumSubscriptionCount:    int32(guild.PremiumSubscriptionCount),
		PreferredLocale:             guild.PreferredLocale,
		MaxVideoChannelUsers:        int32(guild.MaxVideoChannelUsers),
		ApproximateMemberCount:      int32(guild.ApproximateMemberCount),
		ApproximatePresenceCount:    int32(guild.ApproximatePresenceCount),
		NSFWLevel:                   uint32(guild.NSFWLevel),
		StageInstances:              stageInstancesToPB(guild.StageInstances),
		Stickers:                    StickersToPB(guild.Stickers),
		GuildScheduledEvents:        ScheduledEventsToPB(guild.GuildScheduledEvents),
		PremiumProgressBarEnabled:   guild.PremiumProgressBarEnabled,
		OwnerID:                     0,
		Permissions:                 0,
		AFKChannelID:                0,
		WidgetChannelID:             0,
		ApplicationID:               0,
		SystemChannelID:             0,
		SystemChannelFlags:          0,
		PremiumTier:                 0,
		PublicUpdatesChannelID:      0,
		RulesChannelID:              0,
	}

	if guild.OwnerID != nil {
		pbGuild.OwnerID = int64(*guild.OwnerID)
	}

	if guild.Permissions != nil {
		pbGuild.Permissions = int64(*guild.Permissions)
	}

	if guild.AFKChannelID != nil {
		pbGuild.AFKChannelID = int64(*guild.AFKChannelID)
	}

	if guild.WidgetChannelID != nil {
		pbGuild.WidgetChannelID = int64(*guild.WidgetChannelID)
	}

	if guild.ApplicationID != nil {
		pbGuild.ApplicationID = int64(*guild.ApplicationID)
	}

	if guild.SystemChannelID != nil {
		pbGuild.SystemChannelID = int64(*guild.SystemChannelID)
	}

	if guild.SystemChannelFlags != nil {
		pbGuild.SystemChannelFlags = uint32(*guild.SystemChannelFlags)
	}

	if guild.PremiumTier != nil {
		pbGuild.PremiumTier = uint32(*guild.PremiumTier)
	}

	if guild.PublicUpdatesChannelID != nil {
		pbGuild.PublicUpdatesChannelID = int64(*guild.PublicUpdatesChannelID)
	}

	if guild.RulesChannelID != nil {
		pbGuild.RulesChannelID = int64(*guild.RulesChannelID)
	}

	return pbGuild
}

func ChannelToPB(channel *discord.Channel) *pb.Channel {
	if channel == nil {
		return nil
	}
	return &pb.Channel{
		ID:                         int64(channel.ID),
		GuildID:                    int64(ptrSnowflake(channel.GuildID)),
		Type:                       uint32(channel.Type),
		Position:                   int32(channel.Position),
		PermissionOverwrites:       ChannelOverwritesToPB(channel.PermissionOverwrites),
		Name:                       channel.Name,
		Topic:                      channel.Topic,
		NSFW:                       channel.NSFW,
		LastMessageID:              channel.LastMessageID,
		Bitrate:                    int32(channel.Bitrate),
		UserLimit:                  int32(channel.UserLimit),
		RateLimitPerUser:           int32(channel.RateLimitPerUser),
		Recipients:                 UsersToPB(channel.Recipients),
		Icon:                       channel.Icon,
		OwnerID:                    int64(ptrSnowflake(channel.OwnerID)),
		ApplicationID:              int64(ptrSnowflake(channel.ApplicationID)),
		ParentID:                   int64(ptrSnowflake(channel.ParentID)),
		LastPinTimestamp:           ptrTimeToString(channel.LastPinTimestamp),
		RTCRegion:                  channel.RTCRegion,
		VideoQualityMode:           uint32(ptrVideoQualityMode(channel.VideoQualityMode)),
		MessageCount:               int32(channel.MessageCount),
		MemberCount:                int32(channel.MemberCount),
		ThreadMetadata:             ThreadMetadataToPB(channel.ThreadMetadata),
		ThreadMember:               ThreadMemberToPB(channel.ThreadMember),
		DefaultAutoArchiveDuration: int32(channel.DefaultAutoArchiveDuration),
		Permissions:                int64(ptrInt64(channel.Permissions)),
	}
}

func ChannelOverwritesToPB(overwrites []discord.ChannelOverwrite) []*pb.ChannelOverwrite {
	if overwrites == nil {
		return nil
	}
	pbOverwrites := make([]*pb.ChannelOverwrite, len(overwrites))
	for i, overwrite := range overwrites {
		pbOverwrites[i] = &pb.ChannelOverwrite{
			ID:    int64(overwrite.ID),
			Type:  uint32(overwrite.Type),
			Allow: int64(overwrite.Allow),
			Deny:  int64(overwrite.Deny),
		}
	}
	return pbOverwrites
}

func UsersToPB(users []discord.User) []*pb.User {
	if users == nil {
		return nil
	}
	pbUsers := make([]*pb.User, len(users))
	for i, user := range users {
		pbUsers[i] = UserToPB(&user)
	}
	return pbUsers
}

func ptrSnowflake(sf *discord.Snowflake) discord.Snowflake {
	if sf == nil {
		return 0
	}
	return *sf
}

func ptrInt64(i *discord.Int64) int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
}

func ptrTimeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func ptrVideoQualityMode(vqm *discord.VideoQualityMode) discord.VideoQualityMode {
	if vqm == nil {
		return 0
	}
	return *vqm
}

func RolesToPB(roles []discord.Role) []*pb.Role {
	if roles == nil {
		return nil
	}

	pbRoles := make([]*pb.Role, len(roles))
	for i, role := range roles {
		pbRoles[i] = RoleToPB(&role)
	}
	return pbRoles
}

func EmojisToPB(emojis []discord.Emoji) []*pb.Emoji {
	if emojis == nil {
		return nil
	}

	pbEmojis := make([]*pb.Emoji, len(emojis))
	for i, emoji := range emojis {
		pbEmojis[i] = EmojiToPB(&emoji)
	}
	return pbEmojis
}

func VoiceStatesToPB(states []discord.VoiceState) []*pb.VoiceState {
	if states == nil {
		return nil
	}

	pbStates := make([]*pb.VoiceState, len(states))
	for i, state := range states {
		pbStates[i] = VoiceStateToPB(&state)
	}
	return pbStates
}

func GuildMembersToPB(members []discord.GuildMember) []*pb.GuildMember {
	if members == nil {
		return nil
	}

	pbMembers := make([]*pb.GuildMember, len(members))
	for i, member := range members {
		pbMembers[i] = GuildMemberToPB(&member)
	}
	return pbMembers
}

func ChannelsToPB(channels []discord.Channel) []*pb.Channel {
	if channels == nil {
		return nil
	}

	pbChannels := make([]*pb.Channel, len(channels))
	for i, channel := range channels {
		pbChannels[i] = ChannelToPB(&channel)
	}
	return pbChannels
}

func ActivitiesToPB(activities []discord.Activity) []*pb.Activity {
	if activities == nil {
		return nil
	}

	pbActivities := make([]*pb.Activity, len(activities))
	for i, activity := range activities {
		pbActivities[i] = ActivityToPB(&activity)
	}
	return pbActivities
}

func StickersToPB(stickers []discord.Sticker) []*pb.Sticker {
	if stickers == nil {
		return nil
	}

	pbStickers := make([]*pb.Sticker, len(stickers))
	for i, sticker := range stickers {
		pbStickers[i] = StickerToPB(&sticker)
	}
	return pbStickers
}

func snowflakesToInt64s(snowflakes []discord.Snowflake) []int64 {
	if snowflakes == nil {
		return nil
	}

	int64s := make([]int64, len(snowflakes))
	for i, snowflake := range snowflakes {
		int64s[i] = int64(snowflake)
	}
	return int64s
}

func ThreadMetadataToPB(metadata *discord.ThreadMetadata) *pb.ThreadMetadata {
	if metadata == nil {
		return nil
	}

	var archiveTimestamp string
	if metadata.ArchiveTimestamp != (time.Time{}) {
		archiveTimestamp = metadata.ArchiveTimestamp.Format(time.RFC3339)
	}

	return &pb.ThreadMetadata{
		Archived:            metadata.Archived,
		AutoArchiveDuration: int32(metadata.AutoArchiveDuration),
		ArchiveTimestamp:    archiveTimestamp,
		Locked:              metadata.Locked,
	}
}

func ThreadMemberToPB(member *discord.ThreadMember) *pb.ThreadMember {
	if member == nil {
		return nil
	}

	var joinTimestamp string
	if member.JoinTimestamp != (time.Time{}) {
		joinTimestamp = member.JoinTimestamp.Format(time.RFC3339)
	}

	return &pb.ThreadMember{
		ID:            int64(ptrSnowflake(member.ID)),
		UserID:        int64(ptrSnowflake(member.UserID)),
		GuildID:       int64(ptrSnowflake(member.GuildID)),
		JoinTimestamp: joinTimestamp,
		Flags:         int32(member.Flags),
	}
}

func ActivityToPB(activity *discord.Activity) *pb.Activity {
	if activity == nil {
		return nil
	}

	var flags int32
	if activity.Flags != nil {
		flags = int32(*activity.Flags)
	}

	return &pb.Activity{
		Name:          activity.Name,
		Type:          int32(activity.Type),
		URL:           activity.URL,
		Timestamps:    timestampsToPB(activity.Timestamps),
		ApplicationID: int64(activity.ApplicationID),
		Details:       activity.Details,
		State:         activity.State,
		Party:         partyToPB(activity.Party),
		Assets:        assetsToPB(activity.Assets),
		Secrets:       secretsToPB(activity.Secrets),
		Instance:      activity.Instance,
		Flags:         flags,
	}
}

func timestampsToPB(timestamps *discord.Timestamps) *pb.Timestamps {
	if timestamps == nil {
		return nil
	}

	return &pb.Timestamps{
		Start: int32(timestamps.Start),
		End:   int32(timestamps.End),
	}
}

func partyToPB(party *discord.Party) *pb.Party {
	if party == nil {
		return nil
	}

	return &pb.Party{
		ID:   party.ID,
		Size: party.Size,
	}
}

func assetsToPB(assets *discord.Assets) *pb.Assets {
	if assets == nil {
		return nil
	}

	return &pb.Assets{
		LargeImage: assets.LargeImage,
		LargeText:  assets.LargeText,
		SmallImage: assets.SmallImage,
		SmallText:  assets.SmallText,
	}
}

func secretsToPB(secrets *discord.Secrets) *pb.Secrets {
	if secrets == nil {
		return nil
	}

	return &pb.Secrets{
		Join:     secrets.Join,
		Spectate: secrets.Spectate,
		Match:    secrets.Match,
	}
}

func stageInstancesToPB(instances []discord.StageInstance) []*pb.StageInstance {
	if instances == nil {
		return nil
	}

	pbInstances := make([]*pb.StageInstance, len(instances))
	for i, instance := range instances {
		pbInstances[i] = &pb.StageInstance{
			ID:                   int64(instance.ID),
			GuildID:              int64(instance.GuildID),
			ChannelID:            int64(instance.ChannelID),
			Topic:                instance.Topic,
			PrivacyLabel:         uint32(instance.PrivacyLabel),
			DiscoverableDisabled: instance.DiscoverableDisabled,
		}
	}
	return pbInstances
}

func ScheduledEventsToPB(events []discord.ScheduledEvent) []*pb.ScheduledEvent {
	if events == nil {
		return nil
	}

	pbEvents := make([]*pb.ScheduledEvent, len(events))
	for i, event := range events {
		pbEvents[i] = &pb.ScheduledEvent{
			ID:                 int64(event.ID),
			GuildID:            int64(event.GuildID),
			ChannelID:          int64(ptrSnowflake(event.ChannelID)),
			CreatorID:          int64(ptrSnowflake(event.CreatorID)),
			Name:               event.Name,
			Description:        event.Description,
			ScheduledStartTime: event.ScheduledStartTime,
			ScheduledEndTime:   event.ScheduledEndTime,
			PrivacyLevel:       uint32(event.PrivacyLevel),
			Status:             uint32(event.Status),
			EntityType:         uint32(event.EntityType),
			EntityID:           int64(ptrSnowflake(event.EntityID)),
			EntityMetadata:     eventMetadataToPB(event.EntityMetadata),
			Creator:            UserToPB(event.Creator),
			UserCount:          int32(event.UserCount),
		}
	}
	return pbEvents
}

func eventMetadataToPB(metadata *discord.EventMetadata) *pb.EventMetadata {
	if metadata == nil {
		return nil
	}

	return &pb.EventMetadata{
		Location: metadata.Location,
	}
}

func roleTagsToPB(tags *discord.RoleTag) *pb.RoleTag {
	if tags == nil {
		return nil
	}

	return &pb.RoleTag{
		PremiumSubscriber: tags.PremiumSubscriber,
		BotID:             int64(ptrSnowflake(tags.BotID)),
		IntegrationID:     int64(ptrSnowflake(tags.IntegrationID)),
	}
}

func RoleToPB(role *discord.Role) *pb.Role {
	if role == nil {
		return nil
	}

	pbRole := &pb.Role{
		ID:           int64(role.ID),
		Name:         role.Name,
		Color:        role.Color,
		Hoist:        role.Hoist,
		Icon:         role.Icon,
		UnicodeEmoji: role.UnicodeEmoji,
		Position:     int32(role.Position),
		Permissions:  int64(role.Permissions),
		Managed:      role.Managed,
		Mentionable:  role.Mentionable,
		Tags:         roleTagsToPB(role.Tags),
		GuildID:      0,
	}

	if role.GuildID != nil {
		pbRole.GuildID = int64(*role.GuildID)
	}

	return pbRole
}

func EmojiToPB(emoji *discord.Emoji) *pb.Emoji {
	if emoji == nil {
		return nil
	}

	pbEmoji := &pb.Emoji{
		ID:            int64(emoji.ID),
		Name:          emoji.Name,
		Roles:         snowflakesToInt64s(emoji.Roles),
		User:          UserToPB(emoji.User),
		RequireColons: emoji.RequireColons,
		Managed:       emoji.Managed,
		Animated:      emoji.Animated,
		Available:     emoji.Available,
		GuildID:       0,
	}

	if emoji.GuildID != nil {
		pbEmoji.GuildID = int64(*emoji.GuildID)
	}

	return pbEmoji
}

func StickerToPB(sticker *discord.Sticker) *pb.Sticker {
	if sticker == nil {
		return nil
	}

	pbSticker := &pb.Sticker{
		ID:          int64(sticker.ID),
		Name:        sticker.Name,
		Description: sticker.Description,
		Tags:        sticker.Tags,
		Type:        uint32(sticker.Type),
		FormatType:  uint32(sticker.FormatType),
		Available:   sticker.Available,
		User:        UserToPB(sticker.User),
		SortValue:   int32(sticker.SortValue),
		PackID:      0,
		GuildID:     0,
	}

	if sticker.GuildID != nil {
		pbSticker.GuildID = int64(*sticker.GuildID)
	}

	if sticker.PackID != nil {
		pbSticker.PackID = int64(*sticker.PackID)
	}

	return pbSticker
}

func VoiceStateToPB(state *discord.VoiceState) *pb.VoiceState {
	if state == nil {
		return nil
	}

	return &pb.VoiceState{
		UserID:                  int64(state.UserID),
		ChannelID:               int64(state.ChannelID),
		GuildID:                 int64(*state.GuildID),
		Member:                  GuildMemberToPB(state.Member),
		SessionID:               state.SessionID,
		Deaf:                    state.Deaf,
		Mute:                    state.Mute,
		SelfDeaf:                state.SelfDeaf,
		SelfMute:                state.SelfMute,
		SelfStream:              state.SelfStream,
		SelfVideo:               state.SelfVideo,
		Suppress:                state.Suppress,
		RequestToSpeakTimestamp: state.RequestToSpeakTimestamp.Format(time.RFC3339),
	}
}
