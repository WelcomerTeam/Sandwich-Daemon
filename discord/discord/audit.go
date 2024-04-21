package discord

const AuditLogReasonHeader = "X-Audit-Log-Reason"

type AuditLogActionType uint8

const (
	AuditLogActionGuildUpdate AuditLogActionType = 1

	AuditLogActionChannelCreate          AuditLogActionType = 10
	AuditLogActionChannelUpdate          AuditLogActionType = 11
	AuditLogActionChannelDelete          AuditLogActionType = 12
	AuditLogActionChannelOverwriteCreate AuditLogActionType = 13
	AuditLogActionChannelOverwriteUpdate AuditLogActionType = 14
	AuditLogActionChannelOverwriteDelete AuditLogActionType = 15

	AuditLogActionMemberKick       AuditLogActionType = 20
	AuditLogActionMemberPrune      AuditLogActionType = 21
	AuditLogActionMemberBanAdd     AuditLogActionType = 22
	AuditLogActionMemberBanRemove  AuditLogActionType = 23
	AuditLogActionMemberUpdate     AuditLogActionType = 24
	AuditLogActionMemberRoleUpdate AuditLogActionType = 25
	AuditLogActionMemberMove       AuditLogActionType = 26
	AuditLogActionMemberDisconnect AuditLogActionType = 27
	AuditLogActionBotAdd           AuditLogActionType = 28

	AuditLogActionRoleCreate AuditLogActionType = 30
	AuditLogActionRoleUpdate AuditLogActionType = 31
	AuditLogActionRoleDelete AuditLogActionType = 32

	AuditLogActionInviteCreate AuditLogActionType = 40
	AuditLogActionInviteUpdate AuditLogActionType = 41
	AuditLogActionInviteDelete AuditLogActionType = 42

	AuditLogActionWebhookCreate AuditLogActionType = 50
	AuditLogActionWebhookUpdate AuditLogActionType = 51
	AuditLogActionWebhookDelete AuditLogActionType = 52

	AuditLogActionEmojiCreate AuditLogActionType = 60
	AuditLogActionEmojiUpdate AuditLogActionType = 61
	AuditLogActionEmojiDelete AuditLogActionType = 62

	AuditLogActionMessageDelete     AuditLogActionType = 72
	AuditLogActionMessageBulkDelete AuditLogActionType = 73
	AuditLogActionMessagePin        AuditLogActionType = 74
	AuditLogActionMessageUnpin      AuditLogActionType = 75

	AuditLogActionIntegrationCreate AuditLogActionType = 80
	AuditLogActionIntegrationUpdate AuditLogActionType = 81
	AuditLogActionIntegrationDelete AuditLogActionType = 82

	AuditLogActionStageInstanceCreate AuditLogActionType = 83
	AuditLogActionStageInstanceUpdate AuditLogActionType = 84
	AuditLogActionStageInstanceDelete AuditLogActionType = 85

	AuditLogActionStickerCreate AuditLogActionType = 90
	AuditLogActionStickerUpdate AuditLogActionType = 91
	AuditLogActionStickerDelete AuditLogActionType = 92

	AuditLogActionGuildScheduledEventCreate AuditLogActionType = 100
	AuditLogActionGuildScheduledEventUpdate AuditLogActionType = 101
	AuditLogActionGuildScheduledEventDelete AuditLogActionType = 102

	AuditLogActionThreadCreate AuditLogActionType = 110
	AuditLogActionThreadUpdate AuditLogActionType = 111
	AuditLogActionThreadDelete AuditLogActionType = 112
)

type AuditLogChangeKey string

const (
	AuditLogChangeKeyAfkChannelID                AuditLogChangeKey = "afk_channel_id"
	AuditLogChangeKeyAfkTimeout                  AuditLogChangeKey = "afk_timeout"
	AuditLogChangeKeyAllow                       AuditLogChangeKey = "allow"
	AuditLogChangeKeyApplicationID               AuditLogChangeKey = "application_id"
	AuditLogChangeKeyArchived                    AuditLogChangeKey = "archived"
	AuditLogChangeKeyAsset                       AuditLogChangeKey = "asset"
	AuditLogChangeKeyAutoArchiveDuration         AuditLogChangeKey = "auto_archive_duration"
	AuditLogChangeKeyAvailable                   AuditLogChangeKey = "available"
	AuditLogChangeKeyAvatarHash                  AuditLogChangeKey = "avatar_hash"
	AuditLogChangeKeyBannerHash                  AuditLogChangeKey = "banner_hash"
	AuditLogChangeKeyBitrate                     AuditLogChangeKey = "bitrate"
	AuditLogChangeKeyChannelID                   AuditLogChangeKey = "channel_id"
	AuditLogChangeKeyCode                        AuditLogChangeKey = "code"
	AuditLogChangeKeyColor                       AuditLogChangeKey = "color"
	AuditLogChangeKeyCommunicationDisabledUntil  AuditLogChangeKey = "communication_disabled_until"
	AuditLogChangeKeyDeaf                        AuditLogChangeKey = "deaf"
	AuditLogChangeKeyDefaultAutoArchiveDuration  AuditLogChangeKey = "default_auto_archive_duration"
	AuditLogChangeKeyDefaultMessageNotifications AuditLogChangeKey = "default_message_notifications"
	AuditLogChangeKeyDeny                        AuditLogChangeKey = "deny"
	AuditLogChangeKeyDescription                 AuditLogChangeKey = "description"
	AuditLogChangeKeyDiscoverySplashHash         AuditLogChangeKey = "discovery_splash_hash"
	AuditLogChangeKeyEnableEmoticons             AuditLogChangeKey = "enable_emoticons"
	AuditLogChangeKeyEntityType                  AuditLogChangeKey = "entity_type"
	AuditLogChangeKeyExpireBehavior              AuditLogChangeKey = "expire_behavior"
	AuditLogChangeKeyExpireGracePeriod           AuditLogChangeKey = "expire_grace_period"
	AuditLogChangeKeyExplicitContentFilter       AuditLogChangeKey = "explicit_content_filter"
	AuditLogChangeKeyFormatType                  AuditLogChangeKey = "format_type"
	AuditLogChangeKeyGuildID                     AuditLogChangeKey = "guild_id"
	AuditLogChangeKeyHoist                       AuditLogChangeKey = "hoist"
	AuditLogChangeKeyIconHash                    AuditLogChangeKey = "icon_hash"
	AuditLogChangeKeyID                          AuditLogChangeKey = "id"
	AuditLogChangeKeyInvitable                   AuditLogChangeKey = "invitable"
	AuditLogChangeKeyInviterID                   AuditLogChangeKey = "inviter_id"
	AuditLogChangeKeyLocation                    AuditLogChangeKey = "location"
	AuditLogChangeKeyLocked                      AuditLogChangeKey = "locked"
	AuditLogChangeKeyMaxAge                      AuditLogChangeKey = "max_age"
	AuditLogChangeKeyMaxUses                     AuditLogChangeKey = "max_uses"
	AuditLogChangeKeyMentionable                 AuditLogChangeKey = "mentionable"
	AuditLogChangeKeyMfaLevel                    AuditLogChangeKey = "mfa_level"
	AuditLogChangeKeyMute                        AuditLogChangeKey = "mute"
	AuditLogChangeKeyName                        AuditLogChangeKey = "name"
	AuditLogChangeKeyNick                        AuditLogChangeKey = "nick"
	AuditLogChangeKeyNsfw                        AuditLogChangeKey = "nsfw"
	AuditLogChangeKeyOwnerID                     AuditLogChangeKey = "owner_id"
	AuditLogChangeKeyPermissionOverwrites        AuditLogChangeKey = "permission_overwrites"
	AuditLogChangeKeyPermissions                 AuditLogChangeKey = "permissions"
	AuditLogChangeKeyPosition                    AuditLogChangeKey = "position"
	AuditLogChangeKeyPreferredLocale             AuditLogChangeKey = "preferred_locale"
	AuditLogChangeKeyPrivacyLevel                AuditLogChangeKey = "privacy_level"
	AuditLogChangeKeyPruneDeleteDays             AuditLogChangeKey = "prune_delete_days"
	AuditLogChangeKeyPublicUpdatesChannelID      AuditLogChangeKey = "public_updates_channel_id"
	AuditLogChangeKeyRateLimitPerUser            AuditLogChangeKey = "rate_limit_per_user"
	AuditLogChangeKeyRegion                      AuditLogChangeKey = "region"
	AuditLogChangeKeyRulesChannelID              AuditLogChangeKey = "rules_channel_id"
	AuditLogChangeKeySplashHash                  AuditLogChangeKey = "splash_hash"
	AuditLogChangeKeyStatus                      AuditLogChangeKey = "status"
	AuditLogChangeKeySystemChannelID             AuditLogChangeKey = "system_channel_id"
	AuditLogChangeKeyTags                        AuditLogChangeKey = "tags"
	AuditLogChangeKeyTemporary                   AuditLogChangeKey = "temporary"
	AuditLogChangeKeyTopic                       AuditLogChangeKey = "topic"
	AuditLogChangeKeyType                        AuditLogChangeKey = "type"
	AuditLogChangeKeyUnicodeEmoji                AuditLogChangeKey = "unicode_emoji"
	AuditLogChangeKeyUserLimit                   AuditLogChangeKey = "user_limit"
	AuditLogChangeKeyUses                        AuditLogChangeKey = "uses"
	AuditLogChangeKeyVanityURLCode               AuditLogChangeKey = "vanity_url_code"
	AuditLogChangeKeyVerificationLevel           AuditLogChangeKey = "verification_level"
	AuditLogChangeKeyWidgetChannelID             AuditLogChangeKey = "widget_channel_id"
	AuditLogChangeKeyWidgetEnabled               AuditLogChangeKey = "widget_enabled"
	AuditLogChangeKeyRoleAdd                     AuditLogChangeKey = "$add"
	AuditLogChangeKeyRoleRemove                  AuditLogChangeKey = "$remove"
)

type GuildAuditLog struct {
	AuditLogEntries []*AuditLogEntry  `json:"audit_log_entries"`
	ScheduledEvents []*ScheduledEvent `json:"guild_scheduled_events"`
	Integrations    []*Integration    `json:"integrations"`
	Threads         []*Channel        `json:"threads"`
	Users           []*User           `json:"users"`
	Webhooks        []*Webhook        `json:"webhooks"`
}

type AuditLogEntry struct {
	TargetID   *Snowflake         `json:"target_id,omitempty"`
	UserID     *Snowflake         `json:"user_id,omitempty"`
	Options    *AuditLogOptions   `json:"options,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	Changes    []*AuditLogChanges `json:"changes,omitempty"`
	ID         Snowflake          `json:"id"`
	ActionType AuditLogActionType `json:"action_type"`
}

type AuditLogChanges struct {
	NewValue interface{}       `json:"new_value"`
	OldValue interface{}       `json:"old_value"`
	Key      AuditLogChangeKey `json:"key"`
}

type AuditLogOptions struct {
	ChannelID        *Snowflake           `json:"channel_id,omitempty"`
	ID               *Snowflake           `json:"id,omitempty"`
	MessageID        *Snowflake           `json:"message_id,omitempty"`
	Type             *ChannelOverrideType `json:"type,omitempty"`
	RoleName         string               `json:"role_name,omitempty"`
	Count            int32                `json:"count,omitempty"`
	DeleteMemberDays int32                `json:"delete_member_days,omitempty"`
	MembersRemoved   int32                `json:"members_removed,omitempty"`
}
