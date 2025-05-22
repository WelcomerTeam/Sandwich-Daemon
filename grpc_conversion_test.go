package sandwich_test

import (
	"testing"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_daemon "github.com/WelcomerTeam/Sandwich-Daemon"
	"github.com/stretchr/testify/assert"
)

func TestGuildToPB(t *testing.T) {
	t.Parallel()

	now := time.Now()
	guild := &discord.Guild{
		ID:                          123,
		Name:                        "Test Guild",
		Icon:                        "test_icon",
		IconHash:                    "test_hash",
		Splash:                      "test_splash",
		DiscoverySplash:             "test_discovery_splash",
		Owner:                       true,
		Region:                      "us-west",
		AFKTimeout:                  300,
		WidgetEnabled:               true,
		VerificationLevel:           discord.VerificationLevelMedium,
		DefaultMessageNotifications: discord.MessageNotificationsAllMessages,
		ExplicitContentFilter:       discord.ExplicitContentFilterAllMembers,
		Features:                    []string{"feature1", "feature2"},
		MFALevel:                    discord.MFALevelElevated,
		JoinedAt:                    now,
		Large:                       true,
		Unavailable:                 false,
		MemberCount:                 100,
		MaxPresences:                1000,
		MaxMembers:                  10000,
		VanityURLCode:               "test",
		Description:                 "Test Description",
		Banner:                      "test_banner",
		PremiumSubscriptionCount:    5,
		PreferredLocale:             "en-US",
		MaxVideoChannelUsers:        25,
		ApproximateMemberCount:      100,
		ApproximatePresenceCount:    50,
		NSFWLevel:                   1,
		PremiumProgressBarEnabled:   true,
	}

	ownerID := discord.Snowflake(456)
	guild.OwnerID = &ownerID

	permissions := discord.Int64(123456)
	guild.Permissions = &permissions

	afkChannelID := discord.Snowflake(789)
	guild.AFKChannelID = &afkChannelID

	widgetChannelID := discord.Snowflake(101)
	guild.WidgetChannelID = &widgetChannelID

	applicationID := discord.Snowflake(202)
	guild.ApplicationID = &applicationID

	systemChannelID := discord.Snowflake(303)
	guild.SystemChannelID = &systemChannelID

	systemChannelFlags := discord.SystemChannelFlags(1)
	guild.SystemChannelFlags = &systemChannelFlags

	premiumTier := discord.PremiumTier(2)
	guild.PremiumTier = &premiumTier

	publicUpdatesChannelID := discord.Snowflake(404)
	guild.PublicUpdatesChannelID = &publicUpdatesChannelID

	rulesChannelID := discord.Snowflake(505)
	guild.RulesChannelID = &rulesChannelID

	pbGuild := sandwich_daemon.GuildToPB(guild)

	assert.NotNil(t, pbGuild)
	assertEqual(t, int64(123), pbGuild.ID)
	assertEqual(t, "Test Guild", pbGuild.Name)
	assertEqual(t, "test_icon", pbGuild.Icon)
	assertEqual(t, "test_hash", pbGuild.IconHash)
	assertEqual(t, "test_splash", pbGuild.Splash)
	assertEqual(t, "test_discovery_splash", pbGuild.DiscoverySplash)
	assert.True(t, pbGuild.Owner)
	assertEqual(t, "us-west", pbGuild.Region)
	assertEqual(t, int32(300), pbGuild.AFKTimeout)
	assert.True(t, pbGuild.WidgetEnabled)
	assertEqual(t, uint32(discord.VerificationLevelMedium), pbGuild.VerificationLevel)
	assertEqual(t, int32(discord.MessageNotificationsAllMessages), pbGuild.DefaultMessageNotifications)
	assertEqual(t, int32(discord.ExplicitContentFilterAllMembers), pbGuild.ExplicitContentFilter)
	assert.Equal(t, []string{"feature1", "feature2"}, pbGuild.Features)
	assertEqual(t, uint32(discord.MFALevelElevated), pbGuild.MFALevel)
	assertEqual(t, now.Format(time.RFC3339), pbGuild.JoinedAt)
	assert.True(t, pbGuild.Large)
	assert.False(t, pbGuild.Unavailable)
	assertEqual(t, int32(100), pbGuild.MemberCount)
	assertEqual(t, int32(1000), pbGuild.MaxPresences)
	assertEqual(t, int32(10000), pbGuild.MaxMembers)
	assertEqual(t, "test", pbGuild.VanityURLCode)
	assertEqual(t, "Test Description", pbGuild.Description)
	assertEqual(t, "test_banner", pbGuild.Banner)
	assertEqual(t, int32(5), pbGuild.PremiumSubscriptionCount)
	assertEqual(t, "en-US", pbGuild.PreferredLocale)
	assertEqual(t, int32(25), pbGuild.MaxVideoChannelUsers)
	assertEqual(t, int32(100), pbGuild.ApproximateMemberCount)
	assertEqual(t, int32(50), pbGuild.ApproximatePresenceCount)
	assertEqual(t, uint32(1), pbGuild.NSFWLevel)
	assert.True(t, pbGuild.PremiumProgressBarEnabled)
	assertEqual(t, int64(456), pbGuild.OwnerID)
	assertEqual(t, int64(123456), pbGuild.Permissions)
	assertEqual(t, int64(789), pbGuild.AFKChannelID)
	assertEqual(t, int64(101), pbGuild.WidgetChannelID)
	assertEqual(t, int64(202), pbGuild.ApplicationID)
	assertEqual(t, int64(303), pbGuild.SystemChannelID)
	assertEqual(t, uint32(1), pbGuild.SystemChannelFlags)
	assertEqual(t, uint32(2), pbGuild.PremiumTier)
	assertEqual(t, int64(404), pbGuild.PublicUpdatesChannelID)
	assertEqual(t, int64(505), pbGuild.RulesChannelID)
}

func TestChannelToPB(t *testing.T) {
	t.Parallel()

	channel := &discord.Channel{
		ID:                         123,
		Type:                       discord.ChannelTypeGuildText,
		Position:                   1,
		Name:                       "test-channel",
		Topic:                      "test topic",
		NSFW:                       true,
		LastMessageID:              "456",
		Bitrate:                    64000,
		UserLimit:                  10,
		RateLimitPerUser:           5,
		Icon:                       "test_icon",
		RTCRegion:                  "us-west",
		MessageCount:               100,
		MemberCount:                50,
		DefaultAutoArchiveDuration: 1440,
	}

	guildID := discord.Snowflake(789)
	channel.GuildID = &guildID

	ownerID := discord.Snowflake(101)
	channel.OwnerID = &ownerID

	applicationID := discord.Snowflake(202)
	channel.ApplicationID = &applicationID

	parentID := discord.Snowflake(303)
	channel.ParentID = &parentID

	lastPinTimestamp := time.Now()
	channel.LastPinTimestamp = &lastPinTimestamp

	videoQualityMode := discord.VideoQualityModeAuto
	channel.VideoQualityMode = &videoQualityMode

	permissions := discord.Int64(123456)
	channel.Permissions = &permissions

	pbChannel := sandwich_daemon.ChannelToPB(channel)

	assert.NotNil(t, pbChannel)
	assertEqual(t, int64(123), pbChannel.ID)
	assertEqual(t, uint32(discord.ChannelTypeGuildText), pbChannel.Type)
	assertEqual(t, int32(1), pbChannel.Position)
	assertEqual(t, "test-channel", pbChannel.Name)
	assertEqual(t, "test topic", pbChannel.Topic)
	assert.True(t, pbChannel.NSFW)
	assertEqual(t, "456", pbChannel.LastMessageID)
	assertEqual(t, int32(64000), pbChannel.Bitrate)
	assertEqual(t, int32(10), pbChannel.UserLimit)
	assertEqual(t, int32(5), pbChannel.RateLimitPerUser)
	assertEqual(t, "test_icon", pbChannel.Icon)
	assertEqual(t, "us-west", pbChannel.RTCRegion)
	assertEqual(t, int32(100), pbChannel.MessageCount)
	assertEqual(t, int32(50), pbChannel.MemberCount)
	assertEqual(t, int32(1440), pbChannel.DefaultAutoArchiveDuration)
	assertEqual(t, int64(789), pbChannel.GuildID)
	assertEqual(t, int64(101), pbChannel.OwnerID)
	assertEqual(t, int64(202), pbChannel.ApplicationID)
	assertEqual(t, int64(303), pbChannel.ParentID)
	assertEqual(t, lastPinTimestamp.Format(time.RFC3339), pbChannel.LastPinTimestamp)
	assertEqual(t, uint32(discord.VideoQualityModeAuto), pbChannel.VideoQualityMode)
	assertEqual(t, int64(123456), pbChannel.Permissions)
}

func TestUserToPB(t *testing.T) {
	t.Parallel()

	user := &discord.User{
		ID:            123,
		Username:      "testuser",
		Discriminator: "1234",
		Avatar:        "test_avatar",
		Bot:           true,
		System:        false,
		MFAEnabled:    true,
		Banner:        "test_banner",
		Locale:        "en-US",
		Verified:      true,
		Email:         "test@example.com",
		Flags:         discord.UserFlagsDiscordEmployee,
		PublicFlags:   discord.UserFlagsDiscordEmployee,
	}

	pbUser := sandwich_daemon.UserToPB(user)

	assert.NotNil(t, pbUser)
	assertEqual(t, int64(123), pbUser.ID)
	assertEqual(t, "testuser", pbUser.Username)
	assertEqual(t, "1234", pbUser.Discriminator)
	assertEqual(t, "test_avatar", pbUser.Avatar)
	assert.True(t, pbUser.Bot)
	assert.False(t, pbUser.System)
	assert.True(t, pbUser.MFAEnabled)
	assertEqual(t, "test_banner", pbUser.Banner)
	assertEqual(t, "en-US", pbUser.Locale)
	assert.True(t, pbUser.Verified)
	assertEqual(t, "test@example.com", pbUser.Email)
	assertEqual(t, int32(discord.UserFlagsDiscordEmployee), pbUser.Flags)
	assertEqual(t, int32(discord.UserFlagsDiscordEmployee), pbUser.PublicFlags)
}

func TestRoleToPB(t *testing.T) {
	t.Parallel()

	role := &discord.Role{
		ID:           123,
		Name:         "Test Role",
		Color:        16777215,
		Hoist:        true,
		Icon:         "test_icon",
		UnicodeEmoji: "😀",
		Position:     1,
		Permissions:  123456,
		Managed:      true,
		Mentionable:  true,
	}

	guildID := discord.Snowflake(456)
	role.GuildID = &guildID

	tags := &discord.RoleTag{
		PremiumSubscriber: true,
	}
	botID := discord.Snowflake(789)
	tags.BotID = &botID
	integrationID := discord.Snowflake(101)
	tags.IntegrationID = &integrationID
	role.Tags = tags

	pbRole := sandwich_daemon.RoleToPB(role)

	assert.NotNil(t, pbRole)
	assertEqual(t, int64(123), pbRole.ID)
	assertEqual(t, "Test Role", pbRole.Name)
	assertEqual(t, int32(16777215), pbRole.Color)
	assert.True(t, pbRole.Hoist)
	assertEqual(t, "test_icon", pbRole.Icon)
	assertEqual(t, "😀", pbRole.UnicodeEmoji)
	assertEqual(t, int32(1), pbRole.Position)
	assertEqual(t, int64(123456), pbRole.Permissions)
	assert.True(t, pbRole.Managed)
	assert.True(t, pbRole.Mentionable)
	assertEqual(t, int64(456), pbRole.GuildID)
	assert.NotNil(t, pbRole.Tags)
	assert.True(t, pbRole.Tags.PremiumSubscriber)
	assertEqual(t, int64(789), pbRole.Tags.BotID)
	assertEqual(t, int64(101), pbRole.Tags.IntegrationID)
}

func TestEmojiToPB(t *testing.T) {
	t.Parallel()

	emoji := &discord.Emoji{
		ID:            123,
		Name:          "test_emoji",
		Roles:         []discord.Snowflake{456, 789},
		RequireColons: true,
		Managed:       true,
		Animated:      true,
		Available:     true,
	}

	guildID := discord.Snowflake(101)
	emoji.GuildID = &guildID

	user := &discord.User{
		ID:       202,
		Username: "testuser",
	}
	emoji.User = user

	pbEmoji := sandwich_daemon.EmojiToPB(emoji)

	assert.NotNil(t, pbEmoji)
	assertEqual(t, int64(123), pbEmoji.ID)
	assertEqual(t, "test_emoji", pbEmoji.Name)
	assert.Equal(t, []int64{456, 789}, pbEmoji.Roles)
	assert.True(t, pbEmoji.RequireColons)
	assert.True(t, pbEmoji.Managed)
	assert.True(t, pbEmoji.Animated)
	assert.True(t, pbEmoji.Available)
	assertEqual(t, int64(101), pbEmoji.GuildID)
	assert.NotNil(t, pbEmoji.User)
	assertEqual(t, int64(202), pbEmoji.User.ID)
	assertEqual(t, "testuser", pbEmoji.User.Username)
}

func TestVoiceStateToPB(t *testing.T) {
	t.Parallel()

	now := time.Now()
	state := &discord.VoiceState{
		UserID:                  123,
		ChannelID:               456,
		SessionID:               "test_session",
		Deaf:                    true,
		Mute:                    true,
		SelfDeaf:                true,
		SelfMute:                true,
		SelfStream:              true,
		SelfVideo:               true,
		Suppress:                true,
		RequestToSpeakTimestamp: now,
	}

	guildID := discord.Snowflake(789)
	state.GuildID = &guildID

	member := &discord.GuildMember{
		User: &discord.User{
			ID:       123,
			Username: "testuser",
		},
		GuildID: &guildID,
	}
	state.Member = member

	pbState := sandwich_daemon.VoiceStateToPB(state)

	assert.NotNil(t, pbState)
	assertEqual(t, int64(123), pbState.UserID)
	assertEqual(t, int64(456), pbState.ChannelID)
	assertEqual(t, int64(789), pbState.GuildID)
	assertEqual(t, "test_session", pbState.SessionID)
	assert.True(t, pbState.Deaf)
	assert.True(t, pbState.Mute)
	assert.True(t, pbState.SelfDeaf)
	assert.True(t, pbState.SelfMute)
	assert.True(t, pbState.SelfStream)
	assert.True(t, pbState.SelfVideo)
	assert.True(t, pbState.Suppress)
	assertEqual(t, now.Format(time.RFC3339), pbState.RequestToSpeakTimestamp)
	assert.NotNil(t, pbState.Member)
	assertEqual(t, int64(123), pbState.Member.User.ID)
	assertEqual(t, "testuser", pbState.Member.User.Username)
	assertEqual(t, int64(789), pbState.Member.GuildID)
}

func TestActivityToPB(t *testing.T) {
	t.Parallel()

	activity := &discord.Activity{
		Name:     "Test Activity",
		Type:     0, // Playing
		URL:      "https://example.com",
		Details:  "Test Details",
		State:    "Test State",
		Instance: true,
	}

	flags := discord.ActivityFlag(1)
	activity.Flags = &flags

	timestamps := &discord.Timestamps{
		Start: 123,
		End:   456,
	}
	activity.Timestamps = timestamps

	party := &discord.Party{
		ID:   "test_party",
		Size: []int32{1, 2},
	}
	activity.Party = party

	assets := &discord.Assets{
		LargeImage: "test_large",
		LargeText:  "Test Large",
		SmallImage: "test_small",
		SmallText:  "Test Small",
	}
	activity.Assets = assets

	secrets := &discord.Secrets{
		Join:     "test_join",
		Spectate: "test_spectate",
		Match:    "test_match",
	}
	activity.Secrets = secrets

	pbActivity := sandwich_daemon.ActivityToPB(activity)

	assert.NotNil(t, pbActivity)
	assertEqual(t, "Test Activity", pbActivity.Name)
	assertEqual(t, int32(0), pbActivity.Type)
	assertEqual(t, "https://example.com", pbActivity.URL)
	assertEqual(t, "Test Details", pbActivity.Details)
	assertEqual(t, "Test State", pbActivity.State)
	assert.True(t, pbActivity.Instance)
	assertEqual(t, int32(1), pbActivity.Flags)
	assert.NotNil(t, pbActivity.Timestamps)
	assertEqual(t, int32(123), pbActivity.Timestamps.Start)
	assertEqual(t, int32(456), pbActivity.Timestamps.End)
	assert.NotNil(t, pbActivity.Party)
	assertEqual(t, "test_party", pbActivity.Party.ID)
	assert.Equal(t, []int32{1, 2}, pbActivity.Party.Size)
	assert.NotNil(t, pbActivity.Assets)
	assertEqual(t, "test_large", pbActivity.Assets.LargeImage)
	assertEqual(t, "Test Large", pbActivity.Assets.LargeText)
	assertEqual(t, "test_small", pbActivity.Assets.SmallImage)
	assertEqual(t, "Test Small", pbActivity.Assets.SmallText)
	assert.NotNil(t, pbActivity.Secrets)
	assertEqual(t, "test_join", pbActivity.Secrets.Join)
	assertEqual(t, "test_spectate", pbActivity.Secrets.Spectate)
	assertEqual(t, "test_match", pbActivity.Secrets.Match)
}

func TestStickerToPB(t *testing.T) {
	t.Parallel()

	sticker := &discord.Sticker{
		ID:          123,
		Name:        "Test Sticker",
		Description: "Test Description",
		Tags:        "test,tags",
		Type:        discord.StickerTypeStandard,
		FormatType:  discord.StickerFormatTypePNG,
		Available:   true,
		SortValue:   1,
	}

	guildID := discord.Snowflake(456)
	sticker.GuildID = &guildID

	packID := discord.Snowflake(789)
	sticker.PackID = &packID

	user := &discord.User{
		ID:       101,
		Username: "testuser",
	}
	sticker.User = user

	pbSticker := sandwich_daemon.StickerToPB(sticker)

	assert.NotNil(t, pbSticker)
	assertEqual(t, int64(123), pbSticker.ID)
	assertEqual(t, "Test Sticker", pbSticker.Name)
	assertEqual(t, "Test Description", pbSticker.Description)
	assertEqual(t, "test,tags", pbSticker.Tags)
	assertEqual(t, uint32(discord.StickerTypeStandard), pbSticker.Type)
	assertEqual(t, uint32(discord.StickerFormatTypePNG), pbSticker.FormatType)
	assert.True(t, pbSticker.Available)
	assertEqual(t, int32(1), pbSticker.SortValue)
	assertEqual(t, int64(456), pbSticker.GuildID)
	assertEqual(t, int64(789), pbSticker.PackID)
	assert.NotNil(t, pbSticker.User)
	assertEqual(t, int64(101), pbSticker.User.ID)
	assertEqual(t, "testuser", pbSticker.User.Username)
}

func TestScheduledEventToPB(t *testing.T) {
	t.Parallel()

	event := &discord.ScheduledEvent{
		ID:                 123,
		GuildID:            456,
		Name:               "Test Event",
		Description:        "Test Description",
		ScheduledStartTime: "2024-01-01T00:00:00Z",
		ScheduledEndTime:   "2024-01-02T00:00:00Z",
		PrivacyLevel:       1, // GuildOnly
		Status:             1, // Scheduled
		EntityType:         1, // StageInstance
		UserCount:          100,
	}

	channelID := discord.Snowflake(789)
	event.ChannelID = &channelID

	creatorID := discord.Snowflake(101)
	event.CreatorID = &creatorID

	entityID := discord.Snowflake(202)
	event.EntityID = &entityID

	metadata := &discord.EventMetadata{
		Location: "Test Location",
	}
	event.EntityMetadata = metadata

	creator := &discord.User{
		ID:       101,
		Username: "testuser",
	}
	event.Creator = creator

	pbEvents := sandwich_daemon.ScheduledEventsToPB([]discord.ScheduledEvent{*event})
	pbEvent := pbEvents[0]

	assert.NotNil(t, pbEvent)
	assertEqual(t, int64(123), pbEvent.ID)
	assertEqual(t, int64(456), pbEvent.GuildID)
	assertEqual(t, "Test Event", pbEvent.Name)
	assertEqual(t, "Test Description", pbEvent.Description)
	assertEqual(t, "2024-01-01T00:00:00Z", pbEvent.ScheduledStartTime)
	assertEqual(t, "2024-01-02T00:00:00Z", pbEvent.ScheduledEndTime)
	assertEqual(t, uint32(1), pbEvent.PrivacyLevel)
	assertEqual(t, uint32(1), pbEvent.Status)
	assertEqual(t, uint32(1), pbEvent.EntityType)
	assertEqual(t, int32(100), pbEvent.UserCount)
	assertEqual(t, int64(789), pbEvent.ChannelID)
	assertEqual(t, int64(101), pbEvent.CreatorID)
	assertEqual(t, int64(202), pbEvent.EntityID)
	assert.NotNil(t, pbEvent.EntityMetadata)
	assertEqual(t, "Test Location", pbEvent.EntityMetadata.Location)
	assert.NotNil(t, pbEvent.Creator)
	assertEqual(t, int64(101), pbEvent.Creator.ID)
	assertEqual(t, "testuser", pbEvent.Creator.Username)
}

func TestThreadMetadataToPB(t *testing.T) {
	t.Parallel()

	now := time.Now()
	metadata := &discord.ThreadMetadata{
		Archived:            true,
		AutoArchiveDuration: 1440,
		ArchiveTimestamp:    now,
		Locked:              true,
	}

	pbMetadata := sandwich_daemon.ThreadMetadataToPB(metadata)

	assert.NotNil(t, pbMetadata)
	assert.True(t, pbMetadata.Archived)
	assertEqual(t, int32(1440), pbMetadata.AutoArchiveDuration)
	assertEqual(t, now.Format(time.RFC3339), pbMetadata.ArchiveTimestamp)
	assert.True(t, pbMetadata.Locked)
}

func TestThreadMemberToPB(t *testing.T) {
	t.Parallel()

	now := time.Now()
	member := &discord.ThreadMember{
		JoinTimestamp: now,
		Flags:         1,
	}

	id := discord.Snowflake(123)
	member.ID = &id

	userID := discord.Snowflake(456)
	member.UserID = &userID

	guildID := discord.Snowflake(789)
	member.GuildID = &guildID

	pbMember := sandwich_daemon.ThreadMemberToPB(member)

	assert.NotNil(t, pbMember)
	assertEqual(t, int64(123), pbMember.ID)
	assertEqual(t, int64(456), pbMember.UserID)
	assertEqual(t, int64(789), pbMember.GuildID)
	assertEqual(t, now.Format(time.RFC3339), pbMember.JoinTimestamp)
	assertEqual(t, int32(1), pbMember.Flags)
}
