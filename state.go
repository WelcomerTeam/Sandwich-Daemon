package sandwich

import (
	"context"

	"github.com/WelcomerTeam/Discord/discord"
)

type StateProvider interface {
	// Guilds
	GetGuild(ctx context.Context, guildID discord.Snowflake) (discord.Guild, bool)
	SetGuild(ctx context.Context, guildID discord.Snowflake, guild discord.Guild)

	// Guild Members
	GetGuildMembers(ctx context.Context, guildID discord.Snowflake) ([]discord.GuildMember, bool)

	GetGuildMember(ctx context.Context, guildID, userID discord.Snowflake) (discord.GuildMember, bool)
	SetGuildMember(ctx context.Context, guildID discord.Snowflake, member discord.GuildMember)
	RemoveGuildMember(ctx context.Context, guildID, userID discord.Snowflake)

	// Channels
	GetGuildChannels(ctx context.Context, guildID discord.Snowflake) ([]discord.Channel, bool)
	SetGuildChannels(ctx context.Context, guildID discord.Snowflake, channels []discord.Channel)

	GetGuildChannel(ctx context.Context, guildID, channelID discord.Snowflake) (discord.Channel, bool)
	SetGuildChannel(ctx context.Context, guildID discord.Snowflake, channel discord.Channel)
	RemoveGuildChannel(ctx context.Context, guildID, channelID discord.Snowflake)

	// Roles
	GetGuildRoles(ctx context.Context, guildID discord.Snowflake) ([]discord.Role, bool)
	SetGuildRoles(ctx context.Context, guildID discord.Snowflake, roles []discord.Role)

	GetGuildRole(ctx context.Context, guildID, roleID discord.Snowflake) (discord.Role, bool)
	SetGuildRole(ctx context.Context, guildID discord.Snowflake, role discord.Role)
	RemoveGuildRole(ctx context.Context, guildID, roleID discord.Snowflake)

	// Emojis
	GetGuildEmojis(ctx context.Context, guildID discord.Snowflake) ([]discord.Emoji, bool)
	SetGuildEmojis(ctx context.Context, guildID discord.Snowflake, emojis []discord.Emoji)

	GetGuildEmoji(ctx context.Context, guildID, emojiID discord.Snowflake) (discord.Emoji, bool)
	SetGuildEmoji(ctx context.Context, guildID discord.Snowflake, emoji discord.Emoji)
	RemoveGuildEmoji(ctx context.Context, guildID, emojiID discord.Snowflake)

	// Voice States
	GetVoiceStates(ctx context.Context, guildID discord.Snowflake) ([]discord.VoiceState, bool)

	GetVoiceState(ctx context.Context, guildID, userID discord.Snowflake) (discord.VoiceState, bool)
	SetVoiceState(ctx context.Context, guildID discord.Snowflake, voiceState discord.VoiceState)
	RemoveVoiceState(ctx context.Context, guildID, userID discord.Snowflake)

	// Users
	GetUser(ctx context.Context, userID discord.Snowflake) (discord.User, bool)
	SetUser(ctx context.Context, userID discord.Snowflake, user discord.User)

	// User Mutuals
	GetUserMutualGuilds(ctx context.Context, userID discord.Snowflake) ([]discord.Snowflake, bool)
	AddUserMutualGuild(ctx context.Context, userID, guildID discord.Snowflake)
	RemoveUserMutualGuild(ctx context.Context, userID, guildID discord.Snowflake)
}
