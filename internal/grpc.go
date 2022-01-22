package internal

import (
	"bytes"
	"context"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/protobuf"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/xerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"strings"
	"time"
)

var (
	ErrNoGuildIDPresent = xerrors.New("Missing guild ID")
	ErrNoUserIDPresent  = xerrors.New("Missing user ID")
	ErrNoQueryPresent   = xerrors.New("Missing query")

	ErrDuplicateManagerPresent = xerrors.New("Duplicate manager identifier passed")
	ErrNoManagerPresent        = xerrors.New("Invalid manager identifier passed")
	ErrNoShardGroupPresent     = xerrors.New("Invalid shard group identifier passed")
	ErrNoShardPresent          = xerrors.New("Invalid shard ID passed")

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

func onGRPCHit(ok bool) {
	if ok {
		grpcCacheHits.Inc()
	} else {
		grpcCacheMisses.Inc()
	}
}

// Returns if the query matches the matches input.
func requestMatch(query string, matches ...string) bool {
	for _, match := range matches {
		if query == match || strings.Contains(norm.NFKD.String(match), query) {
			return true
		}
	}

	return false
}

// Listen delivers information to consumers.
func (grpc *routeSandwichServer) Listen(request *pb.ListenRequest, listener pb.Sandwich_ListenServer) (err error) {
	onGRPCRequest()

	globalPoolID := grpc.sg.globalPoolAccumulator.Add(1)
	channel := make(chan []byte)

	grpc.sg.globalPoolMu.Lock()
	grpc.sg.globalPool[globalPoolID] = channel
	grpc.sg.globalPoolMu.Unlock()

	defer func() {
		grpc.sg.globalPoolMu.Lock()
		delete(grpc.sg.globalPool, globalPoolID)
		grpc.sg.globalPoolMu.Unlock()

		close(channel)
	}()

	ctx := listener.Context()

	for {
		select {
		case msg := <-channel:
			err = listener.Send(&pb.ListenResponse{
				Timestamp: time.Now().Unix(),
				Data:      msg,
			})
			if err != nil {
				grpc.sg.Logger.Error().Err(err).
					Str("identifier", request.Identifier).
					Msg("Encountered error on GRPC Listen")

				return xerrors.Errorf("Failed to send to listener: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// PostAnalytics is used for consumers to provide information to Sandwich Daemon.
func (grpc *routeSandwichServer) PostAnalytics(ctx context.Context, request *pb.PostAnalyticsRequest) (response *pb.BaseResponse, err error) {
	onGRPCRequest()

	// TODO

	return nil, status.Errorf(codes.Unimplemented, "method PostAnalytics not implemented")
}

// FetchConsumerConfiguration returns the Consumer Configuration.
func (grpc *routeSandwichServer) FetchConsumerConfiguration(ctx context.Context, request *pb.FetchConsumerConfigurationRequest) (response *pb.FetchConsumerConfigurationResponse, err error) {
	onGRPCRequest()

	// ConsumerConfiguration at the moment just contains the Version of the library
	// along with a map of identifiers. The key is the application passed in metadata.

	identifiers := make(map[string]structs.ManagerConsumerConfiguration)

	grpc.sg.managersMu.RLock()
	for _, manager := range grpc.sg.Managers {
		manager.configurationMu.RLock()

		manager.userMu.RLock()
		user := manager.User
		manager.userMu.RUnlock()

		identifiers[manager.Identifier.Load()] = structs.ManagerConsumerConfiguration{
			Token: manager.Configuration.Token,
			ID:    manager.User.ID,
			User:  user,
		}
		manager.configurationMu.RUnlock()
	}
	grpc.sg.managersMu.RUnlock()

	sandwichConsumerConfiguration := structs.SandwichConsumerConfiguration{
		Version:     VERSION,
		Identifiers: identifiers,
	}

	var b bytes.Buffer

	err = jsoniter.NewEncoder(&b).Encode(sandwichConsumerConfiguration)
	if err != nil {
		grpc.sg.Logger.Warn().Err(err).Msg("Failed to marshal consumer configuration")
	}

	return &pb.FetchConsumerConfigurationResponse{
		File: b.Bytes(),
	}, nil
}

// FetchUsers returns users based on query or userID.
// If CreateDMChannel is set, will also create new DM channels (if not already).
func (grpc *routeSandwichServer) FetchUsers(ctx context.Context, request *pb.FetchUsersRequest) (response *pb.UsersResponse, err error) {
	onGRPCRequest()

	response = &pb.UsersResponse{
		Users: make(map[int64]*pb.User),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	hasQuery := request.Query != ""
	fetchDMChannels := request.CreateDMChannel && !hasQuery && request.Token != ""
	userIDs := make([]discord.Snowflake, 0)

	var client *Client
	if fetchDMChannels {
		client = NewClient(baseURL, request.Token)
	}

	if hasQuery {
		grpc.sg.State.usersMu.RLock()
		for _, user := range grpc.sg.State.Users {
			if requestMatch(request.Query, user.Username, user.Username+"#"+user.Discriminator, user.ID.String()) {
				userIDs = append(userIDs, user.ID)
			}
		}
		grpc.sg.State.usersMu.RUnlock()
	} else {
		for _, userID := range request.UserIDs {
			userIDs = append(userIDs, discord.Snowflake(userID))
		}
	}

	for _, userID := range userIDs {
		user, ok := grpc.sg.State.GetUser(userID)
		if ok {
			if fetchDMChannels && user.DMChannelID == nil {
				var resp discord.Channel

				var body io.ReadWriter

				err = jsoniter.NewEncoder(body).Encode(discord.CreateDMChannel{
					RecipientID: user.ID,
				})
				if err != nil {
					grpc.sg.Logger.Warn().Err(err).Int64("userID", int64(user.ID)).Msg("Failed to marshal create dm channel request")

					continue
				}

				_, err = client.FetchJSON(ctx, "GET", "/users/@me/channels", body, nil, &resp)
				if err != nil {
					grpc.sg.Logger.Warn().Err(err).Int64("userID", int64(user.ID)).Msg("Failed to create DM channel for user")

					continue
				}

				user.DMChannelID = &resp.ID

				grpc.sg.State.SetUser(&StateCtx{CacheUsers: true}, user)
			}

			grpcUser, err := pb.UserToGRPC(user)
			if err == nil {
				response.Users[int64(user.ID)] = grpcUser
			} else {
				grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.User to pb.User")
			}
		}

		onGRPCHit(ok)
	}

	response.BaseResponse.Ok = true

	return response, nil
}

// FetchGuildChannels returns guilds based on the guildID.
// Takes either query or channelIDs. Empty query and empty channelIDs will return all.
func (grpc *routeSandwichServer) FetchGuildChannels(ctx context.Context, request *pb.FetchGuildChannelsRequest) (response *pb.ChannelsResponse, err error) {
	onGRPCRequest()

	response = &pb.ChannelsResponse{
		GuildChannels: make(map[int64]*pb.Channel),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

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
				grpcChannel, err := pb.ChannelToGRPC(guildChannel)
				if err == nil {
					response.GuildChannels[int64(guildChannel.ID)] = grpcChannel
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Channel to pb.Channel")
				}
			}

			onGRPCHit(ok)
		}
	} else {
		guildChannels, ok := grpc.sg.State.GetAllGuildChannels(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildChannel := range guildChannels {
			if !hasQuery || requestMatch(request.Query, guildChannel.Name, guildChannel.ID.String()) {
				grpcChannel, err := pb.ChannelToGRPC(guildChannel)
				if err == nil {
					response.GuildChannels[int64(guildChannel.ID)] = grpcChannel
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Channel to pb.Channel")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return response, nil
}

// FetchGuildEmojis returns emojis based on the guildID.
// Takes either query or emojiIDs. Empty query and empty emojiIDs will return all.
func (grpc *routeSandwichServer) FetchGuildEmojis(ctx context.Context, request *pb.FetchGuildEmojisRequest) (response *pb.EmojisResponse, err error) {
	onGRPCRequest()

	response = &pb.EmojisResponse{
		GuildEmojis: make(map[int64]*pb.Emoji),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

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
				grpcEmoji, err := pb.EmojiToGRPC(guildEmoji)
				if err == nil {
					response.GuildEmojis[int64(guildEmoji.ID)] = grpcEmoji
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Emoji to pb.Emoji")
				}
			}

			onGRPCHit(ok)
		}
	} else {
		guildEmojis, ok := grpc.sg.State.GetAllGuildEmojis(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildEmoji := range guildEmojis {
			if !hasQuery || requestMatch(request.Query, guildEmoji.Name, guildEmoji.ID.String()) {
				grpcEmoji, err := pb.EmojiToGRPC(guildEmoji)
				if err == nil {
					response.GuildEmojis[int64(guildEmoji.ID)] = grpcEmoji
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Emoji to pb.Emoji")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return response, nil
}

// FetchGuildMembers returns guild members based on the guildID.
// Takes either query or userIDs. Empty query and empty userIDs will return all.
func (grpc *routeSandwichServer) FetchGuildMembers(ctx context.Context, request *pb.FetchGuildMembersRequest) (response *pb.GuildMembersResponse, err error) {
	onGRPCRequest()

	response = &pb.GuildMembersResponse{
		GuildMembers: make(map[int64]*pb.GuildMember),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

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
				grpcGuildMember, err := pb.GuildMemberToGRPC(guildMember)
				if err == nil {
					response.GuildMembers[int64(guildMember.User.ID)] = grpcGuildMember
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.GuildMember to pb.GuildMember")
				}
			}

			onGRPCHit(ok)
		}
	} else {
		guildGuildMembers, ok := grpc.sg.State.GetAllGuildMembers(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildMember := range guildGuildMembers {
			if !hasQuery || guildMemberMatch(request.Query, guildMember) {
				grpcGuildMember, err := pb.GuildMemberToGRPC(guildMember)
				if err == nil {
					response.GuildMembers[int64(guildMember.User.ID)] = grpcGuildMember
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.GuildMember to pb.GuildMember")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return response, nil
}

func guildMemberMatch(query string, guildMember *discord.GuildMember) (ok bool) {
	if guildMember.Nick != nil {
		return requestMatch(query, *guildMember.Nick, guildMember.User.Username,
			guildMember.User.Username+"#"+guildMember.User.Discriminator, guildMember.User.ID.String())
	}

	return requestMatch(query, guildMember.User.Username,
		guildMember.User.Username+"#"+guildMember.User.Discriminator, guildMember.User.ID.String())
}

// FetchGuild returns guilds based on the guildIDs.
func (grpc *routeSandwichServer) FetchGuild(ctx context.Context, request *pb.FetchGuildRequest) (response *pb.GuildsResponse, err error) {
	onGRPCRequest()

	response = &pb.GuildsResponse{
		Guilds: make(map[int64]*pb.Guild),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	hasGuildIds := len(request.GuildIDs) > 0
	hasQuery := request.Query != ""

	if hasGuildIds {
		for _, guildID := range request.GuildIDs {
			guild, ok := grpc.sg.State.GetGuild(discord.Snowflake(guildID))
			if ok {
				grpcGuild, err := pb.GuildToGRPC(guild)
				if err == nil {
					response.Guilds[int64(guild.ID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}

			onGRPCHit(ok)
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
			if requestMatch(request.Query, guild.Name, guild.ID.String()) {
				grpcGuild, err := pb.GuildToGRPC(grpc.sg.State.GuildFromState(guild))
				if err == nil {
					response.Guilds[int64(guild.ID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return response, nil
}

// FetchGuildRoles returns roles based on the roleIDs.
// Takes either query or roleIDs. Empty query and empty roleIDs will return all.
func (grpc *routeSandwichServer) FetchGuildRoles(ctx context.Context, request *pb.FetchGuildRolesRequest) (response *pb.GuildRolesResponse, err error) {
	onGRPCRequest()

	response = &pb.GuildRolesResponse{
		GuildRoles: make(map[int64]*pb.Role),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

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
				grpcRole, err := pb.RoleToGRPC(guildRole)
				if err == nil {
					response.GuildRoles[int64(guildRole.ID)] = grpcRole
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Role to pb.Role")
				}
			}

			onGRPCHit(ok)
		}
	} else {
		guildRoles, ok := grpc.sg.State.GetAllGuildRoles(discord.Snowflake(request.GuildID))
		if !ok {
			response.BaseResponse.Error = ErrCacheMiss.Error()

			return response, ErrCacheMiss
		}

		request.Query = norm.NFKD.String(request.Query)

		for _, guildRole := range guildRoles {
			if !hasQuery || requestMatch(request.Query, guildRole.Name, guildRole.ID.String()) {
				grpcRole, err := pb.RoleToGRPC(guildRole)
				if err == nil {
					response.GuildRoles[int64(guildRole.ID)] = grpcRole
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Role to pb.Role")
				}
			}
		}
	}

	response.BaseResponse.Ok = true

	return response, nil
}

// FetchMutualGuilds returns a list of all mutual guilds based on userID.
// Populates guildIDs with a list of snowflakes of all guilds.
// If expand is passed and True, will also populate guilds with the guild object.
func (grpc *routeSandwichServer) FetchMutualGuilds(ctx context.Context, request *pb.FetchMutualGuildsRequest) (response *pb.GuildsResponse, err error) {
	onGRPCRequest()

	response = &pb.GuildsResponse{
		GuildIDs: make([]int64, 0),
		Guilds:   make(map[int64]*pb.Guild),
		BaseResponse: &pb.BaseResponse{
			Ok: false,
		},
	}

	if request.UserID == 0 {
		response.BaseResponse.Error = ErrNoUserIDPresent.Error()

		return response, ErrNoUserIDPresent
	}

	guildIDs, _ := grpc.sg.State.GetUserMutualGuilds(discord.Snowflake(request.UserID))

	for _, guildID := range guildIDs {
		if request.Expand {
			guild, ok := grpc.sg.State.GetGuild(guildID)
			if ok {
				grpcGuild, err := pb.GuildToGRPC(guild)
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

	return response, nil
}

// RequestGuildChunk sends a guild chunk request.
// Returns once the guild has been chunked.
func (grpc *routeSandwichServer) RequestGuildChunk(ctx context.Context, request *pb.RequestGuildChunkRequest) (response *pb.BaseResponse, err error) {
	onGRPCRequest()

	response = &pb.BaseResponse{
		Ok:    false,
		Error: "Not implemented",
	}

	// TODO

	return response, status.Errorf(codes.Unimplemented, "method RequestGuildChunk not implemented")
}

// SendWebsocketMessage manually sends a websocket message.
func (grpc *routeSandwichServer) SendWebsocketMessage(ctx context.Context, request *pb.SendWebsocketMessageRequest) (response *pb.BaseResponse, err error) {
	onGRPCRequest()

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

	for _, data := range request.Data {
		err = shard.SendEvent(ctx, discord.GatewayOp(request.GatewayOPCode), data)
		if err != nil {
			response.Error = err.Error()

			return response, err
		}
	}

	response.Ok = true

	return response, nil
}

// WhereIsGuild returns a list of WhereIsGuildLocations based on guildId.
// WhereIsGuildLocations contains the manager, shardGroup and shardId.
func (grpc *routeSandwichServer) WhereIsGuild(ctx context.Context, request *pb.WhereIsGuildRequest) (response *pb.WhereIsGuildResponse, err error) {
	onGRPCRequest()

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
						ShardGroup: int64(shardGroup.ID),
						ShardId:    int64(shard.ShardGroup.ID),
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

	return response, nil
}
