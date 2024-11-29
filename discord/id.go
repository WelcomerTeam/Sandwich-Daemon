package discord

type GuildID Snowflake

func (s *GuildID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s GuildID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type ChannelID Snowflake

func (s *ChannelID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s ChannelID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type MessageID Snowflake

func (s *MessageID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s MessageID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type UserID Snowflake

func (s *UserID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

type RoleID Snowflake

func (s *RoleID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s RoleID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type EmojiID Snowflake

func (s *EmojiID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s EmojiID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type ApplicationID Snowflake

func (s *ApplicationID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s ApplicationID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type ApplicationTeamID Snowflake

func (s *ApplicationTeamID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s ApplicationTeamID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type ApplicationCommandID Snowflake

func (s *ApplicationCommandID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s ApplicationCommandID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type ApplicationCommandPermissionsID Snowflake

func (s *ApplicationCommandPermissionsID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s ApplicationCommandPermissionsID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type IntegrationID Snowflake

func (s *IntegrationID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s IntegrationID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type AuditLogEntryID Snowflake

func (s *AuditLogEntryID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s AuditLogEntryID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type StageInstanceID Snowflake

func (s *StageInstanceID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s StageInstanceID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type WebhookID Snowflake

func (s *WebhookID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s WebhookID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type InteractionID Snowflake

func (s *InteractionID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s InteractionID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type AttachmentID Snowflake

func (s *AttachmentID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s AttachmentID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

type ScheduledEventID Snowflake

func (s *ScheduledEventID) UnmarshalJSON(b []byte) error {
	return toSnowflake(b, (*Snowflake)(s))
}

func (s ScheduledEventID) MarshalJSON() ([]byte, error) {
	return Snowflake(s).MarshalJSON()
}

// Corresponding List types
type GuildIDList List[GuildID]
type ChannelIDList List[ChannelID]
type MessageIDList List[MessageID]
type UserIDList List[UserID]
type RoleIDList List[RoleID]
type EmojiIDList List[EmojiID]
type ApplicationIDList List[ApplicationID]
type ApplicationTeamIDList List[ApplicationTeamID]
type ApplicationCommandIDList List[ApplicationCommandID]
type ApplicationCommandPermissionsIDList List[ApplicationCommandPermissionsID]
type IntegrationIDList List[IntegrationID]
type AuditLogEntryIDList List[AuditLogEntryID]
type StageInstanceIDList List[StageInstanceID]
type WebhookIDList List[WebhookID]
type InteractionIDList List[InteractionID]
type AttachmentIDList List[AttachmentID]
type ScheduledEventIDList List[ScheduledEventID]

// ID functions
func (s *GuildID) IsNil() bool {
	return *s == 0
}

func (s *ChannelID) IsNil() bool {
	return *s == 0
}
