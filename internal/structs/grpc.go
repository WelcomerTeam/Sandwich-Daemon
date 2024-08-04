package structs

import "github.com/WelcomerTeam/Sandwich-Daemon/discord"

// BaseResponse represents data included in all GRPC responses.
type BaseResponse struct {
	Version string `json:"version"`

	Error string `json:"error"`

	Ok bool `json:"ok"`
}

// Requests.

type FetchGuildRequest struct {
	Query    string
	GuildIDs []int64
}

type FetchGuildRolesRequest struct {
	Query   string
	RoleIDs []int64
	GuildID int64
}

type FetchGuildChannelsRequest struct {
	Query      string
	ChannelIDs []int64
	GuildID    int64
}

type FetchGuildEmojisRequest struct {
	Query    string
	EmojiIDs []int64
	GuildID  int64
}

type FetchGuildMembersRequest struct {
	Query   string
	UserIDs []int64
	GuildID int64
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
	Data       []byte
	ShardGroup int
	Shard      int

	GatewayOPCode int64
}

type RequestGuildChunkRequest struct {
	GuildID int64
}

// Responses.

type GuildRolesResponse struct {
	GuildRoles map[int64]*discord.Role
	BaseResponse
}

type ChannelsResponse struct {
	GuildChannels map[int64]*discord.Channel
	BaseResponse
}

type EmojisResponse struct {
	GuildEmojis map[int64]*discord.Emoji
	BaseResponse
}

type GuildMembersResponse struct {
	GuildMembers map[int64]*discord.GuildMember
	BaseResponse
}

type GuildsResponse struct {
	BaseResponse
	Guilds   map[int64]*discord.Guild
	GuildIDs []int64
}

type GuildResponse struct {
	Guild *discord.Guild
	BaseResponse
}

type WhereIsGuildResponse struct {
	BaseResponse
	Locations []WhereIsGuildLocation
}

type WhereIsGuildLocation struct {
	GuildMember *discord.GuildMember
	Manager     string
	ShardGroup  int
	ShardID     int
}

type FetchConsumerConfigurationResponse struct {
	File []byte
}

// SendWebsocketMessage (op, data)
// RequestGuildChunk    (guild_id)
