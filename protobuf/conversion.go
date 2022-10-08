package sandwich

import (
	"fmt"

	"github.com/WelcomerTeam/Discord/discord"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/encoding/protojson"
)

// discord.* -> gRPC Converters.

// Converts discord.User to gRPC counterpart.
func UserToGRPC(user *discord.User) (sandwichUser *User, err error) {
	userJSON, err := jsoniter.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from discord.User: %w", err)
	}

	sandwichUser = &User{}

	err = protojson.Unmarshal(userJSON, sandwichUser)
	if err != nil {
		return sandwichUser, fmt.Errorf("failed to unmarshal to pb.User: %w", err)
	}

	return
}

// Converts discord.Guild to gRPC counterpart.
func GuildToGRPC(guild *discord.Guild) (sandwichGuild *Guild, err error) {
	guildJSON, err := jsoniter.Marshal(guild)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from discord.Guild: %w", err)
	}

	sandwichGuild = &Guild{}

	err = protojson.Unmarshal(guildJSON, sandwichGuild)
	if err != nil {
		return sandwichGuild, fmt.Errorf("failed to unmarshal to pb.Guild: %w", err)
	}

	return
}

// Converts discord.Channel to gRPC counterpart.
func ChannelToGRPC(channel *discord.Channel) (sandwichChannel *Channel, err error) {
	channelJSON, err := jsoniter.Marshal(channel)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from discord.Channel: %w", err)
	}

	sandwichChannel = &Channel{}

	err = protojson.Unmarshal(channelJSON, sandwichChannel)
	if err != nil {
		return sandwichChannel, fmt.Errorf("failed to unmarshal to pb.Channel: %w", err)
	}

	return
}

// Converts discord.Emoji to gRPC counterpart.
func EmojiToGRPC(emoji *discord.Emoji) (sandwichEmoji *Emoji, err error) {
	emojiJSON, err := jsoniter.Marshal(emoji)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from discord.Emoji: %w", err)
	}

	sandwichEmoji = &Emoji{}

	err = protojson.Unmarshal(emojiJSON, sandwichEmoji)
	if err != nil {
		return sandwichEmoji, fmt.Errorf("failed to unmarshal to pb.Emoji: %w", err)
	}

	return
}

// Converts discord.GuildMember to gRPC counterpart.
func GuildMemberToGRPC(guildMember *discord.GuildMember) (sandwichGuildMember *GuildMember, err error) {
	guildMemberJSON, err := jsoniter.Marshal(guildMember)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from discord.GuildMember: %w", err)
	}

	sandwichGuildMember = &GuildMember{}

	err = protojson.Unmarshal(guildMemberJSON, sandwichGuildMember)
	if err != nil {
		return sandwichGuildMember, fmt.Errorf("failed to unmarshal to pb.GuildMember: %w", err)
	}

	return
}

// Converts discord.Role to gRPC counterpart.
func RoleToGRPC(role *discord.Role) (sandwichRole *Role, err error) {
	guildRoleJSON, err := jsoniter.Marshal(role)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from discord.Role: %w", err)
	}

	sandwichRole = &Role{}

	err = protojson.Unmarshal(guildRoleJSON, sandwichRole)
	if err != nil {
		return sandwichRole, fmt.Errorf("failed to unmarshal to pb.Role: %w", err)
	}

	return
}

// gRPC -> discord.* Converters.

// Converts gRPC to discord.User counterpart.
func GRPCToUser(sandwichUser *User) (user *discord.User, err error) {
	userJSON, err := protojson.Marshal(sandwichUser)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from pb.User: %w", err)
	}

	user = &discord.User{}

	err = jsoniter.Unmarshal(userJSON, user)
	if err != nil {
		return user, fmt.Errorf("failed to unmarshal to discord.User: %w", err)
	}

	return
}

// Converts gRPC to discord.Guild counterpart.
func GRPCToGuild(sandwichGuild *Guild) (guild *discord.Guild, err error) {
	guildJSON, err := protojson.Marshal(sandwichGuild)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from pb.Guild: %w", err)
	}

	guild = &discord.Guild{}

	err = jsoniter.Unmarshal(guildJSON, guild)
	if err != nil {
		return guild, fmt.Errorf("failed to unmarshal to discord.Guild: %w", err)
	}

	return
}

// Converts gRPC to discord.Channel counterpart.
func GRPCToChannel(sandwichChannel *Channel) (channel *discord.Channel, err error) {
	channelJSON, err := protojson.Marshal(sandwichChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from pb.Channel: %w", err)
	}

	channel = &discord.Channel{}

	err = jsoniter.Unmarshal(channelJSON, channel)
	if err != nil {
		return channel, fmt.Errorf("failed to unmarshal to discord.Channel: %w", err)
	}

	return
}

// Converts gRPC to discord.Emoji counterpart.
func GRPCToEmoji(sandwichEmoji *Emoji) (emoji *discord.Emoji, err error) {
	emojiJSON, err := protojson.Marshal(sandwichEmoji)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from pb.Emoji: %w", err)
	}

	emoji = &discord.Emoji{}

	err = jsoniter.Unmarshal(emojiJSON, emoji)
	if err != nil {
		return emoji, fmt.Errorf("failed to unmarshal to discord.Emoji: %w", err)
	}

	return
}

// Converts gRPC to discord.GuildMember counterpart.
func GRPCToGuildMember(sandwichGuildMember *GuildMember) (guildMember *discord.GuildMember, err error) {
	guildMemberJSON, err := protojson.Marshal(sandwichGuildMember)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from pb.GuildMember: %w", err)
	}

	guildMember = &discord.GuildMember{}

	err = jsoniter.Unmarshal(guildMemberJSON, guildMember)
	if err != nil {
		return guildMember, fmt.Errorf("failed to unmarshal to discord.GuildMember: %w", err)
	}

	return
}

// Converts gRPC to discord.Role counterpart.
func GRPCToRole(sandwichRole *Role) (role *discord.Role, err error) {
	guildRoleJSON, err := protojson.Marshal(sandwichRole)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal from pb.Role: %w", err)
	}

	role = &discord.Role{}

	err = jsoniter.Unmarshal(guildRoleJSON, role)
	if err != nil {
		return role, fmt.Errorf("failed to unmarshal to discord.Role: %w", err)
	}

	return
}
