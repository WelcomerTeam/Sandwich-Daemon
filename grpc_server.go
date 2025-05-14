package sandwich

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/WelcomerTeam/Discord/discord"
	pb "github.com/WelcomerTeam/Sandwich-Daemon/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ pb.SandwichServer = &GRPCServer{}

type GRPCServer struct {
	pb.UnimplementedSandwichServer

	sandwich *Sandwich
	logger   *slog.Logger
}

func (sandwich *Sandwich) NewGRPCServer() *GRPCServer {
	return &GRPCServer{
		UnimplementedSandwichServer: pb.UnimplementedSandwichServer{},
		sandwich:                    sandwich,
		logger:                      sandwich.logger.With("service", "grpc"),
	}
}

// Listen implements the Listen RPC method
func (grpcServer *GRPCServer) Listen(req *pb.ListenRequest, stream pb.Sandwich_ListenServer) error {
	channel := make(chan *listenerData)

	counter := grpcServer.sandwich.addListener(channel)
	defer grpcServer.sandwich.removeListener(counter)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case data := <-channel:
			err := stream.Send(&pb.ListenResponse{
				Timestamp: data.timestamp.Unix(),
				Data:      data.payload,
			})
			if err != nil {
				grpcServer.logger.Error("failed to send listen response", "error", err, "identifier", req.GetIdentifier())

				return fmt.Errorf("failed to send listen response: %w", err)
			}
		}
	}
}

// RelayMessage implements the RelayMessage RPC method
func (grpcServer *GRPCServer) RelayMessage(ctx context.Context, req *pb.RelayMessageRequest) (*pb.BaseResponse, error) {
	application, ok := grpcServer.sandwich.applications.Load(req.GetIdentifier())
	if !ok {
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	var shard *Shard

	application.shards.Range(func(_ int32, value *Shard) bool {
		shard = value

		return false
	})

	err := grpcServer.sandwich.eventProvider.Dispatch(ctx, shard, &discord.GatewayPayload{
		Type:     req.GetType(),
		Data:     req.GetData(),
		Sequence: 0,
		Op:       discord.GatewayOpDispatch,
	}, nil)
	if err != nil {
		return &pb.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &pb.BaseResponse{
		Ok:    true,
		Error: "",
	}, nil
}

// ReloadConfiguration implements the ReloadConfiguration RPC method
func (grpcServer *GRPCServer) ReloadConfiguration(ctx context.Context, req *emptypb.Empty) (*pb.BaseResponse, error) {
	err := grpcServer.sandwich.getConfig(ctx)
	if err != nil {
		return &pb.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &pb.BaseResponse{
		Ok:    true,
		Error: "",
	}, nil
}

// FetchApplication implements the FetchApplication RPC method
func (grpcServer *GRPCServer) FetchApplication(ctx context.Context, req *pb.ApplicationIdentifier) (*pb.FetchApplicationResponse, error) {
	applications := make(map[string]*pb.SandwichApplication)

	grpcServer.sandwich.applications.Range(func(key string, application *Application) bool {
		if req.GetApplicationIdentifier() != "" && key != req.GetApplicationIdentifier() {
			return true
		}

		applications[key] = applicationToPB(application)
		return true
	})

	return &pb.FetchApplicationResponse{
		BaseResponse: &pb.BaseResponse{
			Ok:    true,
			Error: "",
		},
		Applications: applications,
	}, nil
}

// StartApplication implements the StartApplication RPC method
func (grpcServer *GRPCServer) StartApplication(ctx context.Context, req *pb.ApplicationIdentifierWithBlocking) (*pb.BaseResponse, error) {
	application, ok := grpcServer.sandwich.applications.Load(req.GetApplicationIdentifier())
	if !ok {
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	switch ApplicationStatus(application.status.Load()) {
	case ApplicationStatusStarting, ApplicationStatusConnecting, ApplicationStatusConnected, ApplicationStatusReady:
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrApplicationAlreadyRunning.Error(),
		}, ErrApplicationAlreadyRunning
	case ApplicationStatusIdle, ApplicationStatusFailed, ApplicationStatusStopping, ApplicationStatusStopped:
		break
	}

	wait := make(chan error)

	go func() {
		err := application.Start(ctx)
		if err != nil {
			wait <- err
		}

		close(wait)
	}()

	if req.GetBlocking() {
		err := <-wait
		if err != nil {
			return &pb.BaseResponse{
				Ok:    false,
				Error: err.Error(),
			}, err
		}
	}

	return &pb.BaseResponse{
		Ok: true,
	}, nil
}

// StopApplication implements the StopApplication RPC method
func (grpcServer *GRPCServer) StopApplication(ctx context.Context, req *pb.ApplicationIdentifierWithBlocking) (*pb.BaseResponse, error) {
	application, ok := grpcServer.sandwich.applications.Load(req.GetApplicationIdentifier())
	if !ok {
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	switch ApplicationStatus(application.status.Load()) {
	case ApplicationStatusStarting, ApplicationStatusConnecting, ApplicationStatusConnected, ApplicationStatusReady:
		wait := make(chan error)

		go func() {
			err := application.Stop(ctx)
			if err != nil {
				wait <- err
			}

			close(wait)
		}()

		if req.GetBlocking() {
			err := <-wait
			if err != nil {
				return &pb.BaseResponse{
					Ok:    false,
					Error: err.Error(),
				}, err
			}
		}
	case ApplicationStatusIdle, ApplicationStatusFailed, ApplicationStatusStopping, ApplicationStatusStopped:
		break
	}

	return &pb.BaseResponse{
		Ok: true,
	}, nil
}

// CreateApplication implements the CreateApplication RPC method
func (grpcServer *GRPCServer) CreateApplication(ctx context.Context, req *pb.CreateApplicationRequest) (*pb.SandwichApplication, error) {
	var defaultPresence discord.UpdateStatus

	if req.GetDefaultPresence() != nil {
		err := json.Unmarshal(req.GetDefaultPresence(), &defaultPresence)
		if err != nil {
			return nil, err
		}
	}

	var values map[string]any

	if req.GetValues() != nil {
		err := json.Unmarshal(req.GetValues(), &values)
		if err != nil {
			return nil, err
		}
	}

	applicationConfiguration := &ApplicationConfiguration{
		ProducerIdentifier:    req.GetProducerIdentifier(),
		DisplayName:           req.GetDisplayName(),
		BotToken:              req.GetBotToken(),
		ShardCount:            req.GetShardCount(),
		AutoSharded:           req.GetAutoSharded(),
		ApplicationIdentifier: req.GetApplicationIdentifier(),
		ClientName:            req.GetClientName(),
		IncludeRandomSuffix:   req.GetIncludeRandomSuffix(),
		AutoStart:             req.GetAutoStart(),
		DefaultPresence:       defaultPresence,
		Intents:               req.GetIntents(),
		ChunkGuildsOnStart:    req.GetChunkGuildsOnStart(),
		EventBlacklist:        req.GetEventBlacklist(),
		ProduceBlacklist:      req.GetProduceBlacklist(),
		ShardIDs:              req.GetShardIds(),
		Values:                values,
	}

	err := grpcServer.sandwich.validateApplicationConfig(applicationConfiguration)
	if err != nil {
		return nil, err
	}

	application, err := grpcServer.sandwich.addApplication(ctx, applicationConfiguration)
	if err != nil {
		return nil, err
	}

	return applicationToPB(application), nil
}

// DeleteApplication implements the DeleteApplication RPC method
func (grpcServer *GRPCServer) DeleteApplication(ctx context.Context, req *pb.ApplicationIdentifier) (*pb.BaseResponse, error) {
	application, ok := grpcServer.sandwich.applications.Load(req.GetApplicationIdentifier())
	if !ok {
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	err := application.Stop(ctx)
	if err != nil {
		grpcServer.logger.Error("failed to stop application", "error", err, "identifier", req.GetApplicationIdentifier())
	}

	grpcServer.sandwich.applications.Delete(req.GetApplicationIdentifier())

	return &pb.BaseResponse{
		Ok: true,
	}, nil
}

// RequestGuildChunk implements the RequestGuildChunk RPC method
func (grpcServer *GRPCServer) RequestGuildChunk(ctx context.Context, req *pb.RequestGuildChunkRequest) (*pb.BaseResponse, error) {
	var shard *Shard

	grpcServer.sandwich.applications.Range(func(key string, application *Application) bool {
		application.shards.Range(func(_ int32, value *Shard) bool {
			if value.guilds.Has(discord.Snowflake(req.GetGuildId())) {
				shard = value

				return false
			}

			return true
		})

		return shard != nil
	})

	if shard == nil {
		grpcServer.logger.Error("failed to find shard for guild", "guild", req.GetGuildId())

		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrGuildNotFound.Error(),
		}, ErrGuildNotFound
	}

	err := shard.chunkGuild(ctx, discord.Snowflake(req.GetGuildId()), req.GetAlwaysChunk())
	if err != nil {
		return &pb.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &pb.BaseResponse{
		Ok: true,
	}, nil
}

// SendWebsocketMessage implements the SendWebsocketMessage RPC method
func (grpcServer *GRPCServer) SendWebsocketMessage(ctx context.Context, req *pb.SendWebsocketMessageRequest) (*pb.BaseResponse, error) {
	application, ok := grpcServer.sandwich.applications.Load(req.GetIdentifier())
	if !ok {
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	shard, ok := application.shards.Load(req.GetShard())
	if !ok {
		return &pb.BaseResponse{
			Ok:    false,
			Error: ErrShardNotFound.Error(),
		}, ErrShardNotFound
	}

	err := shard.send(ctx, discord.GatewayOp(req.GetGatewayOpCode()), req.GetData())
	if err != nil {
		return &pb.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &pb.BaseResponse{
		Ok: true,
	}, nil
}

// WhereIsGuild implements the WhereIsGuild RPC method
func (grpcServer *GRPCServer) WhereIsGuild(ctx context.Context, req *pb.WhereIsGuildRequest) (*pb.WhereIsGuildResponse, error) {
	locations := make(map[int64]*pb.WhereIsGuildLocation)

	grpcServer.sandwich.applications.Range(func(applicationIdentifier string, application *Application) bool {
		hasGuild := false

		application.shards.Range(func(_ int32, shard *Shard) bool {
			if shard.guilds.Has(discord.Snowflake(req.GetGuildId())) {
				hasGuild = true

				user := application.user.Load()

				var pbGuildMember *pb.GuildMember

				guildMember, ok := grpcServer.sandwich.stateProvider.GetGuildMember(
					ctx,
					discord.Snowflake(req.GetGuildId()),
					user.ID,
				)
				if ok {
					pbGuildMember = guildMemberToPB(guildMember)
				}

				locations[int64(user.ID)] = &pb.WhereIsGuildLocation{
					Identifier:  applicationIdentifier,
					ShardId:     shard.shardID,
					GuildMember: pbGuildMember,
				}

				return false
			}

			return true
		})

		return hasGuild
	})

	return &pb.WhereIsGuildResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Locations: locations,
	}, nil
}

// FetchGuild implements the FetchGuild RPC method
func (grpcServer *GRPCServer) FetchGuild(ctx context.Context, req *pb.FetchGuildRequest) (*pb.FetchGuildResponse, error) {
	guilds := make(map[int64]*pb.Guild)

	guildIDs := req.GetGuildIds()

	if len(guildIDs) == 0 {
		stateGuilds, ok := grpcServer.sandwich.stateProvider.GetGuilds(ctx)
		if !ok {
			return &pb.FetchGuildResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Guilds: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuild := range stateGuilds {
			guilds[int64(stateGuild.ID)] = guildToPB(stateGuild)
		}
	} else {
		for _, guildID := range guildIDs {
			guild, ok := grpcServer.sandwich.stateProvider.GetGuild(ctx, discord.Snowflake(guildID))
			if !ok {
				continue
			}

			guilds[int64(guild.ID)] = guildToPB(guild)
		}
	}

	return &pb.FetchGuildResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Guilds: guilds,
	}, nil
}

// FetchGuildMember implements the FetchGuildMember RPC method
func (grpcServer *GRPCServer) FetchGuildMember(ctx context.Context, req *pb.FetchGuildMemberRequest) (*pb.FetchGuildMemberResponse, error) {
	guildMembers := make(map[int64]*pb.GuildMember)

	userIDs := req.GetUserIds()

	if len(userIDs) == 0 {
		stateGuildMembers, ok := grpcServer.sandwich.stateProvider.GetGuildMembers(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &pb.FetchGuildMemberResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				GuildMembers: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildMember := range stateGuildMembers {
			guildMembers[int64(stateGuildMember.User.ID)] = guildMemberToPB(stateGuildMember)
		}
	} else {
		for _, userID := range req.GetUserIds() {
			stateGuildMember, ok := grpcServer.sandwich.stateProvider.GetGuildMember(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(userID))
			if !ok {
				continue
			}

			guildMembers[int64(stateGuildMember.User.ID)] = guildMemberToPB(stateGuildMember)
		}
	}

	return &pb.FetchGuildMemberResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		GuildMembers: guildMembers,
	}, nil
}

// FetchGuildChannel implements the FetchGuildChannel RPC method
func (grpcServer *GRPCServer) FetchGuildChannel(ctx context.Context, req *pb.FetchGuildChannelRequest) (*pb.FetchGuildChannelResponse, error) {
	guildChannels := make(map[int64]*pb.Channel)

	channelIDs := req.GetChannelIds()

	if len(channelIDs) == 0 {
		stateGuildChannels, ok := grpcServer.sandwich.stateProvider.GetGuildChannels(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &pb.FetchGuildChannelResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Channels: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildChannel := range stateGuildChannels {
			guildChannels[int64(stateGuildChannel.ID)] = channelToPB(stateGuildChannel)
		}
	} else {
		for _, channelID := range channelIDs {
			stateGuildChannel, ok := grpcServer.sandwich.stateProvider.GetGuildChannel(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(channelID))
			if !ok {
				continue
			}

			guildChannels[int64(stateGuildChannel.ID)] = channelToPB(stateGuildChannel)
		}
	}

	return &pb.FetchGuildChannelResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Channels: guildChannels,
	}, nil
}

// FetchGuildRole implements the FetchGuildRole RPC method
func (grpcServer *GRPCServer) FetchGuildRole(ctx context.Context, req *pb.FetchGuildRoleRequest) (*pb.FetchGuildRoleResponse, error) {
	guildRoles := make(map[int64]*pb.Role)

	roleIDs := req.GetRoleIds()

	if len(roleIDs) == 0 {
		stateGuildRoles, ok := grpcServer.sandwich.stateProvider.GetGuildRoles(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &pb.FetchGuildRoleResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Roles: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildRole := range stateGuildRoles {
			guildRoles[int64(stateGuildRole.ID)] = roleToPB(stateGuildRole)
		}
	} else {
		for _, roleID := range roleIDs {
			stateGuildRole, ok := grpcServer.sandwich.stateProvider.GetGuildRole(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(roleID))
			if !ok {
				continue
			}

			guildRoles[int64(stateGuildRole.ID)] = roleToPB(stateGuildRole)
		}
	}

	return &pb.FetchGuildRoleResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Roles: guildRoles,
	}, nil
}

// FetchGuildEmoji implements the FetchGuildEmoji RPC method
func (grpcServer *GRPCServer) FetchGuildEmoji(ctx context.Context, req *pb.FetchGuildEmojiRequest) (*pb.FetchGuildEmojiResponse, error) {
	guildEmojis := make(map[int64]*pb.Emoji)

	emojiIDs := req.GetEmojiIds()

	if len(emojiIDs) == 0 {
		stateGuildEmojis, ok := grpcServer.sandwich.stateProvider.GetGuildEmojis(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &pb.FetchGuildEmojiResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Emojis: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildEmoji := range stateGuildEmojis {
			guildEmojis[int64(stateGuildEmoji.ID)] = emojiToPB(stateGuildEmoji)
		}
	} else {
		for _, emojiID := range emojiIDs {
			stateGuildEmoji, ok := grpcServer.sandwich.stateProvider.GetGuildEmoji(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(emojiID))
			if !ok {
				continue
			}

			guildEmojis[int64(stateGuildEmoji.ID)] = emojiToPB(stateGuildEmoji)
		}
	}

	return &pb.FetchGuildEmojiResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Emojis: guildEmojis,
	}, nil
}

// FetchGuildSticker implements the FetchGuildSticker RPC method
func (grpcServer *GRPCServer) FetchGuildSticker(ctx context.Context, req *pb.FetchGuildStickerRequest) (*pb.FetchGuildStickerResponse, error) {
	guildStickers := make(map[int64]*pb.Sticker)

	stickerIDs := req.GetStickerIds()

	if len(stickerIDs) == 0 {
		stateGuildStickers, ok := grpcServer.sandwich.stateProvider.GetGuildStickers(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &pb.FetchGuildStickerResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Stickers: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildSticker := range stateGuildStickers {
			guildStickers[int64(stateGuildSticker.ID)] = stickerToPB(stateGuildSticker)
		}
	} else {
		for _, stickerID := range stickerIDs {
			stateGuildSticker, ok := grpcServer.sandwich.stateProvider.GetGuildSticker(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(stickerID))
			if !ok {
				continue
			}

			guildStickers[int64(stateGuildSticker.ID)] = stickerToPB(stateGuildSticker)
		}
	}

	return &pb.FetchGuildStickerResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Stickers: guildStickers,
	}, nil
}

// FetchGuildVoiceState implements the FetchGuildVoiceState RPC method
func (grpcServer *GRPCServer) FetchGuildVoiceState(ctx context.Context, req *pb.FetchGuildVoiceStateRequest) (*pb.FetchGuildVoiceStateResponse, error) {
	voiceStates := make(map[int64]*pb.VoiceState)

	userIDs := req.GetUserIds()

	if len(userIDs) == 0 {
		stateVoiceStates, ok := grpcServer.sandwich.stateProvider.GetVoiceStates(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &pb.FetchGuildVoiceStateResponse{
				BaseResponse: &pb.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				VoiceStates: nil,
			}, ErrGuildNotFound
		}

		for _, stateVoiceState := range stateVoiceStates {
			voiceStates[int64(stateVoiceState.UserID)] = voiceStateToPB(stateVoiceState)
		}
	} else {
		for _, userID := range userIDs {
			stateVoiceState, ok := grpcServer.sandwich.stateProvider.GetVoiceState(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(userID))
			if !ok {
				continue
			}

			voiceStates[int64(stateVoiceState.UserID)] = voiceStateToPB(stateVoiceState)
		}
	}

	return &pb.FetchGuildVoiceStateResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		VoiceStates: voiceStates,
	}, nil
}

// FetchUser implements the FetchUser RPC method
func (grpcServer *GRPCServer) FetchUser(ctx context.Context, req *pb.FetchUserRequest) (*pb.FetchUserResponse, error) {
	users := make(map[int64]*pb.User)

	userIDs := req.GetUserIds()

	for _, userID := range userIDs {
		stateUser, ok := grpcServer.sandwich.stateProvider.GetUser(ctx, discord.Snowflake(userID))
		if ok {
			users[int64(stateUser.ID)] = userToPB(stateUser)
		}
	}

	return &pb.FetchUserResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Users: users,
	}, nil
}

// FetchUserMutualGuilds implements the FetchUserMutualGuilds RPC method
func (grpcServer *GRPCServer) FetchUserMutualGuilds(ctx context.Context, req *pb.FetchUserMutualGuildsRequest) (*pb.FetchUserMutualGuildsResponse, error) {
	mutualGuilds := make(map[int64]*pb.Guild)

	mutualGuildsState, ok := grpcServer.sandwich.stateProvider.GetUserMutualGuilds(ctx, discord.Snowflake(req.GetUserId()))
	if !ok {
		return &pb.FetchUserMutualGuildsResponse{
			BaseResponse: &pb.BaseResponse{
				Ok:    false,
				Error: ErrUserNotFound.Error(),
			},
		}, ErrUserNotFound
	}

	for _, mutualGuild := range mutualGuildsState {
		guildState, ok := grpcServer.sandwich.stateProvider.GetGuild(ctx, discord.Snowflake(mutualGuild))
		if ok {
			mutualGuilds[int64(guildState.ID)] = guildToPB(guildState)
		} else {
			mutualGuilds[int64(mutualGuild)] = &pb.Guild{
				ID: int64(mutualGuild),
			}
		}
	}

	return &pb.FetchUserMutualGuildsResponse{
		BaseResponse: &pb.BaseResponse{
			Ok: true,
		},
		Guilds: mutualGuilds,
	}, nil
}
