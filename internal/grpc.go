package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/protobuf"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNoGuildIDPresent = errors.New("missing guild ID")
	ErrNoUserIDPresent  = errors.New("missing user ID")
	ErrNoQueryPresent   = errors.New("missing query")

	ErrDuplicateManagerPresent = errors.New("duplicate manager identifier passed")
	ErrNoManagerPresent        = errors.New("invalid manager identifier passed")
	ErrNoShardGroupPresent     = errors.New("invalid shard group identifier passed")
	ErrNoShardPresent          = errors.New("invalid shard ID passed")

	ErrCacheMiss = errors.New("item not present in cache")
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

func onGRPCHit(cacheHit bool) {
	if cacheHit {
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
func (grpc *routeSandwichServer) Listen(request *pb.ListenRequest, listener pb.Sandwich_ListenServer) error {
	onGRPCRequest()

	globalPoolID := grpc.sg.globalPoolAccumulator.Add(1)
	channel := make(chan []byte)

	grpc.sg.globalPool.Store(globalPoolID, channel)

	defer func() {
		grpc.sg.globalPool.Delete(globalPoolID)
		close(channel)
	}()

	ctx := listener.Context()

	for {
		select {
		case msg := <-channel:
			err := listener.Send(&pb.ListenResponse{
				Timestamp: time.Now().Unix(),
				Data:      msg,
			})
			if err != nil {
				grpc.sg.Logger.Error().Err(err).
					Str("identifier", request.Identifier).
					Msg("Encountered error on GRPC Listen")

				return fmt.Errorf("failed to send to listener: %w", err)
			}
		case <-ctx.Done():
			return nil
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

	identifiers := make(map[string]sandwich_structs.ManagerConsumerConfiguration)

	grpc.sg.Managers.Range(func(key string, manager *Manager) bool {
		manager.configurationMu.RLock()

		manager.userMu.RLock()
		user := manager.User
		manager.userMu.RUnlock()

		identifiers[manager.Identifier.Load()] = sandwich_structs.ManagerConsumerConfiguration{
			Token: manager.Configuration.Token,
			ID:    manager.User.ID,
			User:  user,
		}
		manager.configurationMu.RUnlock()
		return false
	})

	sandwichConsumerConfiguration := sandwich_structs.SandwichConsumerConfiguration{
		Version:     VERSION,
		Identifiers: identifiers,
	}

	var buf bytes.Buffer

	err = sandwichjson.MarshalToWriter(&buf, sandwichConsumerConfiguration)
	if err != nil {
		grpc.sg.Logger.Warn().Err(err).Msg("Failed to marshal consumer configuration")
	}

	return &pb.FetchConsumerConfigurationResponse{
		File: buf.Bytes(),
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

	// var client *Client
	// if fetchDMChannels {
	// 	client = NewClient(baseURL, request.Token)
	// }

	if hasQuery {
		grpc.sg.State.Users.Range(func(key discord.Snowflake, user *sandwich_structs.StateUser) bool {
			if requestMatch(request.Query, user.Username, user.Username+"#"+user.Discriminator, user.GlobalName, user.ID.String()) {
				userIDs = append(userIDs, user.ID)
			}
			return false
		})
	} else {
		for _, userID := range request.UserIDs {
			userIDs = append(userIDs, discord.Snowflake(userID))
		}
	}

	for _, userID := range userIDs {
		user, cacheHit := grpc.sg.State.GetUser(userID)
		if cacheHit {
			if fetchDMChannels && user.DMChannelID == nil {
				// var resp discord.Channel
				// var body io.ReadWriter
				// TODO: Refactor to use Discord session.
				// user.DMChannelID = &resp.ID

				grpc.sg.State.SetUser(&StateCtx{CacheUsers: true}, user)
			}

			grpcUser, err := pb.UserToGRPC(user)
			if err == nil {
				response.Users[int64(user.ID)] = grpcUser
			} else {
				grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.User to pb.User")
			}
		}

		onGRPCHit(cacheHit)
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
			guildChannel, cacheHit := grpc.sg.State.GetGuildChannel((*discord.Snowflake)(&request.GuildID), discord.Snowflake(channelID))
			if cacheHit {
				grpcChannel, err := pb.ChannelToGRPC(guildChannel)
				if err == nil {
					response.GuildChannels[int64(guildChannel.ID)] = grpcChannel
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Channel to pb.Channel")
				}
			}

			onGRPCHit(cacheHit)
		}
	} else {
		guildChannels, cacheHit := grpc.sg.State.GetAllGuildChannels(discord.Snowflake(request.GuildID))
		if !cacheHit {
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
			guildEmoji, cacheHit := grpc.sg.State.GetGuildEmoji(discord.Snowflake(request.GuildID), discord.Snowflake(emojiID))
			if cacheHit {
				grpcEmoji, err := pb.EmojiToGRPC(guildEmoji)
				if err == nil {
					response.GuildEmojis[int64(guildEmoji.ID)] = grpcEmoji
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Emoji to pb.Emoji")
				}
			}

			onGRPCHit(cacheHit)
		}
	} else {
		guildEmojis, cacheHit := grpc.sg.State.GetAllGuildEmojis(discord.Snowflake(request.GuildID))
		if !cacheHit {
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

	// Chunk the guild if requested.
	if request.ChunkGuild {
		ok, err := grpc.chunkGuild(discord.Snowflake(request.GuildID), request.AlwaysChunk)
		if err != nil {
			response.BaseResponse.Error = err.Error()

			return response, err
		}

		response.BaseResponse.Ok = ok

		if !ok {
			return response, nil
		}
	}

	if hasGuildMemberIds {
		for _, GuildMemberID := range request.UserIDs {
			guildMember, cacheHit := grpc.sg.State.GetGuildMember(discord.Snowflake(request.GuildID), discord.Snowflake(GuildMemberID))
			if cacheHit {
				grpcGuildMember, err := pb.GuildMemberToGRPC(guildMember)
				if err == nil {
					response.GuildMembers[int64(guildMember.User.ID)] = grpcGuildMember
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.GuildMember to pb.GuildMember")
				}
			}

			onGRPCHit(cacheHit)
		}
	} else {
		guildGuildMembers, cacheHit := grpc.sg.State.GetAllGuildMembers(discord.Snowflake(request.GuildID))
		if !cacheHit {
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
	if guildMember.Nick != "" {
		return requestMatch(query, guildMember.Nick, guildMember.User.Username,
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
			guild, cacheHit := grpc.sg.State.GetGuild(discord.Snowflake(guildID))
			if cacheHit {
				grpcGuild, err := pb.GuildToGRPC(guild)
				if err == nil {
					response.Guilds[int64(guild.ID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}

			onGRPCHit(cacheHit)
		}
	} else {
		if !hasQuery {
			response.BaseResponse.Error = ErrNoQueryPresent.Error()

			return response, ErrNoQueryPresent
		}

		request.Query = norm.NFKD.String(request.Query)

		grpc.sg.State.Guilds.Range(func(key discord.Snowflake, guild *discord.Guild) bool {
			if requestMatch(request.Query, guild.Name, guild.ID.String()) {
				grpcGuild, err := pb.GuildToGRPC(guild)
				if err == nil {
					response.Guilds[int64(guild.ID)] = grpcGuild
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Guild to pb.Guild")
				}
			}
			return false
		})
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
			guildRole, cacheHit := grpc.sg.State.GetGuildRole(discord.Snowflake(request.GuildID), discord.Snowflake(roleID))
			if cacheHit {
				grpcRole, err := pb.RoleToGRPC(guildRole)
				if err == nil {
					response.GuildRoles[int64(guildRole.ID)] = grpcRole
				} else {
					grpc.sg.Logger.Warn().Err(err).Msg("Failed to convert discord.Role to pb.Role")
				}
			}

			onGRPCHit(cacheHit)
		}
	} else {
		guildRoles, cacheHit := grpc.sg.State.GetAllGuildRoles(discord.Snowflake(request.GuildID))
		if !cacheHit {
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
			guild, cacheHit := grpc.sg.State.GetGuild(guildID)
			if cacheHit {
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
		Ok: false,
	}

	ok, err := grpc.chunkGuild(discord.Snowflake(request.GuildId), request.AlwaysChunk)

	response.Ok = ok

	if err != nil {
		response.Error = err.Error()
	}

	return response, err
}

func (grpc *routeSandwichServer) chunkGuild(guildID discord.Snowflake, alwaysChunk bool) (ok bool, err error) {
	grpc.sg.Managers.Range(func(key string, manager *Manager) bool {
		manager.ShardGroups.Range(func(key int32, shardGroup *ShardGroup) bool {
			shardGroup.Shards.Range(func(shardID int32, shard *Shard) bool {
				if _, ok := shard.Guilds.Load(guildID); ok {
					err = shard.ChunkGuild(guildID, alwaysChunk)
					if err != nil {
						return true
					} else {
						return true
					}
				}
				return false
			})
			return false
		})

		return false
	})

	ok = (err == nil)

	return
}

// SendWebsocketMessage manually sends a websocket message.
func (grpc *routeSandwichServer) SendWebsocketMessage(ctx context.Context, request *pb.SendWebsocketMessageRequest) (response *pb.BaseResponse, err error) {
	onGRPCRequest()

	response = &pb.BaseResponse{
		Ok: false,
	}

	manager, cacheHit := grpc.sg.Managers.Load(request.Manager)

	if !cacheHit {
		response.Error = ErrNoManagerPresent.Error()

		return response, ErrNoManagerPresent
	}

	shardGroup, ok := manager.ShardGroups.Load(request.ShardGroup)

	if !ok {
		response.Error = ErrNoShardGroupPresent.Error()

		return response, ErrNoShardGroupPresent
	}

	shard, ok := shardGroup.Shards.Load(request.Shard)

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

	grpc.sg.Managers.Range(func(key string, manager *Manager) bool {
		manager.ShardGroups.Range(func(key int32, shardGroup *ShardGroup) bool {
			shardGroup.Shards.Range(func(shardID int32, shard *Shard) bool {
				if _, ok := shard.Guilds.Load(discord.Snowflake(request.GuildID)); ok {
					var guildMember *pb.GuildMember

					guildMember_sandwich, ok := shard.Sandwich.State.GetGuildMember(discord.Snowflake(request.GuildID), shard.Manager.User.ID)
					if ok {
						guildMember, err = pb.GuildMemberToGRPC(guildMember_sandwich)
						if err != nil {
							grpc.sg.Logger.Error().Err(err).Int("guild_id", int(request.GuildID)).Int64("user_id", int64(shard.Manager.User.ID)).Msg("Failed to convert guild member to GRPC")
						}
					} else {
						grpc.sg.Logger.Warn().Int("guild_id", int(request.GuildID)).Int64("user_id", int64(shard.Manager.User.ID)).Msg("Failed to find own guild member")
					}

					response.Locations = append(response.Locations, &pb.WhereIsGuildLocation{
						Manager:     manager.Identifier.Load(),
						ShardGroup:  shardGroup.ID,
						ShardId:     shard.ShardGroup.ID,
						GuildMember: guildMember,
					})
				}
				return false
			})
			return false
		})
		return false
	})

	response.BaseResponse.Ok = true

	return response, nil
}

// RelayMessage creates a new event and sends it immediately back to consumers.
// All relayed messages will have the dispatch opcode and the sequence of 0.
func (grpc *routeSandwichServer) RelayMessage(ctx context.Context, request *pb.RelayMessageRequest) (response *pb.BaseResponse, err error) {
	onGRPCRequest()

	response = &pb.BaseResponse{
		Ok: false,
	}

	manager, cacheHit := grpc.sg.Managers.Load(request.Manager)

	if !cacheHit {
		response.Error = ErrNoManagerPresent.Error()

		return response, ErrNoManagerPresent
	}

	err = manager.PublishEvent(ctx, request.Type, request.Data)
	if err != nil {
		response.Error = err.Error()

		return response, err
	}

	response.Ok = true

	return response, nil
}
