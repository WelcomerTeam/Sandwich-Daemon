package sandwich

import (
	discord_structs "github.com/WelcomerTeam/Discord/structs"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/xerrors"
	"google.golang.org/protobuf/encoding/protojson"
)

// discord.* -> gRPC Converters.

// Converts discord.User to gRPC counterpart.
func UserToGRPC(user *discord_structs.User) (sandwichUser *User, err error) {
	userJSON, err := jsoniter.Marshal(user)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from discord.User: %v", err)
	}

	sandwichUser = &User{}

	err = protojson.Unmarshal(userJSON, sandwichUser)
	if err != nil {
		return sandwichUser, xerrors.Errorf("Failed to unmarshal to pb.User: %v", err)
	}

	return
}

// Converts discord.Guild to gRPC counterpart.
func GuildToGRPC(guild *discord_structs.Guild) (sandwichGuild *Guild, err error) {
	guildJSON, err := jsoniter.Marshal(guild)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from discord.Guild: %v", err)
	}

	sandwichGuild = &Guild{}

	err = protojson.Unmarshal(guildJSON, sandwichGuild)
	if err != nil {
		return sandwichGuild, xerrors.Errorf("Failed to unmarshal to pb.Guild: %v", err)
	}

	return
}

// Converts discord.Channel to gRPC counterpart.
func ChannelToGRPC(channel *discord_structs.Channel) (sandwichChannel *Channel, err error) {
	channelJSON, err := jsoniter.Marshal(channel)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from discord.Channel: %v", err)
	}

	sandwichChannel = &Channel{}

	err = protojson.Unmarshal(channelJSON, sandwichChannel)
	if err != nil {
		return sandwichChannel, xerrors.Errorf("Failed to unmarshal to pb.Channel: %v", err)
	}

	return
}

// Converts discord.Emoji to gRPC counterpart.
func EmojiToGRPC(emoji *discord_structs.Emoji) (sandwichEmoji *Emoji, err error) {
	emojiJSON, err := jsoniter.Marshal(emoji)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from discord.Emoji: %v", err)
	}

	sandwichEmoji = &Emoji{}

	err = protojson.Unmarshal(emojiJSON, sandwichEmoji)
	if err != nil {
		return sandwichEmoji, xerrors.Errorf("Failed to unmarshal to pb.Emoji: %v", err)
	}

	return
}

// Converts discord.GuildMember to gRPC counterpart.
func GuildMemberToGRPC(guildMember *discord_structs.GuildMember) (sandwichGuildMember *GuildMember, err error) {
	guildMemberJSON, err := jsoniter.Marshal(guildMember)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from discord.GuildMember: %v", err)
	}

	sandwichGuildMember = &GuildMember{}

	err = protojson.Unmarshal(guildMemberJSON, sandwichGuildMember)
	if err != nil {
		return sandwichGuildMember, xerrors.Errorf("Failed to unmarshal to pb.GuildMember: %v", err)
	}

	return
}

// Converts discord.Role to gRPC counterpart.
func RoleToGRPC(role *discord_structs.Role) (sandwichRole *Role, err error) {
	guildRoleJSON, err := jsoniter.Marshal(role)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from discord.Role: %v", err)
	}

	sandwichRole = &Role{}

	err = protojson.Unmarshal(guildRoleJSON, sandwichRole)
	if err != nil {
		return sandwichRole, xerrors.Errorf("Failed to unmarshal to pb.Role: %v", err)
	}

	return
}

// gRPC -> discord.* Converters.

// Converts gRPC to discord.User counterpart.
func GRPCToUser(sandwichUser *User) (user *discord_structs.User, err error) {
	userJSON, err := protojson.Marshal(sandwichUser)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from pb.User: %v", err)
	}

	user = &discord_structs.User{}

	err = jsoniter.Unmarshal(userJSON, user)
	if err != nil {
		return user, xerrors.Errorf("Failed to unmarshal to discord.User: %v", err)
	}

	return
}

// Converts gRPC to discord.Guild counterpart.
func GRPCToGuild(sandwichGuild *Guild) (guild *discord_structs.Guild, err error) {
	guildJSON, err := protojson.Marshal(sandwichGuild)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from pb.Guild: %v", err)
	}

	guild = &discord_structs.Guild{}

	err = jsoniter.Unmarshal(guildJSON, guild)
	if err != nil {
		return guild, xerrors.Errorf("Failed to unmarshal to discord.Guild: %v", err)
	}

	return
}

// Converts gRPC to discord.Channel counterpart.
func GRPCToChannel(sandwichChannel *Channel) (channel *discord_structs.Channel, err error) {
	channelJSON, err := protojson.Marshal(sandwichChannel)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from pb.Channel: %v", err)
	}

	channel = &discord_structs.Channel{}

	err = jsoniter.Unmarshal(channelJSON, channel)
	if err != nil {
		return channel, xerrors.Errorf("Failed to unmarshal to discord.Channel: %v", err)
	}

	return
}

// Converts gRPC to discord.Emoji counterpart.
func GRPCToEmoji(sandwichEmoji *Emoji) (emoji *discord_structs.Emoji, err error) {
	emojiJSON, err := protojson.Marshal(sandwichEmoji)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from pb.Emoji: %v", err)
	}

	emoji = &discord_structs.Emoji{}

	err = jsoniter.Unmarshal(emojiJSON, emoji)
	if err != nil {
		return emoji, xerrors.Errorf("Failed to unmarshal to discord.Emoji: %v", err)
	}

	return
}

// Converts gRPC to discord.GuildMember counterpart.
func GRPCToGuildMember(sandwichGuildMember *GuildMember) (guildMember *discord_structs.GuildMember, err error) {
	guildMemberJSON, err := protojson.Marshal(sandwichGuildMember)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from pb.GuildMember: %v", err)
	}

	guildMember = &discord_structs.GuildMember{}

	err = jsoniter.Unmarshal(guildMemberJSON, guildMember)
	if err != nil {
		return guildMember, xerrors.Errorf("Failed to unmarshal to discord.GuildMember: %v", err)
	}

	return
}

// Converts gRPC to discord.Role counterpart.
func GRPCToRole(sandwichRole *Role) (role *discord_structs.Role, err error) {
	guildRoleJSON, err := protojson.Marshal(sandwichRole)
	if err != nil {
		return nil, xerrors.Errorf("Failed to marshal from pb.Role: %v", err)
	}

	role = &discord_structs.Role{}

	err = jsoniter.Unmarshal(guildRoleJSON, role)
	if err != nil {
		return role, xerrors.Errorf("Failed to unmarshal to discord.Role: %v", err)
	}

	return
}
