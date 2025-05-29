package sandwich_test

import (
	"testing"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_protobuf "github.com/WelcomerTeam/Sandwich-Daemon/proto"
	"github.com/stretchr/testify/assert"
)

var NilDate = time.Time{}

func assertEqual[v comparable](t assert.TestingT, a, b v) {
	assert.Equal(t, a, b)
}

func TestPBToGuild(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)
	pbGuild := &sandwich_protobuf.Guild{
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
		VerificationLevel:           uint32(1), // Medium
		DefaultMessageNotifications: int32(0),  // AllMessages
		ExplicitContentFilter:       int32(1),  // AllMembers
		Features:                    []string{"feature1", "feature2"},
		MFALevel:                    uint32(1), // Elevated
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
		OwnerID:                     456,
		Permissions:                 123456,
		AFKChannelID:                789,
		WidgetChannelID:             101,
		ApplicationID:               202,
		SystemChannelID:             303,
		SystemChannelFlags:          1,
		PremiumTier:                 2,
		PublicUpdatesChannelID:      404,
		RulesChannelID:              505,
	}

	guild := sandwich_protobuf.PBToGuild(pbGuild)

	assert.NotNil(t, guild)
	assertEqual(t, discord.Snowflake(123), guild.ID)
	assertEqual(t, "Test Guild", guild.Name)
	assertEqual(t, "test_icon", guild.Icon)
	assertEqual(t, "test_hash", guild.IconHash)
	assertEqual(t, "test_splash", guild.Splash)
	assertEqual(t, "test_discovery_splash", guild.DiscoverySplash)
	assert.True(t, guild.Owner)
	assertEqual(t, "us-west", guild.Region)
	assertEqual(t, 300, guild.AFKTimeout)
	assert.True(t, guild.WidgetEnabled)
	assertEqual(t, discord.VerificationLevel(1), guild.VerificationLevel)
	assertEqual(t, discord.MessageNotificationLevel(0), guild.DefaultMessageNotifications)
	assertEqual(t, discord.ExplicitContentFilterLevel(1), guild.ExplicitContentFilter)
	assert.Equal(t, []string{"feature1", "feature2"}, guild.Features)
	assertEqual(t, discord.MFALevel(1), guild.MFALevel)
	assertEqual(t, now, guild.JoinedAt.Format(time.RFC3339))
	assert.True(t, guild.Large)
	assert.False(t, guild.Unavailable)
	assertEqual(t, 100, guild.MemberCount)
	assertEqual(t, 1000, guild.MaxPresences)
	assertEqual(t, 10000, guild.MaxMembers)
	assertEqual(t, "test", guild.VanityURLCode)
	assertEqual(t, "Test Description", guild.Description)
	assertEqual(t, "test_banner", guild.Banner)
	assertEqual(t, 5, guild.PremiumSubscriptionCount)
	assertEqual(t, "en-US", guild.PreferredLocale)
	assertEqual(t, 25, guild.MaxVideoChannelUsers)
	assertEqual(t, 100, guild.ApproximateMemberCount)
	assertEqual(t, 50, guild.ApproximatePresenceCount)
	assertEqual(t, discord.GuildNSFWLevelType(1), guild.NSFWLevel)
	assert.True(t, guild.PremiumProgressBarEnabled)
	assertEqual(t, discord.Snowflake(456), *guild.OwnerID)
	assertEqual(t, discord.Int64(123456), *guild.Permissions)
	assertEqual(t, discord.Snowflake(789), *guild.AFKChannelID)
	assertEqual(t, discord.Snowflake(101), *guild.WidgetChannelID)
	assertEqual(t, discord.Snowflake(202), *guild.ApplicationID)
	assertEqual(t, discord.Snowflake(303), *guild.SystemChannelID)
	assertEqual(t, discord.SystemChannelFlags(1), *guild.SystemChannelFlags)
	assertEqual(t, discord.PremiumTier(2), *guild.PremiumTier)
	assertEqual(t, discord.Snowflake(404), *guild.PublicUpdatesChannelID)
	assertEqual(t, discord.Snowflake(505), *guild.RulesChannelID)
}

func TestPBToChannel(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)
	pbChannel := &sandwich_protobuf.Channel{
		ID:                         123,
		Type:                       uint32(discord.ChannelTypeGuildText),
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
		GuildID:                    789,
		OwnerID:                    101,
		ApplicationID:              202,
		ParentID:                   303,
		LastPinTimestamp:           now,
		VideoQualityMode:           uint32(discord.VideoQualityModeAuto),
		Permissions:                123456,
	}

	channel := sandwich_protobuf.PBToChannel(pbChannel)

	assert.NotNil(t, channel)
	assertEqual(t, discord.Snowflake(123), channel.ID)
	assertEqual(t, discord.ChannelTypeGuildText, channel.Type)
	assertEqual(t, 1, channel.Position)
	assertEqual(t, "test-channel", channel.Name)
	assertEqual(t, "test topic", channel.Topic)
	assert.True(t, channel.NSFW)
	assertEqual(t, "456", channel.LastMessageID)
	assertEqual(t, 64000, channel.Bitrate)
	assertEqual(t, 10, channel.UserLimit)
	assertEqual(t, 5, channel.RateLimitPerUser)
	assertEqual(t, "test_icon", channel.Icon)
	assertEqual(t, "us-west", channel.RTCRegion)
	assertEqual(t, 100, channel.MessageCount)
	assertEqual(t, 50, channel.MemberCount)
	assertEqual(t, 1440, channel.DefaultAutoArchiveDuration)
	assertEqual(t, discord.Snowflake(789), *channel.GuildID)
	assertEqual(t, discord.Snowflake(101), *channel.OwnerID)
	assertEqual(t, discord.Snowflake(202), *channel.ApplicationID)
	assertEqual(t, discord.Snowflake(303), *channel.ParentID)
	assertEqual(t, now, channel.LastPinTimestamp.Format(time.RFC3339))
	assertEqual(t, discord.VideoQualityModeAuto, *channel.VideoQualityMode)
	assertEqual(t, discord.Int64(123456), *channel.Permissions)
}

func TestPBToUser(t *testing.T) {
	t.Parallel()

	pbUser := &sandwich_protobuf.User{
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
		Flags:         int32(1), // DiscordEmployee
		PublicFlags:   int32(1), // DiscordEmployee
	}

	user := sandwich_protobuf.PBToUser(pbUser)

	assert.NotNil(t, user)
	assertEqual(t, discord.Snowflake(123), user.ID)
	assertEqual(t, "testuser", user.Username)
	assertEqual(t, "1234", user.Discriminator)
	assertEqual(t, "test_avatar", user.Avatar)
	assert.True(t, user.Bot)
	assert.False(t, user.System)
	assert.True(t, user.MFAEnabled)
	assertEqual(t, "test_banner", user.Banner)
	assertEqual(t, "en-US", user.Locale)
	assert.True(t, user.Verified)
	assertEqual(t, "test@example.com", user.Email)
	assertEqual(t, discord.UserFlags(1), user.Flags)
	assertEqual(t, discord.UserFlags(1), user.PublicFlags)
}

func TestPBToRole(t *testing.T) {
	t.Parallel()

	pbRole := &sandwich_protobuf.Role{
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
		GuildID:      456,
		Tags: &sandwich_protobuf.RoleTag{
			PremiumSubscriber: true,
			BotID:             789,
			IntegrationID:     101,
		},
	}

	role := sandwich_protobuf.PBToRole(pbRole)

	assert.NotNil(t, role)
	assertEqual(t, discord.Snowflake(123), role.ID)
	assertEqual(t, "Test Role", role.Name)
	assertEqual(t, int32(16777215), role.Color)
	assert.True(t, role.Hoist)
	assertEqual(t, "test_icon", role.Icon)
	assertEqual(t, "😀", role.UnicodeEmoji)
	assertEqual(t, 1, role.Position)
	assertEqual(t, discord.Int64(123456), role.Permissions)
	assert.True(t, role.Managed)
	assert.True(t, role.Mentionable)
	assertEqual(t, discord.Snowflake(456), *role.GuildID)
	assert.NotNil(t, role.Tags)
	assert.True(t, role.Tags.PremiumSubscriber)
	assertEqual(t, discord.Snowflake(789), *role.Tags.BotID)
	assertEqual(t, discord.Snowflake(101), *role.Tags.IntegrationID)
}

func TestPBToEmoji(t *testing.T) {
	t.Parallel()

	pbEmoji := &sandwich_protobuf.Emoji{
		ID:            123,
		Name:          "test_emoji",
		Roles:         []int64{456, 789},
		RequireColons: true,
		Managed:       true,
		Animated:      true,
		Available:     true,
		GuildID:       101,
		User: &sandwich_protobuf.User{
			ID:       202,
			Username: "testuser",
		},
	}

	emoji := sandwich_protobuf.PBToEmoji(pbEmoji)

	assert.NotNil(t, emoji)
	assertEqual(t, discord.Snowflake(123), emoji.ID)
	assertEqual(t, "test_emoji", emoji.Name)
	assert.Equal(t, []discord.Snowflake{456, 789}, emoji.Roles)
	assert.True(t, emoji.RequireColons)
	assert.True(t, emoji.Managed)
	assert.True(t, emoji.Animated)
	assert.True(t, emoji.Available)
	assertEqual(t, discord.Snowflake(101), *emoji.GuildID)
	assert.NotNil(t, emoji.User)
	assertEqual(t, discord.Snowflake(202), emoji.User.ID)
	assertEqual(t, "testuser", emoji.User.Username)
}

func TestPBToVoiceState(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)
	pbState := &sandwich_protobuf.VoiceState{
		UserID:                  123,
		ChannelID:               456,
		GuildID:                 789,
		SessionID:               "test_session",
		Deaf:                    true,
		Mute:                    true,
		SelfDeaf:                true,
		SelfMute:                true,
		SelfStream:              true,
		SelfVideo:               true,
		Suppress:                true,
		RequestToSpeakTimestamp: now,
		Member: &sandwich_protobuf.GuildMember{
			User: &sandwich_protobuf.User{
				ID:       123,
				Username: "testuser",
			},
			GuildID: 789,
		},
	}

	state := sandwich_protobuf.PBToVoiceState(pbState)

	assert.NotNil(t, state)
	assertEqual(t, discord.Snowflake(123), state.UserID)
	assertEqual(t, discord.Snowflake(456), state.ChannelID)
	assertEqual(t, discord.Snowflake(789), *state.GuildID)
	assertEqual(t, "test_session", state.SessionID)
	assert.True(t, state.Deaf)
	assert.True(t, state.Mute)
	assert.True(t, state.SelfDeaf)
	assert.True(t, state.SelfMute)
	assert.True(t, state.SelfStream)
	assert.True(t, state.SelfVideo)
	assert.True(t, state.Suppress)
	assertEqual(t, now, state.RequestToSpeakTimestamp.Format(time.RFC3339))
	assert.NotNil(t, state.Member)
	assertEqual(t, discord.Snowflake(123), state.Member.User.ID)
	assertEqual(t, "testuser", state.Member.User.Username)
	assertEqual(t, discord.Snowflake(789), *state.Member.GuildID)
}

func TestPBToActivity(t *testing.T) {
	t.Parallel()

	pbActivity := &sandwich_protobuf.Activity{
		Name:     "Test Activity",
		Type:     int32(0), // Playing
		URL:      "https://example.com",
		Details:  "Test Details",
		State:    "Test State",
		Instance: true,
		Flags:    1,
		Timestamps: &sandwich_protobuf.Timestamps{
			Start: 123,
			End:   456,
		},
		Party: &sandwich_protobuf.Party{
			ID:   "test_party",
			Size: []int32{1, 2},
		},
		Assets: &sandwich_protobuf.Assets{
			LargeImage: "test_large",
			LargeText:  "Test Large",
			SmallImage: "test_small",
			SmallText:  "Test Small",
		},
		Secrets: &sandwich_protobuf.Secrets{
			Join:     "test_join",
			Spectate: "test_spectate",
			Match:    "test_match",
		},
	}

	activity := sandwich_protobuf.PBToActivity(pbActivity)

	assert.NotNil(t, activity)
	assertEqual(t, "Test Activity", activity.Name)
	assertEqual(t, discord.ActivityType(0), activity.Type)
	assertEqual(t, "https://example.com", activity.URL)
	assertEqual(t, "Test Details", activity.Details)
	assertEqual(t, "Test State", activity.State)
	assert.True(t, activity.Instance)
	assertEqual(t, discord.ActivityFlag(1), *activity.Flags)
	assert.NotNil(t, activity.Timestamps)
	assertEqual(t, 123, activity.Timestamps.Start)
	assertEqual(t, 456, activity.Timestamps.End)
	assert.NotNil(t, activity.Party)
	assertEqual(t, "test_party", activity.Party.ID)
	assert.Equal(t, []int32{1, 2}, activity.Party.Size)
	assert.NotNil(t, activity.Assets)
	assertEqual(t, "test_large", activity.Assets.LargeImage)
	assertEqual(t, "Test Large", activity.Assets.LargeText)
	assertEqual(t, "test_small", activity.Assets.SmallImage)
	assertEqual(t, "Test Small", activity.Assets.SmallText)
	assert.NotNil(t, activity.Secrets)
	assertEqual(t, "test_join", activity.Secrets.Join)
	assertEqual(t, "test_spectate", activity.Secrets.Spectate)
	assertEqual(t, "test_match", activity.Secrets.Match)
}

func TestPBToSticker(t *testing.T) {
	t.Parallel()

	pbSticker := &sandwich_protobuf.Sticker{
		ID:          123,
		Name:        "Test Sticker",
		Description: "Test Description",
		Tags:        "test,tags",
		Type:        uint32(discord.StickerTypeStandard),
		FormatType:  uint32(discord.StickerFormatTypePNG),
		Available:   true,
		SortValue:   1,
		GuildID:     456,
		PackID:      789,
		User: &sandwich_protobuf.User{
			ID:       101,
			Username: "testuser",
		},
	}

	sticker := sandwich_protobuf.PBToSticker(pbSticker)

	assert.NotNil(t, sticker)
	assertEqual(t, discord.Snowflake(123), sticker.ID)
	assertEqual(t, "Test Sticker", sticker.Name)
	assertEqual(t, "Test Description", sticker.Description)
	assertEqual(t, "test,tags", sticker.Tags)
	assertEqual(t, discord.StickerTypeStandard, sticker.Type)
	assertEqual(t, discord.StickerFormatTypePNG, sticker.FormatType)
	assert.True(t, sticker.Available)
	assertEqual(t, 1, sticker.SortValue)
	assertEqual(t, discord.Snowflake(456), *sticker.GuildID)
	assertEqual(t, discord.Snowflake(789), *sticker.PackID)
	assert.NotNil(t, sticker.User)
	assertEqual(t, discord.Snowflake(101), sticker.User.ID)
	assertEqual(t, "testuser", sticker.User.Username)
}

func TestPBToScheduledEvent(t *testing.T) {
	t.Parallel()

	pbEvent := &sandwich_protobuf.ScheduledEvent{
		ID:                 123,
		GuildID:            456,
		Name:               "Test Event",
		Description:        "Test Description",
		ScheduledStartTime: "2024-01-01T00:00:00Z",
		ScheduledEndTime:   "2024-01-02T00:00:00Z",
		PrivacyLevel:       uint32(1), // GuildOnly
		Status:             uint32(1), // Scheduled
		EntityType:         uint32(1), // StageInstance
		UserCount:          100,
		ChannelID:          789,
		CreatorID:          101,
		EntityID:           202,
		EntityMetadata: &sandwich_protobuf.EventMetadata{
			Location: "Test Location",
		},
		Creator: &sandwich_protobuf.User{
			ID:       101,
			Username: "testuser",
		},
	}

	event := sandwich_protobuf.PBToScheduledEvent(pbEvent)

	assert.NotNil(t, event)
	assertEqual(t, discord.Snowflake(123), event.ID)
	assertEqual(t, discord.Snowflake(456), event.GuildID)
	assertEqual(t, "Test Event", event.Name)
	assertEqual(t, "Test Description", event.Description)
	assertEqual(t, "2024-01-01T00:00:00Z", event.ScheduledStartTime)
	assertEqual(t, "2024-01-02T00:00:00Z", event.ScheduledEndTime)
	assertEqual(t, discord.StageChannelPrivacyLevel(1), event.PrivacyLevel)
	assertEqual(t, discord.EventStatus(1), event.Status)
	assertEqual(t, discord.ScheduledEntityType(1), event.EntityType)
	assertEqual(t, 100, event.UserCount)
	assertEqual(t, discord.Snowflake(789), *event.ChannelID)
	assertEqual(t, discord.Snowflake(101), *event.CreatorID)
	assertEqual(t, discord.Snowflake(202), *event.EntityID)
	assert.NotNil(t, event.EntityMetadata)
	assertEqual(t, "Test Location", event.EntityMetadata.Location)
	assert.NotNil(t, event.Creator)
	assertEqual(t, discord.Snowflake(101), event.Creator.ID)
	assertEqual(t, "testuser", event.Creator.Username)
}

func TestPBToThreadMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)
	pbMetadata := &sandwich_protobuf.ThreadMetadata{
		Archived:            true,
		AutoArchiveDuration: 1440,
		ArchiveTimestamp:    now,
		Locked:              true,
	}

	metadata := sandwich_protobuf.PBToThreadMetadata(pbMetadata)

	assert.NotNil(t, metadata)
	assert.True(t, metadata.Archived)
	assertEqual(t, 1440, metadata.AutoArchiveDuration)
	assertEqual(t, now, metadata.ArchiveTimestamp.Format(time.RFC3339))
	assert.True(t, metadata.Locked)
}

func TestPBToThreadMember(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)
	pbMember := &sandwich_protobuf.ThreadMember{
		ID:            123,
		UserID:        456,
		GuildID:       789,
		JoinTimestamp: now,
		Flags:         1,
	}

	member := sandwich_protobuf.PBToThreadMember(pbMember)

	assert.NotNil(t, member)
	assertEqual(t, discord.Snowflake(123), *member.ID)
	assertEqual(t, discord.Snowflake(456), *member.UserID)
	assertEqual(t, discord.Snowflake(789), *member.GuildID)
	assertEqual(t, now, member.JoinTimestamp.Format(time.RFC3339))
	assertEqual(t, 1, member.Flags)
}

func TestEmptyPBToGuild(t *testing.T) {
	t.Parallel()

	pbGuild := &sandwich_protobuf.Guild{}
	guild := sandwich_protobuf.PBToGuild(pbGuild)

	assert.NotNil(t, guild)
	assertEqual(t, discord.Snowflake(0), guild.ID)
	assertEqual(t, "", guild.Name)
	assertEqual(t, "", guild.Icon)
	assertEqual(t, "", guild.IconHash)
	assertEqual(t, "", guild.Splash)
	assertEqual(t, "", guild.DiscoverySplash)
	assert.False(t, guild.Owner)
	assertEqual(t, "", guild.Region)
	assertEqual(t, 0, guild.AFKTimeout)
	assert.False(t, guild.WidgetEnabled)
	assertEqual(t, discord.VerificationLevel(0), guild.VerificationLevel)
	assertEqual(t, discord.MessageNotificationLevel(0), guild.DefaultMessageNotifications)
	assertEqual(t, discord.ExplicitContentFilterLevel(0), guild.ExplicitContentFilter)
	assert.Nil(t, guild.Features)
	assertEqual(t, discord.MFALevel(0), guild.MFALevel)
	assertEqual(t, NilDate, guild.JoinedAt)
	assert.False(t, guild.Large)
	assert.False(t, guild.Unavailable)
	assertEqual(t, 0, guild.MemberCount)
	assertEqual(t, 0, guild.MaxPresences)
	assertEqual(t, 0, guild.MaxMembers)
	assertEqual(t, "", guild.VanityURLCode)
	assertEqual(t, "", guild.Description)
	assertEqual(t, "", guild.Banner)
	assertEqual(t, 0, guild.PremiumSubscriptionCount)
	assertEqual(t, "", guild.PreferredLocale)
	assertEqual(t, 0, guild.MaxVideoChannelUsers)
	assertEqual(t, 0, guild.ApproximateMemberCount)
	assertEqual(t, 0, guild.ApproximatePresenceCount)
	assertEqual(t, discord.GuildNSFWLevelType(0), guild.NSFWLevel)
	assert.False(t, guild.PremiumProgressBarEnabled)
	assert.Nil(t, guild.OwnerID)
	assert.Nil(t, guild.Permissions)
	assert.Nil(t, guild.AFKChannelID)
	assert.Nil(t, guild.WidgetChannelID)
	assert.Nil(t, guild.ApplicationID)
	assert.Nil(t, guild.SystemChannelID)
	assert.Nil(t, guild.SystemChannelFlags)
	assert.Nil(t, guild.PremiumTier)
	assert.Nil(t, guild.PublicUpdatesChannelID)
	assert.Nil(t, guild.RulesChannelID)
}

func TestEmptyPBToChannel(t *testing.T) {
	t.Parallel()

	pbChannel := &sandwich_protobuf.Channel{}
	channel := sandwich_protobuf.PBToChannel(pbChannel)

	assert.NotNil(t, channel)
	assertEqual(t, discord.Snowflake(0), channel.ID)
	assertEqual(t, discord.ChannelType(0), channel.Type)
	assertEqual(t, 0, channel.Position)
	assertEqual(t, "", channel.Name)
	assertEqual(t, "", channel.Topic)
	assert.False(t, channel.NSFW)
	assertEqual(t, "", channel.LastMessageID)
	assertEqual(t, 0, channel.Bitrate)
	assertEqual(t, 0, channel.UserLimit)
	assertEqual(t, 0, channel.RateLimitPerUser)
	assertEqual(t, "", channel.Icon)
	assertEqual(t, "", channel.RTCRegion)
	assertEqual(t, 0, channel.MessageCount)
	assertEqual(t, 0, channel.MemberCount)
	assertEqual(t, 0, channel.DefaultAutoArchiveDuration)
	assert.Nil(t, channel.GuildID)
	assert.Nil(t, channel.OwnerID)
	assert.Nil(t, channel.ApplicationID)
	assert.Nil(t, channel.ParentID)
	assert.Nil(t, channel.LastPinTimestamp)
	assert.Nil(t, channel.VideoQualityMode)
	assert.Nil(t, channel.Permissions)
}

func TestEmptyPBToUser(t *testing.T) {
	t.Parallel()

	pbUser := &sandwich_protobuf.User{}
	user := sandwich_protobuf.PBToUser(pbUser)

	assert.NotNil(t, user)
	assertEqual(t, discord.Snowflake(0), user.ID)
	assertEqual(t, "", user.Username)
	assertEqual(t, "", user.Discriminator)
	assertEqual(t, "", user.Avatar)
	assert.False(t, user.Bot)
	assert.False(t, user.System)
	assert.False(t, user.MFAEnabled)
	assertEqual(t, "", user.Banner)
	assertEqual(t, "", user.Locale)
	assert.False(t, user.Verified)
	assertEqual(t, "", user.Email)
	assertEqual(t, discord.UserFlags(0), user.Flags)
	assertEqual(t, discord.UserFlags(0), user.PublicFlags)
}

func TestEmptyPBToRole(t *testing.T) {
	t.Parallel()

	pbRole := &sandwich_protobuf.Role{}
	role := sandwich_protobuf.PBToRole(pbRole)

	assert.NotNil(t, role)
	assertEqual(t, discord.Snowflake(0), role.ID)
	assertEqual(t, "", role.Name)
	assertEqual(t, int32(0), role.Color)
	assert.False(t, role.Hoist)
	assertEqual(t, "", role.Icon)
	assertEqual(t, "", role.UnicodeEmoji)
	assertEqual(t, 0, role.Position)
	assertEqual(t, discord.Int64(0), role.Permissions)
	assert.False(t, role.Managed)
	assert.False(t, role.Mentionable)
	assert.Nil(t, role.GuildID)
	assert.Nil(t, role.Tags)
}

func TestEmptyPBToEmoji(t *testing.T) {
	t.Parallel()

	pbEmoji := &sandwich_protobuf.Emoji{}
	emoji := sandwich_protobuf.PBToEmoji(pbEmoji)

	assert.NotNil(t, emoji)
	assertEqual(t, discord.Snowflake(0), emoji.ID)
	assertEqual(t, "", emoji.Name)
	assert.Nil(t, emoji.Roles)
	assert.False(t, emoji.RequireColons)
	assert.False(t, emoji.Managed)
	assert.False(t, emoji.Animated)
	assert.False(t, emoji.Available)
	assert.Nil(t, emoji.GuildID)
	assert.Nil(t, emoji.User)
}

func TestEmptyPBToVoiceState(t *testing.T) {
	t.Parallel()

	pbState := &sandwich_protobuf.VoiceState{}
	state := sandwich_protobuf.PBToVoiceState(pbState)

	assert.NotNil(t, state)
	assertEqual(t, discord.Snowflake(0), state.UserID)
	assertEqual(t, discord.Snowflake(0), state.ChannelID)
	assert.Nil(t, state.GuildID)
	assertEqual(t, "", state.SessionID)
	assert.False(t, state.Deaf)
	assert.False(t, state.Mute)
	assert.False(t, state.SelfDeaf)
	assert.False(t, state.SelfMute)
	assert.False(t, state.SelfStream)
	assert.False(t, state.SelfVideo)
	assert.False(t, state.Suppress)
	assertEqual(t, NilDate, state.RequestToSpeakTimestamp)
	assert.Nil(t, state.Member)
}

func TestEmptyPBToActivity(t *testing.T) {
	t.Parallel()

	pbActivity := &sandwich_protobuf.Activity{}
	activity := sandwich_protobuf.PBToActivity(pbActivity)

	assert.NotNil(t, activity)
	assertEqual(t, "", activity.Name)
	assertEqual(t, discord.ActivityType(0), activity.Type)
	assertEqual(t, "", activity.URL)
	assertEqual(t, "", activity.Details)
	assertEqual(t, "", activity.State)
	assert.False(t, activity.Instance)
	assert.Nil(t, activity.Flags)
	assert.Nil(t, activity.Timestamps)
	assert.Nil(t, activity.Party)
	assert.Nil(t, activity.Assets)
	assert.Nil(t, activity.Secrets)
}

func TestEmptyPBToSticker(t *testing.T) {
	t.Parallel()

	pbSticker := &sandwich_protobuf.Sticker{}
	sticker := sandwich_protobuf.PBToSticker(pbSticker)

	assert.NotNil(t, sticker)
	assertEqual(t, discord.Snowflake(0), sticker.ID)
	assertEqual(t, "", sticker.Name)
	assertEqual(t, "", sticker.Description)
	assertEqual(t, "", sticker.Tags)
	assertEqual(t, discord.StickerType(0), sticker.Type)
	assertEqual(t, discord.StickerFormatType(0), sticker.FormatType)
	assert.False(t, sticker.Available)
	assertEqual(t, 0, sticker.SortValue)
	assert.Nil(t, sticker.GuildID)
	assert.Nil(t, sticker.PackID)
	assert.Nil(t, sticker.User)
}

func TestEmptyPBToScheduledEvent(t *testing.T) {
	t.Parallel()

	pbEvent := &sandwich_protobuf.ScheduledEvent{}
	event := sandwich_protobuf.PBToScheduledEvent(pbEvent)

	assert.NotNil(t, event)
	assertEqual(t, discord.Snowflake(0), event.ID)
	assertEqual(t, discord.Snowflake(0), event.GuildID)
	assertEqual(t, "", event.Name)
	assertEqual(t, "", event.Description)
	assertEqual(t, "", event.ScheduledStartTime)
	assertEqual(t, "", event.ScheduledEndTime)
	assertEqual(t, discord.StageChannelPrivacyLevel(0), event.PrivacyLevel)
	assertEqual(t, discord.EventStatus(0), event.Status)
	assertEqual(t, discord.ScheduledEntityType(0), event.EntityType)
	assertEqual(t, 0, event.UserCount)
	assert.Nil(t, event.ChannelID)
	assert.Nil(t, event.CreatorID)
	assert.Nil(t, event.EntityID)
	assert.Nil(t, event.EntityMetadata)
	assert.Nil(t, event.Creator)
}

func TestEmptyPBToThreadMetadata(t *testing.T) {
	t.Parallel()

	pbMetadata := &sandwich_protobuf.ThreadMetadata{}
	metadata := sandwich_protobuf.PBToThreadMetadata(pbMetadata)

	assert.NotNil(t, metadata)
	assert.False(t, metadata.Archived)
	assertEqual(t, 0, metadata.AutoArchiveDuration)
	assertEqual(t, NilDate, metadata.ArchiveTimestamp)
	assert.False(t, metadata.Locked)
}

func TestEmptyPBToThreadMember(t *testing.T) {
	t.Parallel()

	pbMember := &sandwich_protobuf.ThreadMember{}
	member := sandwich_protobuf.PBToThreadMember(pbMember)

	assert.NotNil(t, member)
	assert.Nil(t, member.ID)
	assert.Nil(t, member.UserID)
	assert.Nil(t, member.GuildID)
	assertEqual(t, NilDate, member.JoinTimestamp)
	assertEqual(t, 0, member.Flags)
}
