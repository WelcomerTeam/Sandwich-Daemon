package structs

import discord "github.com/WelcomerTeam/Discord/structs"

// BaseResponse represents data included in all GRPC responses.
type BaseResponse struct {
	Version string `json:"version"`

	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// Requests.

type FetchGuildRequest struct {
	GuildIDs []int64
	Query    string
}

type FetchGuildRolesRequest struct {
	GuildID int64
	RoleIDs []int64
	Query   string
}

type FetchGuildChannelsRequest struct {
	GuildID    int64
	ChannelIDs []int64
	Query      string
}

type FetchGuildEmojisRequest struct {
	GuildID  int64
	EmojiIDs []int64
	Query    string
}

type FetchGuildMembersRequest struct {
	GuildID int64
	UserIDs []int64
	Query   string
}

type FetchMutualGuildsRequest struct {
	UserID int64
	Expand bool
}

type WhereIsGuildRequest struct {
	GuildID int64
}

type FetchConsumerConfigurationRequest struct {
	Identifier string
}

type SendWebsocketMessageRequest struct {
	Manager    string
	ShardGroup int
	Shard      int

	GatewayOPCode int64
	Data          []byte
}

type RequestGuildChunkRequest struct {
	GuildID int64
}

// Responses.

type GuildRolesResponse struct {
	BaseResponse
	GuildRoles map[int64]*discord.Role
}

type ChannelsResponse struct {
	BaseResponse
	GuildChannels map[int64]*discord.Channel
}

type EmojisResponse struct {
	BaseResponse
	GuildEmojis map[int64]*discord.Emoji
}

type GuildMembersResponse struct {
	BaseResponse
	GuildMembers map[int64]*discord.GuildMember
}

type GuildsResponse struct {
	BaseResponse
	Guilds   map[int64]*discord.Guild
	GuildIDs []int64
}

type GuildResponse struct {
	BaseResponse
	Guild *discord.Guild
}

type WhereIsGuildResponse struct {
	BaseResponse
	Locations []WhereIsGuildLocation
}

type WhereIsGuildLocation struct {
	Manager    string
	ShardGroup int
	ShardID    int
}

type FetchConsumerConfigurationResponse struct {
	File []byte
}

// SendWebsocketMessage (op, data)
// RequestGuildChunk    (guild_id)
