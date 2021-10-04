package internal

import (
	"context"
	"strconv"
	"strings"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/next/protobuf"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/xerrors"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	ErrNoGuildIDPresent = xerrors.New("Missing guild ID")
	ErrNoUserIDPresent  = xerrors.New("Missing user ID")
	ErrNoQueryPresent   = xerrors.New("Missing query")

	ErrNoManagerPresent    = xerrors.New("Invalid manager identifier passed")
	ErrNoShardGroupPresent = xerrors.New("Invalid shard group identifier passed")
	ErrNoShardPresent      = xerrors.New("Invalid shard ID passed")

	ErrCacheMiss = xerrors.New("Could not find state for this guild ID")
)

func (sg *Sandwich) newSandwichServer() *routeSandwichServer {
	return &routeSandwichServer{
		sg: sg,
	}
}

type routeSandwichServer struct {
	pb.SandwichServer

	sg *Sandwich
}

func onGRPCRequest() {
	grpcCacheRequests.Inc()
}

func onGRPCHitMiss(ok bool, guild_id int64) {
	if ok {
		grpcCacheHits.WithLabelValues(strconv.FormatInt(guild_id, MagicDecimalBase)).Inc()
	} else {
		grpcCacheMisses.WithLabelValues(strconv.FormatInt(guild_id, MagicDecimalBase)).Inc()
	}
}

func onGRPCMiss(guild_id int64) {
}

// Returns if the query matches the ID or contains part of query.
func requestMatch(query string, id discord.Snowflake, name string) bool {
	return query == id.String() || strings.Contains(norm.NFKD.String(name), query)
}

// FetchConsumerConfiguration returns the Consumer Configuration.
func (grpc *routeSandwichServer) FetchConsumerConfiguration(ctx context.Context, request *pb.FetchConsumerConfigurationRequest) (response *pb.FetchConsumerConfigurationResponse, err error) {
	// TODO: Implementation for this. Are we sending a file in FS or will this be bootstrapped from the yaml?

	return &pb.FetchConsumerConfigurationResponse{}, nil
}

// FetchGuildChannels returns guilds based on the guildID.
// Takes either query or channelIDs. Empty query and empty channelIDs will return all.
func (grpc *routeSandwichServer) FetchGuildChannels(ctx context.Context, request *pb.FetchGuildChannelsRequest) (response *pb.ChannelsResponse, err error) {
	response = &pb.ChannelsResponse{
		GuildChannels: make(map[int64]*pb.Channel),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	onGRPCRequest()

	hasChannelIds := len(request.ChannelIDs) > 0
	hasQuery := request.Query != ""

	if request.GuildID == 0 {
		response.BaseResponse.Error = ErrNoGuildIDPresent.Error()

		return response, ErrNoGuildIDPresent
	}

	if hasChannelIds {
		for _, channelID := range request.ChannelIDs {
			guildChannel, ok := grpc.sg.State.GetGuildChannel((*discord.Snowflake)(&request.GuildID), discord.Snowflake(channelID))
			if ok {
				grpcChannel, err := grpc.ChannelToGRPC(guildChannel)
				if err == nil {
					response.GuildChannels[int64(guildChannel.ID)] = grpcChannel
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Channel to pb.Channel")
				}
			}

			onGRPCHitMiss(ok, request.GuildID)
		}
	} else {
		guildChannels, ok := grpc.sg.State.GetAllGuildChannels(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildChannel := range guildChannels {
			if !hasQuery || requestMatch(request.Query, guildChannel.ID, guildChannel.Name) {
				grpcChannel, err := grpc.ChannelToGRPC(guildChannel)
				if err == nil {
					response.GuildChannels[int64(guildChannel.ID)] = grpcChannel
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Channel to pb.Channel")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return
}

// FetchGuildEmojis returns emojis based on the guildID.
// Takes either query or emojiIDs. Empty query and empty emojiIDs will return all.
func (grpc *routeSandwichServer) FetchGuildEmojis(ctx context.Context, request *pb.FetchGuildEmojisRequest) (response *pb.EmojisResponse, err error) {
	response = &pb.EmojisResponse{
		GuildEmojis: make(map[int64]*pb.Emoji),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	onGRPCRequest()

	hasEmojiIds := len(request.EmojiIDs) > 0
	hasQuery := request.Query != ""

	if request.GuildID == 0 {
		response.BaseResponse.Error = ErrNoGuildIDPresent.Error()

		return response, ErrNoGuildIDPresent
	}

	if hasEmojiIds {
		for _, emojiID := range request.EmojiIDs {
			guildEmoji, ok := grpc.sg.State.GetGuildEmoji(discord.Snowflake(request.GuildID), discord.Snowflake(emojiID))
			if ok {
				grpcEmoji, err := grpc.EmojiToGRPC(guildEmoji)
				if err == nil {
					response.GuildEmojis[int64(guildEmoji.ID)] = grpcEmoji
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Emoji to pb.Emoji")
				}
			}

			onGRPCHitMiss(ok, request.GuildID)
		}
	} else {
		guildEmojis, ok := grpc.sg.State.GetAllGuildEmojis(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildEmoji := range guildEmojis {
			if !hasQuery || requestMatch(request.Query, guildEmoji.ID, guildEmoji.Name) {
				grpcEmoji, err := grpc.EmojiToGRPC(guildEmoji)
				if err == nil {
					response.GuildEmojis[int64(guildEmoji.ID)] = grpcEmoji
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Emoji to pb.Emoji")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return
}

// FetchGuildMembers returns guild members based on the guildID.
// Takes either query or userIDs. Empty query and empty userIDs will return all.
func (grpc *routeSandwichServer) FetchGuildMembers(ctx context.Context, request *pb.FetchGuildMembersRequest) (response *pb.GuildMembersResponse, err error) {
	response = &pb.GuildMembersResponse{
		GuildMembers: make(map[int64]*pb.GuildMember),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	onGRPCRequest()

	hasGuildMemberIds := len(request.UserIDs) > 0
	hasQuery := request.Query != ""

	if request.GuildID == 0 {
		response.BaseResponse.Error = ErrNoGuildIDPresent.Error()

		return response, ErrNoGuildIDPresent
	}

	if hasGuildMemberIds {
		for _, GuildMemberID := range request.UserIDs {
			guildMember, ok := grpc.sg.State.GetGuildMember(discord.Snowflake(request.GuildID), discord.Snowflake(GuildMemberID))
			if ok {
				grpcGuildMember, err := grpc.GuildMemberToGRPC(guildMember)
				if err == nil {
					response.GuildMembers[int64(guildMember.User.ID)] = grpcGuildMember
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.GuildMember to pb.GuildMember")
				}
			}

			onGRPCHitMiss(ok, request.GuildID)
		}
	} else {
		guildGuildMembers, ok := grpc.sg.State.GetAllGuildMembers(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildMember := range guildGuildMembers {
			if !hasQuery || requestMatch(request.Query, guildMember.User.ID, *guildMember.Nick+guildMember.User.Username) {
				grpcGuildMember, err := grpc.GuildMemberToGRPC(guildMember)
				if err == nil {
					response.GuildMembers[int64(guildMember.User.ID)] = grpcGuildMember
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.GuildMember to pb.GuildMember")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return
}

// FetchGuild returns guilds based on the guildIDs.
func (grpc *routeSandwichServer) FetchGuild(ctx context.Context, request *pb.FetchGuildRequest) (response *pb.GuildsResponse, err error) {
	response = &pb.GuildsResponse{
		Guilds: make(map[int64]*pb.Guild),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	onGRPCRequest()

	hasGuildIds := len(request.GuildIDs) > 0
	hasQuery := request.Query != ""

	if hasGuildIds {
		for _, guildID := range request.GuildIDs {
			guild, ok := grpc.sg.State.GetGuild(discord.Snowflake(guildID))
			if ok {
				grpcGuild, err := grpc.GuildToGRPC(guild)
				if err == nil {
					response.Guilds[int64(guild.ID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}

			onGRPCHitMiss(ok, guildID)
		}
	} else {
		if !hasQuery {
			response.BaseResponse.Error = ErrNoQueryPresent.Error()

			return response, ErrNoQueryPresent
		}

		request.Query = norm.NFKD.String(request.Query)

		grpc.sg.State.guildsMu.RLock()
		defer grpc.sg.State.guildsMu.RUnlock()

		for _, guild := range grpc.sg.State.Guilds {
			if requestMatch(request.Query, guild.ID, guild.Name) {
				grpcGuild, err := grpc.GuildToGRPC(grpc.sg.State.GuildFromState(guild))
				if err == nil {
					response.Guilds[int64(guild.ID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return
}

// FetchGuildRoles returns roles based on the roleIDs.
// Takes either query or roleIDs. Empty query and empty roleIDs will return all.
func (grpc *routeSandwichServer) FetchGuildRoles(ctx context.Context, request *pb.FetchGuildRolesRequest) (response *pb.GuildRolesResponse, err error) {
	response = &pb.GuildRolesResponse{
		GuildRoles: make(map[int64]*pb.Role),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	onGRPCRequest()

	hasRoleIds := len(request.RoleIDs) > 0
	hasQuery := request.Query != ""

	if request.GuildID == 0 {
		response.BaseResponse.Error = ErrNoGuildIDPresent.Error()
		return response, ErrNoGuildIDPresent
	}

	if hasRoleIds {
		for _, roleID := range request.RoleIDs {
			guildRole, ok := grpc.sg.State.GetGuildRole(discord.Snowflake(request.GuildID), discord.Snowflake(roleID))
			if ok {
				grpcRole, err := grpc.RoleToGRPC(guildRole)
				if err == nil {
					response.GuildRoles[int64(guildRole.ID)] = grpcRole
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Role to pb.Role")
				}
			}

			onGRPCHitMiss(ok, request.GuildID)
		}
	} else {
		guildRoles, ok := grpc.sg.State.GetAllGuildRoles(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()
			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildRole := range guildRoles {
			if !hasQuery || requestMatch(request.Query, guildRole.ID, guildRole.Name) {
				grpcRole, err := grpc.RoleToGRPC(guildRole)
				if err == nil {
					response.GuildRoles[int64(guildRole.ID)] = grpcRole
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Role to pb.Role")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return
}

// FetchMutualGuilds returns a list of all mutual guilds based on userID.
// Populates guildIDs with a list of snowflakes of all guilds.
// If expand is passed and True, will also populate guilds with the guild object.
func (grpc *routeSandwichServer) FetchMutualGuilds(ctx context.Context, request *pb.FetchMutualGuildsRequest) (response *pb.GuildsResponse, err error) {
	response = &pb.GuildsResponse{
		GuildIDs: make([]int64, 0),
		Guilds:   make(map[int64]*pb.Guild),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	onGRPCRequest()

	if request.UserID == 0 {
		response.BaseResponse.Error = ErrNoUserIDPresent.Error()
		return response, ErrNoUserIDPresent
	}

	guildIDs, _ := grpc.sg.State.GetUserMutualGuilds(discord.Snowflake(request.UserID))

	for _, guildID := range guildIDs {
		if request.Expand {
			guild, ok := grpc.sg.State.GetGuild(guildID)
			if ok {
				grpcGuild, err := grpc.GuildToGRPC(guild)
				if err == nil {
					response.Guilds[int64(guildID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}
		}
		response.GuildIDs = append(response.GuildIDs, int64(guildID))
	}

	response.BaseResponse.Ok = true

	return
}

// RequestGuildChunk sends a guild chunk request.
// Returns once the guild has been chunked.
func (grpc *routeSandwichServer) RequestGuildChunk(ctx context.Context, request *pb.RequestGuildChunkRequest) (response *pb.BaseResponse, err error) {
	response = &pb.BaseResponse{
		Ok: false,
	}

	return response, nil
}

// SendWebsocketMessage manually sends a websocket message.
func (grpc *routeSandwichServer) SendWebsocketMessage(ctx context.Context, request *pb.SendWebsocketMessageRequest) (response *pb.BaseResponse, err error) {
	response = &pb.BaseResponse{
		Ok: false,
	}

	grpc.sg.managersMu.RLock()
	manager, ok := grpc.sg.Managers[request.Manager]
	grpc.sg.managersMu.RUnlock()

	if !ok {
		response.Error = ErrNoManagerPresent.Error()
		return response, ErrNoManagerPresent
	}

	manager.shardGroupsMu.RLock()
	shardGroup, ok := manager.ShardGroups[request.ShardGroup]
	manager.shardGroupsMu.RUnlock()

	if !ok {
		response.Error = ErrNoShardGroupPresent.Error()
		return response, ErrNoShardGroupPresent
	}

	shardGroup.shardsMu.RLock()
	shard, ok := shardGroup.Shards[int(request.Shard)]
	defer shardGroup.shardsMu.RUnlock()

	if !ok {
		response.Error = ErrNoShardPresent.Error()
		return response, ErrNoShardPresent
	}

	err = shard.SendEvent(ctx, discord.GatewayOp(request.GatewayOPCode), request.Data)
	if err != nil {
		response.Error = err.Error()
		return response, err
	}

	response.Ok = true

	return response, nil
}

// WhereIsGuild returns a list of WhereIsGuildLocations based on guildId.
// WhereIsGuildLocations contains the manager, shardGroup and shardId.
func (grpc *routeSandwichServer) WhereIsGuild(ctx context.Context, request *pb.WhereIsGuildRequest) (response *pb.WhereIsGuildResponse, err error) {
	response = &pb.WhereIsGuildResponse{
		Locations: make([]*pb.WhereIsGuildLocation, 0),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	grpc.sg.managersMu.RLock()
	for _, manager := range grpc.sg.Managers {
		manager.shardGroupsMu.RLock()
		for _, shardGroup := range manager.ShardGroups {
			shardGroup.shardsMu.RLock()
			for _, shard := range shardGroup.Shards {
				shard.guildsMu.RLock()
				if _, ok := shard.Guilds[discord.Snowflake(request.GuildID)]; ok {
					response.Locations = append(response.Locations, &pb.WhereIsGuildLocation{
						Manager:    manager.Identifier.Load(),
						ShardGroup: shardGroup.ID,
						ShardId:    shard.ShardGroup.ID,
					})
				}
				shard.guildsMu.RUnlock()
			}
			shardGroup.shardsMu.RUnlock()
		}
		manager.shardGroupsMu.RUnlock()
	}
	grpc.sg.managersMu.RUnlock()

	response.BaseResponse.Ok = true

	return
}

// Converts discord.Guild to gRPC counterpart.
func (grpc *routeSandwichServer) GuildToGRPC(guild *discord.Guild) (sandwichGuild *pb.Guild, err error) {
	guildJson, err := json.Marshal(guild)
	if err != nil {
		return
	}

	sandwichGuild = &pb.Guild{}

	err = protojson.Unmarshal(guildJson, sandwichGuild)
	if err != nil {
		return sandwichGuild, err
	}

	return
}

// Converts discord.Channel to gRPC counterpart.
func (grpc *routeSandwichServer) ChannelToGRPC(channel *discord.Channel) (sandwichChannel *pb.Channel, err error) {
	channelJson, err := json.Marshal(channel)
	if err != nil {
		return
	}

	sandwichChannel = &pb.Channel{}

	err = protojson.Unmarshal(channelJson, sandwichChannel)
	if err != nil {
		return sandwichChannel, err
	}

	return
}

// Converts discord.Emoji to gRPC counterpart.
func (grpc *routeSandwichServer) EmojiToGRPC(emoji *discord.Emoji) (sandwichEmoji *pb.Emoji, err error) {
	emojiJson, err := json.Marshal(emoji)
	if err != nil {
		return
	}

	sandwichEmoji = &pb.Emoji{}

	err = protojson.Unmarshal(emojiJson, sandwichEmoji)
	if err != nil {
		return sandwichEmoji, err
	}

	return
}

// Converts discord.GuildMember to gRPC counterpart.
func (grpc *routeSandwichServer) GuildMemberToGRPC(guildMember *discord.GuildMember) (sandwichGuildMember *pb.GuildMember, err error) {
	guildMemberJson, err := json.Marshal(guildMember)
	if err != nil {
		return
	}

	sandwichGuildMember = &pb.GuildMember{}

	err = protojson.Unmarshal(guildMemberJson, sandwichGuildMember)
	if err != nil {
		return sandwichGuildMember, err
	}

	return
}

// Converts discord.Role to gRPC counterpart.
func (grpc *routeSandwichServer) RoleToGRPC(role *discord.Role) (sandwichRole *pb.Role, err error) {
	guildRole, err := json.Marshal(role)
	if err != nil {
		return
	}

	sandwichRole = &pb.Role{}

	err = protojson.Unmarshal(guildRole, sandwichRole)
	if err != nil {
		return sandwichRole, err
	}

	return
}
