package sandwich

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_protobuf "github.com/WelcomerTeam/Sandwich-Daemon/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ sandwich_protobuf.SandwichServer = &GRPCServer{}

type GRPCServer struct {
	sandwich_protobuf.UnimplementedSandwichServer

	sandwich *Sandwich
	logger   *slog.Logger
}

func (sandwich *Sandwich) NewGRPCServer() *GRPCServer {
	return &GRPCServer{
		UnimplementedSandwichServer: sandwich_protobuf.UnimplementedSandwichServer{},
		sandwich:                    sandwich,
		logger:                      sandwich.Logger.With("service", "grpc"),
	}
}

// Listen implements the Listen RPC method
func (grpcServer *GRPCServer) Listen(req *sandwich_protobuf.ListenRequest, stream sandwich_protobuf.Sandwich_ListenServer) error {
	RecordGRPCRequest()

	channel := make(chan *listenerData)

	counter := grpcServer.sandwich.addListener(channel)
	defer grpcServer.sandwich.removeListener(counter)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case data := <-channel:
			err := stream.Send(&sandwich_protobuf.ListenResponse{
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
func (grpcServer *GRPCServer) RelayMessage(ctx context.Context, req *sandwich_protobuf.RelayMessageRequest) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	application, ok := grpcServer.sandwich.Applications.Load(req.GetIdentifier())
	if !ok {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	var shard *Shard

	application.Shards.Range(func(_ int32, value *Shard) bool {
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
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &sandwich_protobuf.BaseResponse{
		Ok:    true,
		Error: "",
	}, nil
}

// ReloadConfiguration implements the ReloadConfiguration RPC method
func (grpcServer *GRPCServer) ReloadConfiguration(ctx context.Context, req *emptypb.Empty) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	err := grpcServer.sandwich.getConfig(ctx)
	if err != nil {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &sandwich_protobuf.BaseResponse{
		Ok:    true,
		Error: "",
	}, nil
}

// FetchApplication implements the FetchApplication RPC method
func (grpcServer *GRPCServer) FetchApplication(ctx context.Context, req *sandwich_protobuf.ApplicationIdentifier) (*sandwich_protobuf.FetchApplicationResponse, error) {
	RecordGRPCRequest()

	applications := make(map[string]*sandwich_protobuf.SandwichApplication)

	grpcServer.sandwich.Applications.Range(func(key string, application *Application) bool {
		if req.GetApplicationIdentifier() != "" && key != req.GetApplicationIdentifier() {
			return true
		}

		applications[key] = applicationToPB(application)
		return true
	})

	return &sandwich_protobuf.FetchApplicationResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok:    true,
			Error: "",
		},
		Applications: applications,
	}, nil
}

// StartApplication implements the StartApplication RPC method
func (grpcServer *GRPCServer) StartApplication(ctx context.Context, req *sandwich_protobuf.ApplicationIdentifierWithBlocking) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	application, ok := grpcServer.sandwich.Applications.Load(req.GetApplicationIdentifier())
	if !ok {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	// Override the context to a background context as if we are not blocking,
	// the context will be cancelled when the RPC call ends.
	ctx = context.Background()

	switch ApplicationStatus(application.Status.Load()) {
	case ApplicationStatusStarting, ApplicationStatusConnecting, ApplicationStatusConnected, ApplicationStatusReady:
		return &sandwich_protobuf.BaseResponse{
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
			return &sandwich_protobuf.BaseResponse{
				Ok:    false,
				Error: err.Error(),
			}, err
		}
	}

	return &sandwich_protobuf.BaseResponse{
		Ok: true,
	}, nil
}

// StopApplication implements the StopApplication RPC method
func (grpcServer *GRPCServer) StopApplication(ctx context.Context, req *sandwich_protobuf.ApplicationIdentifierWithBlocking) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	application, ok := grpcServer.sandwich.Applications.Load(req.GetApplicationIdentifier())
	if !ok {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	// Override the context to a background context as if we are not blocking,
	// the context will be cancelled when the RPC call ends.
	ctx = context.Background()

	switch ApplicationStatus(application.Status.Load()) {
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
				return &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: err.Error(),
				}, err
			}
		}
	case ApplicationStatusIdle, ApplicationStatusFailed, ApplicationStatusStopping, ApplicationStatusStopped:
		break
	}

	return &sandwich_protobuf.BaseResponse{
		Ok: true,
	}, nil
}

// CreateApplication implements the CreateApplication RPC method
func (grpcServer *GRPCServer) CreateApplication(ctx context.Context, req *sandwich_protobuf.CreateApplicationRequest) (*sandwich_protobuf.SandwichApplication, error) {
	RecordGRPCRequest()

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

	configuration := grpcServer.sandwich.Config.Load()

	// Remove existing application configuration.
	configuration.Applications = slices.DeleteFunc(configuration.Applications, func(application *ApplicationConfiguration) bool {
		return application.ApplicationIdentifier == applicationConfiguration.ApplicationIdentifier
	})

	configuration.Applications = append(configuration.Applications, applicationConfiguration)

	grpcServer.sandwich.Config.Store(configuration)

	// Override the context to a background context as if we autostart the application,
	// it will use the context that is passed to the RPC method which will be cancelled.
	ctx = context.Background()

	application, err := grpcServer.sandwich.addApplication(ctx, applicationConfiguration)
	if err != nil {
		return nil, err
	}

	if req.GetSaveConfig() {
		err := grpcServer.sandwich.configProvider.SaveConfig(ctx, configuration)
		if err != nil {
			return nil, err
		}
	}

	return applicationToPB(application), nil
}

// DeleteApplication implements the DeleteApplication RPC method
func (grpcServer *GRPCServer) DeleteApplication(ctx context.Context, req *sandwich_protobuf.ApplicationIdentifier) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	application, ok := grpcServer.sandwich.Applications.Load(req.GetApplicationIdentifier())
	if !ok {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	err := application.Stop(ctx)
	if err != nil {
		grpcServer.logger.Error("failed to stop application", "error", err, "identifier", req.GetApplicationIdentifier())
	}

	grpcServer.sandwich.Applications.Delete(req.GetApplicationIdentifier())

	return &sandwich_protobuf.BaseResponse{
		Ok: true,
	}, nil
}

// RequestGuildChunk implements the RequestGuildChunk RPC method
func (grpcServer *GRPCServer) RequestGuildChunk(ctx context.Context, req *sandwich_protobuf.RequestGuildChunkRequest) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	var shard *Shard

	grpcServer.sandwich.Applications.Range(func(key string, application *Application) bool {
		application.Shards.Range(func(_ int32, value *Shard) bool {
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

		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrGuildNotFound.Error(),
		}, ErrGuildNotFound
	}

	err := shard.chunkGuild(ctx, discord.Snowflake(req.GetGuildId()), req.GetAlwaysChunk())
	if err != nil {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &sandwich_protobuf.BaseResponse{
		Ok: true,
	}, nil
}

// SendWebsocketMessage implements the SendWebsocketMessage RPC method
func (grpcServer *GRPCServer) SendWebsocketMessage(ctx context.Context, req *sandwich_protobuf.SendWebsocketMessageRequest) (*sandwich_protobuf.BaseResponse, error) {
	RecordGRPCRequest()

	application, ok := grpcServer.sandwich.Applications.Load(req.GetIdentifier())
	if !ok {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrApplicationNotFound.Error(),
		}, ErrApplicationNotFound
	}

	shard, ok := application.Shards.Load(req.GetShard())
	if !ok {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: ErrShardNotFound.Error(),
		}, ErrShardNotFound
	}

	err := shard.send(ctx, discord.GatewayOp(req.GetGatewayOpCode()), req.GetData())
	if err != nil {
		return &sandwich_protobuf.BaseResponse{
			Ok:    false,
			Error: err.Error(),
		}, err
	}

	return &sandwich_protobuf.BaseResponse{
		Ok: true,
	}, nil
}

// WhereIsGuild implements the WhereIsGuild RPC method
func (grpcServer *GRPCServer) WhereIsGuild(ctx context.Context, req *sandwich_protobuf.WhereIsGuildRequest) (*sandwich_protobuf.WhereIsGuildResponse, error) {
	RecordGRPCRequest()

	locations := make(map[int64]*sandwich_protobuf.WhereIsGuildLocation)

	grpcServer.sandwich.Applications.Range(func(applicationIdentifier string, application *Application) bool {
		println("WhereIsGuild", "applicationIdentifier", applicationIdentifier, "guildId", req.GetGuildId())
		application.Shards.Range(func(_ int32, shard *Shard) bool {
			has := shard.guilds.Has(discord.Snowflake(req.GetGuildId()))

			println("WhereIsGuild", "applicationIdentifier", applicationIdentifier, "shardId", shard.ShardID, "hasGuild", has)

			if has {
				user := application.User.Load()

				var pbGuildMember *sandwich_protobuf.GuildMember

				guildMember, ok := grpcServer.sandwich.stateProvider.GetGuildMember(
					ctx,
					discord.Snowflake(req.GetGuildId()),
					user.ID,
				)
				if ok {
					pbGuildMember = sandwich_protobuf.GuildMemberToPB(guildMember)
				}

				locations[int64(user.ID)] = &sandwich_protobuf.WhereIsGuildLocation{
					Identifier:  applicationIdentifier,
					ShardId:     shard.ShardID,
					GuildMember: pbGuildMember,
				}
			}

			return true
		})

		return true
	})

	println("WhereIsGuild", "locations", len(locations), "guildId", req.GetGuildId())

	return &sandwich_protobuf.WhereIsGuildResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Locations: locations,
	}, nil
}

// FetchGuild implements the FetchGuild RPC method
func (grpcServer *GRPCServer) FetchGuild(ctx context.Context, req *sandwich_protobuf.FetchGuildRequest) (*sandwich_protobuf.FetchGuildResponse, error) {
	RecordGRPCRequest()

	guilds := make(map[int64]*sandwich_protobuf.Guild)

	guildIDs := req.GetGuildIds()

	if len(guildIDs) == 0 {
		stateGuilds, ok := grpcServer.sandwich.stateProvider.GetGuilds(ctx)
		if !ok {
			return &sandwich_protobuf.FetchGuildResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Guilds: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuild := range stateGuilds {
			guilds[int64(stateGuild.ID)] = sandwich_protobuf.GuildToPB(stateGuild)
		}
	} else {
		for _, guildID := range guildIDs {
			guild, ok := grpcServer.sandwich.stateProvider.GetGuild(ctx, discord.Snowflake(guildID))
			if !ok {
				continue
			}

			guilds[int64(guild.ID)] = sandwich_protobuf.GuildToPB(guild)
		}
	}

	return &sandwich_protobuf.FetchGuildResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Guilds: guilds,
	}, nil
}

// FetchGuildMember implements the FetchGuildMember RPC method
func (grpcServer *GRPCServer) FetchGuildMember(ctx context.Context, req *sandwich_protobuf.FetchGuildMemberRequest) (*sandwich_protobuf.FetchGuildMemberResponse, error) {
	RecordGRPCRequest()

	guildMembers := make(map[int64]*sandwich_protobuf.GuildMember)

	userIDs := req.GetUserIds()

	if len(userIDs) == 0 {
		stateGuildMembers, ok := grpcServer.sandwich.stateProvider.GetGuildMembers(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &sandwich_protobuf.FetchGuildMemberResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				GuildMembers: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildMember := range stateGuildMembers {
			guildMembers[int64(stateGuildMember.User.ID)] = sandwich_protobuf.GuildMemberToPB(stateGuildMember)
		}
	} else {
		for _, userID := range req.GetUserIds() {
			stateGuildMember, ok := grpcServer.sandwich.stateProvider.GetGuildMember(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(userID))
			if !ok {
				continue
			}

			guildMembers[int64(stateGuildMember.User.ID)] = sandwich_protobuf.GuildMemberToPB(stateGuildMember)
		}
	}

	return &sandwich_protobuf.FetchGuildMemberResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		GuildMembers: guildMembers,
	}, nil
}

// FetchGuildChannel implements the FetchGuildChannel RPC method
func (grpcServer *GRPCServer) FetchGuildChannel(ctx context.Context, req *sandwich_protobuf.FetchGuildChannelRequest) (*sandwich_protobuf.FetchGuildChannelResponse, error) {
	RecordGRPCRequest()

	guildChannels := make(map[int64]*sandwich_protobuf.Channel)

	channelIDs := req.GetChannelIds()

	if len(channelIDs) == 0 {
		stateGuildChannels, ok := grpcServer.sandwich.stateProvider.GetGuildChannels(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &sandwich_protobuf.FetchGuildChannelResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Channels: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildChannel := range stateGuildChannels {
			guildChannels[int64(stateGuildChannel.ID)] = sandwich_protobuf.ChannelToPB(stateGuildChannel)
		}
	} else {
		for _, channelID := range channelIDs {
			stateGuildChannel, ok := grpcServer.sandwich.stateProvider.GetGuildChannel(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(channelID))
			if !ok {
				continue
			}

			guildChannels[int64(stateGuildChannel.ID)] = sandwich_protobuf.ChannelToPB(stateGuildChannel)
		}
	}

	return &sandwich_protobuf.FetchGuildChannelResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Channels: guildChannels,
	}, nil
}

// FetchGuildRole implements the FetchGuildRole RPC method
func (grpcServer *GRPCServer) FetchGuildRole(ctx context.Context, req *sandwich_protobuf.FetchGuildRoleRequest) (*sandwich_protobuf.FetchGuildRoleResponse, error) {
	RecordGRPCRequest()

	guildRoles := make(map[int64]*sandwich_protobuf.Role)

	roleIDs := req.GetRoleIds()

	if len(roleIDs) == 0 {
		stateGuildRoles, ok := grpcServer.sandwich.stateProvider.GetGuildRoles(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &sandwich_protobuf.FetchGuildRoleResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Roles: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildRole := range stateGuildRoles {
			guildRoles[int64(stateGuildRole.ID)] = sandwich_protobuf.RoleToPB(stateGuildRole)
		}
	} else {
		for _, roleID := range roleIDs {
			stateGuildRole, ok := grpcServer.sandwich.stateProvider.GetGuildRole(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(roleID))
			if !ok {
				continue
			}

			guildRoles[int64(stateGuildRole.ID)] = sandwich_protobuf.RoleToPB(stateGuildRole)
		}
	}

	return &sandwich_protobuf.FetchGuildRoleResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Roles: guildRoles,
	}, nil
}

// FetchGuildEmoji implements the FetchGuildEmoji RPC method
func (grpcServer *GRPCServer) FetchGuildEmoji(ctx context.Context, req *sandwich_protobuf.FetchGuildEmojiRequest) (*sandwich_protobuf.FetchGuildEmojiResponse, error) {
	RecordGRPCRequest()

	guildEmojis := make(map[int64]*sandwich_protobuf.Emoji)

	emojiIDs := req.GetEmojiIds()

	if len(emojiIDs) == 0 {
		stateGuildEmojis, ok := grpcServer.sandwich.stateProvider.GetGuildEmojis(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &sandwich_protobuf.FetchGuildEmojiResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Emojis: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildEmoji := range stateGuildEmojis {
			guildEmojis[int64(stateGuildEmoji.ID)] = sandwich_protobuf.EmojiToPB(stateGuildEmoji)
		}
	} else {
		for _, emojiID := range emojiIDs {
			stateGuildEmoji, ok := grpcServer.sandwich.stateProvider.GetGuildEmoji(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(emojiID))
			if !ok {
				continue
			}

			guildEmojis[int64(stateGuildEmoji.ID)] = sandwich_protobuf.EmojiToPB(stateGuildEmoji)
		}
	}

	return &sandwich_protobuf.FetchGuildEmojiResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Emojis: guildEmojis,
	}, nil
}

// FetchGuildSticker implements the FetchGuildSticker RPC method
func (grpcServer *GRPCServer) FetchGuildSticker(ctx context.Context, req *sandwich_protobuf.FetchGuildStickerRequest) (*sandwich_protobuf.FetchGuildStickerResponse, error) {
	RecordGRPCRequest()

	guildStickers := make(map[int64]*sandwich_protobuf.Sticker)

	stickerIDs := req.GetStickerIds()

	if len(stickerIDs) == 0 {
		stateGuildStickers, ok := grpcServer.sandwich.stateProvider.GetGuildStickers(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &sandwich_protobuf.FetchGuildStickerResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				Stickers: nil,
			}, ErrGuildNotFound
		}

		for _, stateGuildSticker := range stateGuildStickers {
			guildStickers[int64(stateGuildSticker.ID)] = sandwich_protobuf.StickerToPB(stateGuildSticker)
		}
	} else {
		for _, stickerID := range stickerIDs {
			stateGuildSticker, ok := grpcServer.sandwich.stateProvider.GetGuildSticker(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(stickerID))
			if !ok {
				continue
			}

			guildStickers[int64(stateGuildSticker.ID)] = sandwich_protobuf.StickerToPB(stateGuildSticker)
		}
	}

	return &sandwich_protobuf.FetchGuildStickerResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Stickers: guildStickers,
	}, nil
}

// FetchGuildVoiceState implements the FetchGuildVoiceState RPC method
func (grpcServer *GRPCServer) FetchGuildVoiceState(ctx context.Context, req *sandwich_protobuf.FetchGuildVoiceStateRequest) (*sandwich_protobuf.FetchGuildVoiceStateResponse, error) {
	RecordGRPCRequest()

	voiceStates := make(map[int64]*sandwich_protobuf.VoiceState)

	userIDs := req.GetUserIds()

	if len(userIDs) == 0 {
		stateVoiceStates, ok := grpcServer.sandwich.stateProvider.GetVoiceStates(ctx, discord.Snowflake(req.GetGuildId()))
		if !ok {
			return &sandwich_protobuf.FetchGuildVoiceStateResponse{
				BaseResponse: &sandwich_protobuf.BaseResponse{
					Ok:    false,
					Error: ErrGuildNotFound.Error(),
				},
				VoiceStates: nil,
			}, ErrGuildNotFound
		}

		for _, stateVoiceState := range stateVoiceStates {
			voiceStates[int64(stateVoiceState.UserID)] = sandwich_protobuf.VoiceStateToPB(stateVoiceState)
		}
	} else {
		for _, userID := range userIDs {
			stateVoiceState, ok := grpcServer.sandwich.stateProvider.GetVoiceState(ctx, discord.Snowflake(req.GetGuildId()), discord.Snowflake(userID))
			if !ok {
				continue
			}

			voiceStates[int64(stateVoiceState.UserID)] = sandwich_protobuf.VoiceStateToPB(stateVoiceState)
		}
	}

	return &sandwich_protobuf.FetchGuildVoiceStateResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		VoiceStates: voiceStates,
	}, nil
}

// FetchUser implements the FetchUser RPC method
func (grpcServer *GRPCServer) FetchUser(ctx context.Context, req *sandwich_protobuf.FetchUserRequest) (*sandwich_protobuf.FetchUserResponse, error) {
	RecordGRPCRequest()

	users := make(map[int64]*sandwich_protobuf.User)

	userIDs := req.GetUserIds()

	for _, userID := range userIDs {
		stateUser, ok := grpcServer.sandwich.stateProvider.GetUser(ctx, discord.Snowflake(userID))
		if ok {
			users[int64(stateUser.ID)] = sandwich_protobuf.UserToPB(stateUser)
		}
	}

	return &sandwich_protobuf.FetchUserResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Users: users,
	}, nil
}

// FetchUserMutualGuilds implements the FetchUserMutualGuilds RPC method
func (grpcServer *GRPCServer) FetchUserMutualGuilds(ctx context.Context, req *sandwich_protobuf.FetchUserMutualGuildsRequest) (*sandwich_protobuf.FetchUserMutualGuildsResponse, error) {
	RecordGRPCRequest()

	mutualGuilds := make(map[int64]*sandwich_protobuf.Guild)

	mutualGuildsState, ok := grpcServer.sandwich.stateProvider.GetUserMutualGuilds(ctx, discord.Snowflake(req.GetUserId()))
	if !ok {
		return &sandwich_protobuf.FetchUserMutualGuildsResponse{
			BaseResponse: &sandwich_protobuf.BaseResponse{
				Ok:    false,
				Error: ErrUserNotFound.Error(),
			},
		}, ErrUserNotFound
	}

	for _, mutualGuild := range mutualGuildsState {
		guildState, ok := grpcServer.sandwich.stateProvider.GetGuild(ctx, discord.Snowflake(mutualGuild))
		if ok {
			mutualGuilds[int64(guildState.ID)] = sandwich_protobuf.GuildToPB(guildState)
		} else {
			mutualGuilds[int64(mutualGuild)] = &sandwich_protobuf.Guild{
				ID: int64(mutualGuild),
			}
		}
	}

	return &sandwich_protobuf.FetchUserMutualGuildsResponse{
		BaseResponse: &sandwich_protobuf.BaseResponse{
			Ok: true,
		},
		Guilds: mutualGuilds,
	}, nil
}
