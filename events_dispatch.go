package sandwich

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
)

const (
	ReadyTimeout = 1 * time.Second
)

func onDispatchEvent(eventType string) {
	// TODO
}

func onGuildDispatchEvent(eventType string, guildID discord.Snowflake) {
	// TODO
}

func safeOnGuildDispatchEvent(eventType string, guildID *discord.Snowflake) {
	var guildIDValue discord.Snowflake

	if guildID != nil {
		guildIDValue = *guildID
	}

	onGuildDispatchEvent(eventType, guildIDValue)
}

// OnReady handles the READY event.
// It will go and mark guilds as unavailable and go through
// any GUILD_CREATE events for the next few seconds.
func OnReady(ctx context.Context, shard *Shard, msg discord.GatewayPayload, trace *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var readyPayload discord.Ready

	var readyGatewayURL struct {
		ReadyGatewayURL string `json:"resume_gateway_url"`
	}

	err := unmarshalPayload(msg, &readyPayload)
	if err != nil {
		shard.logger.Error("Failed to unmarshal ready payload", "error", err)

		return DispatchResult{nil, nil}, false, err
	}

	err = unmarshalPayload(msg, &readyGatewayURL)
	if err != nil {
		shard.logger.Error("Failed to unmarshal ready gateway url", "error", err)

		return DispatchResult{nil, nil}, false, err
	}

	shard.logger.Info("Received READY payload")

	shard.sessionID.Store(&readyPayload.SessionID)
	shard.resumeGatewayURL.Store(&readyGatewayURL.ReadyGatewayURL)

	shard.manager.user.Store(&readyPayload.User)

	for _, guild := range readyPayload.Guilds {
		shard.lazyGuilds.Store(guild.ID, true)
		shard.guilds.Store(guild.ID, true)
	}

	guildCreateEvents := 0

	readyTimeout := time.NewTicker(ReadyTimeout)

	shard.logger.Info("Starting lazy loading guilds")

ready:
	for {
		select {
		case <-readyTimeout.C:
			slog.Info("Finished lazy loading guilds", "guilds", guildCreateEvents)

			break ready
		default:
		}

		msg, err := shard.read(ctx, shard.websocketConn)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				shard.logger.Error("Encountered error during READY", "error", err)
			}

			break ready
		}

		if msg.Type == discord.DiscordEventGuildCreate {
			guildCreateEvents++

			shard.logger.Debug("Received GUILD_CREATE event", "guilds", guildCreateEvents)

			readyTimeout.Reset(ReadyTimeout)
		}

		err = shard.OnEvent(ctx, msg, trace)
		if err != nil && !errors.Is(err, ErrNoDispatchHandler) {
			shard.logger.Error("Failed to dispatch event", "error", err)
		}
	}

	shard.logger.Info("Finished lazy loading guilds", "guilds", guildCreateEvents)

	shard.logger.Info("Shard is ready")

	select {
	case shard.ready <- struct{}{}:
	default:
	}

	// ctx.SetStatus(sandwich_structs.ShardStatusReady)

	configuration := shard.manager.configuration.Load()

	if configuration.ChunkGuildsOnStart {
		shard.chunkAllGuilds(ctx)
	}

	return DispatchResult{nil, nil}, false, nil
}

func OnResumed(_ context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	shard.logger.Info("Shard has resumed")

	select {
	case shard.ready <- struct{}{}:
	default:
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnApplicationCommandCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var applicationCommandCreatePayload discord.ApplicationCommandCreate

	err := unmarshalPayload(msg, &applicationCommandCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnApplicationCommandUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var applicationCommandUpdatePayload discord.ApplicationCommandUpdate

	err := unmarshalPayload(msg, &applicationCommandUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnApplicationCommandDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var applicationCommandDeletePayload discord.ApplicationCommandDelete

	err := unmarshalPayload(msg, &applicationCommandDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildCreate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildCreatePayload discord.GuildCreate

	err := unmarshalPayload(msg, &guildCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildCreatePayload.ID)

	shard.sandwich.stateProvider.SetGuild(ctx, guildCreatePayload.ID, discord.Guild(guildCreatePayload))

	lazy, exists := shard.lazyGuilds.Load(guildCreatePayload.ID)
	if exists {
		shard.lazyGuilds.Delete(guildCreatePayload.ID)
	}

	unavailable, exists := shard.unavailableGuilds.Load(guildCreatePayload.ID)
	if exists {
		shard.unavailableGuilds.Delete(guildCreatePayload.ID)
	}

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"lazy":        lazy,
			"unavailable": unavailable,
		},
	}, true, nil
}

func OnGuildMembersChunk(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var guildMembersChunkPayload discord.GuildMembersChunk

	err := unmarshalPayload(msg, &guildMembersChunkPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	for _, member := range guildMembersChunkPayload.Members {
		shard.sandwich.stateProvider.SetGuildMember(ctx, guildMembersChunkPayload.GuildID, member)
	}

	shard.logger.Debug("Chunked guild members", "memberCount", len(guildMembersChunkPayload.Members), "chunkIndex", guildMembersChunkPayload.ChunkIndex, "chunkCount", guildMembersChunkPayload.ChunkCount, "guildID", guildMembersChunkPayload.GuildID)

	guildChunk, exists := shard.sandwich.guildChunks.Load(guildMembersChunkPayload.GuildID)

	if !exists {
		shard.logger.Warn("Received guild member chunk, but there is no record in the GuildChunks map", "guildID", guildMembersChunkPayload.GuildID)

		return DispatchResult{nil, nil}, false, nil
	}

	if guildChunk.complete.Load() {
		shard.logger.Warn("Received guild member chunk, but there is no record in the GuildChunks map", "guildID", guildMembersChunkPayload.GuildID)
	}

	select {
	case guildChunk.chunkingChannel <- GuildChunkPartial{
		chunkIndex: guildMembersChunkPayload.ChunkIndex,
		chunkCount: guildMembersChunkPayload.ChunkCount,
		nonce:      guildMembersChunkPayload.Nonce,
	}:
	default:
	}

	return DispatchResult{nil, nil}, false, nil
}

func OnChannelCreate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelCreatePayload discord.ChannelCreate

	err := unmarshalPayload(msg, &channelCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, channelCreatePayload.GuildID)

	shard.sandwich.stateProvider.SetGuildChannel(ctx, *channelCreatePayload.GuildID, discord.Channel(channelCreatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnChannelUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelUpdatePayload discord.ChannelUpdate

	err := unmarshalPayload(msg, &channelUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, channelUpdatePayload.GuildID)

	beforeChannel, _ := shard.sandwich.stateProvider.GetGuildChannel(ctx, *channelUpdatePayload.GuildID, channelUpdatePayload.ID)
	shard.sandwich.stateProvider.SetGuildChannel(ctx, *channelUpdatePayload.GuildID, discord.Channel(channelUpdatePayload))

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeChannel,
		},
	}, true, nil
}

func OnChannelDelete(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelDeletePayload discord.ChannelDelete

	err := unmarshalPayload(msg, &channelDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, channelDeletePayload.GuildID)

	beforeChannel, _ := shard.sandwich.stateProvider.GetGuildChannel(ctx, *channelDeletePayload.GuildID, channelDeletePayload.ID)
	shard.sandwich.stateProvider.RemoveGuildChannel(ctx, *channelDeletePayload.GuildID, channelDeletePayload.ID)

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeChannel,
		},
	}, true, nil
}

func OnChannelPinsUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelPinsUpdatePayload discord.ChannelPinsUpdate

	err := unmarshalPayload(msg, &channelPinsUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, channelPinsUpdatePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnThreadCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadCreatePayload discord.ThreadCreate

	err := unmarshalPayload(msg, &threadCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnThreadUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadUpdatePayload discord.ThreadUpdate

	err := unmarshalPayload(msg, &threadUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	beforeChannel, _ := shard.sandwich.stateProvider.GetGuildChannel(ctx, *threadUpdatePayload.GuildID, threadUpdatePayload.ID)

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeChannel,
		},
	}, true, nil
}

func OnThreadDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadDeletePayload discord.ThreadDelete

	err := unmarshalPayload(msg, &threadDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnThreadListSync(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadListSyncPayload discord.ThreadListSync

	err := unmarshalPayload(msg, &threadListSyncPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnThreadMemberUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadMemberUpdatePayload discord.ThreadMemberUpdate

	err := unmarshalPayload(msg, &threadMemberUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnThreadMembersUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadMembersUpdatePayload discord.ThreadMembersUpdate

	err := unmarshalPayload(msg, &threadMembersUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildAuditLogEntryCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var threadMembersUpdatePayload discord.GuildAuditLogEntryCreate

	err := unmarshalPayload(msg, &threadMembersUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnEntitlementCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var entitlementCreatePayload discord.Entitlement

	err := unmarshalPayload(msg, &entitlementCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnEntitlementUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var entitlementUpdatePayload discord.Entitlement

	err := unmarshalPayload(msg, &entitlementUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnEntitlementDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var entitlementUpdatePayload discord.Entitlement

	err := unmarshalPayload(msg, &entitlementUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildUpdatePayload discord.GuildUpdate

	err := unmarshalPayload(msg, &guildUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildUpdatePayload.ID)

	beforeGuild, exists := shard.sandwich.stateProvider.GetGuild(ctx, guildUpdatePayload.ID)

	if exists {
		// Preserve values only present in GUILD_CREATE events.
		if guildUpdatePayload.StageInstances == nil {
			guildUpdatePayload.StageInstances = beforeGuild.StageInstances
		}

		if guildUpdatePayload.Channels == nil {
			guildUpdatePayload.Channels = beforeGuild.Channels
		}

		if guildUpdatePayload.Members == nil {
			guildUpdatePayload.Members = beforeGuild.Members
		}

		if guildUpdatePayload.VoiceStates == nil {
			guildUpdatePayload.VoiceStates = beforeGuild.VoiceStates
		}

		if guildUpdatePayload.MemberCount == 0 {
			guildUpdatePayload.MemberCount = beforeGuild.MemberCount
		}

		guildUpdatePayload.Large = beforeGuild.Large
		guildUpdatePayload.JoinedAt = beforeGuild.JoinedAt
	} else {
		shard.logger.Warn("Received "+discord.DiscordEventGuildUpdate+" event, but previous guild not present in state", "guild_id", guildUpdatePayload.ID)
	}

	shard.sandwich.stateProvider.SetGuild(ctx, guildUpdatePayload.ID, discord.Guild(guildUpdatePayload))

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeGuild,
		},
	}, true, nil
}

func OnGuildDelete(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildDeletePayload discord.GuildDelete

	err := unmarshalPayload(msg, &guildDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildDeletePayload.ID)

	beforeGuild, _ := shard.sandwich.stateProvider.GetGuild(ctx, guildDeletePayload.ID)

	if guildDeletePayload.Unavailable {
		shard.unavailableGuilds.Store(guildDeletePayload.ID, true)
	} else {
		// We do not remove the actual guild as other managers may be using it.
		// Dereferencing it locally ensures that if other managers are using it,
		// it will stay.
		shard.guilds.Delete(guildDeletePayload.ID)
		shard.manager.guilds.Delete(guildDeletePayload.ID)
	}

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeGuild,
		},
	}, true, nil
}

func OnGuildBanAdd(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildBanAddPayload discord.GuildBanAdd

	err := unmarshalPayload(msg, &guildBanAddPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, *guildBanAddPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildBanRemove(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildBanRemovePayload discord.GuildBanRemove

	err := unmarshalPayload(msg, &guildBanRemovePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, *guildBanRemovePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildEmojisUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildEmojisUpdatePayload discord.GuildEmojisUpdate

	err := unmarshalPayload(msg, &guildEmojisUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildEmojisUpdatePayload.GuildID)

	beforeEmojis, _ := shard.sandwich.stateProvider.GetGuildEmojis(ctx, guildEmojisUpdatePayload.GuildID)

	shard.sandwich.stateProvider.SetGuildEmojis(ctx, guildEmojisUpdatePayload.GuildID, guildEmojisUpdatePayload.Emojis)

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeEmojis,
		},
	}, true, nil
}

func OnGuildStickersUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildStickersUpdatePayload discord.GuildStickersUpdate

	err := unmarshalPayload(msg, &guildStickersUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildStickersUpdatePayload.GuildID)

	beforeGuild, exists := shard.sandwich.stateProvider.GetGuild(ctx, guildStickersUpdatePayload.GuildID)
	beforeStickers := beforeGuild.Stickers

	if exists {
		beforeGuild.Stickers = guildStickersUpdatePayload.Stickers

		// TODO add stickers to state

		shard.sandwich.stateProvider.SetGuild(ctx, beforeGuild.ID, beforeGuild)
	} else {
		shard.logger.Warn("Received "+discord.DiscordEventGuildStickersUpdate+" event, however guild is not present in state", "guild_id", guildStickersUpdatePayload.GuildID)
	}

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeStickers,
		},
	}, true, nil
}

func OnGuildIntegrationsUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildIntegrationsUpdatePayload discord.GuildIntegrationsUpdate

	err := unmarshalPayload(msg, &guildIntegrationsUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildIntegrationsUpdatePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildMemberAdd(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildMemberAddPayload discord.GuildMemberAdd

	err := unmarshalPayload(msg, &guildMemberAddPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	guild, exists := shard.sandwich.stateProvider.GetGuild(ctx, *guildMemberAddPayload.GuildID)
	if exists {
		guild.MemberCount++
		shard.sandwich.stateProvider.SetGuild(ctx, *guildMemberAddPayload.GuildID, guild)
	}

	defer onGuildDispatchEvent(msg.Type, *guildMemberAddPayload.GuildID)

	shard.sandwich.stateProvider.SetGuildMember(ctx, *guildMemberAddPayload.GuildID, discord.GuildMember(guildMemberAddPayload))
	shard.sandwich.stateProvider.AddUserMutualGuild(ctx, guildMemberAddPayload.User.ID, *guildMemberAddPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildMemberRemove(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildMemberRemovePayload discord.GuildMemberRemove

	err := unmarshalPayload(msg, &guildMemberRemovePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	// TODO: Implement deduping.

	guild, exists := shard.sandwich.stateProvider.GetGuild(ctx, guildMemberRemovePayload.GuildID)

	if exists {
		guild.MemberCount--
		shard.sandwich.stateProvider.SetGuild(ctx, guildMemberRemovePayload.GuildID, guild)
	}

	defer onGuildDispatchEvent(msg.Type, guildMemberRemovePayload.GuildID)

	guildMember, _ := shard.sandwich.stateProvider.GetGuildMember(ctx, guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)

	shard.sandwich.stateProvider.RemoveGuildMember(ctx, guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)
	shard.sandwich.stateProvider.RemoveUserMutualGuild(ctx, guildMemberRemovePayload.User.ID, guildMemberRemovePayload.GuildID)

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": guildMember,
		},
	}, true, nil
}

func OnGuildMemberUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildMemberUpdatePayload discord.GuildMemberUpdate

	err := unmarshalPayload(msg, &guildMemberUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, *guildMemberUpdatePayload.GuildID)

	beforeGuildMember, _ := shard.sandwich.stateProvider.GetGuildMember(
		ctx,
		*guildMemberUpdatePayload.GuildID,
		guildMemberUpdatePayload.User.ID,
	)

	shard.sandwich.stateProvider.SetGuildMember(ctx, *guildMemberUpdatePayload.GuildID, discord.GuildMember(guildMemberUpdatePayload))

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeGuildMember,
		},
	}, true, nil
}

func OnGuildRoleCreate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildRoleCreatePayload discord.GuildRoleCreate

	err := unmarshalPayload(msg, &guildRoleCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, *guildRoleCreatePayload.GuildID)

	shard.sandwich.stateProvider.SetGuildRole(ctx, *guildRoleCreatePayload.GuildID, discord.Role(guildRoleCreatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildRoleUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildRoleUpdatePayload discord.GuildRoleUpdate

	err := unmarshalPayload(msg, &guildRoleUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildRoleUpdatePayload.GuildID)

	beforeRole, _ := shard.sandwich.stateProvider.GetGuildRole(ctx, guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role.ID)

	shard.sandwich.stateProvider.SetGuildRole(ctx, guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role)

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeRole,
		},
	}, true, nil
}

func OnGuildRoleDelete(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildRoleDeletePayload discord.GuildRoleDelete

	err := unmarshalPayload(msg, &guildRoleDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, guildRoleDeletePayload.GuildID)

	shard.sandwich.stateProvider.RemoveGuildRole(ctx, guildRoleDeletePayload.GuildID, guildRoleDeletePayload.RoleID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnIntegrationCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var integrationCreatePayload discord.IntegrationCreate

	err := unmarshalPayload(msg, &integrationCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnIntegrationUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var integrationUpdatePayload discord.IntegrationUpdate

	err := unmarshalPayload(msg, &integrationUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnIntegrationDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var integrationDeletePayload discord.IntegrationDelete

	err := unmarshalPayload(msg, &integrationDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnInteractionCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var interactionCreatePayload discord.InteractionCreate

	err := unmarshalPayload(msg, &interactionCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnInviteCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var inviteCreatePayload discord.InviteCreate

	err := unmarshalPayload(msg, &inviteCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	if inviteCreatePayload.GuildID != nil {
		defer onGuildDispatchEvent(msg.Type, *inviteCreatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnInviteDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var inviteDeletePayload discord.InviteDelete

	err := unmarshalPayload(msg, &inviteDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	if inviteDeletePayload.GuildID != nil {
		defer onGuildDispatchEvent(msg.Type, *inviteDeletePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageCreatePayload discord.MessageCreate

	err := unmarshalPayload(msg, &messageCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, messageCreatePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageUpdatePayload discord.MessageUpdate

	err := unmarshalPayload(msg, &messageUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, messageUpdatePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageDeletePayload discord.MessageDelete

	err := unmarshalPayload(msg, &messageDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, messageDeletePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageDeleteBulk(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageDeleteBulkPayload discord.MessageDeleteBulk

	err := unmarshalPayload(msg, &messageDeleteBulkPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, messageDeleteBulkPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionAdd(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionAddPayload discord.MessageReactionAdd

	err := unmarshalPayload(msg, &messageReactionAddPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, messageReactionAddPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionRemove(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionRemovePayload discord.MessageReactionRemove

	err := unmarshalPayload(msg, &messageReactionRemovePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, messageReactionRemovePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionRemoveAll(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionRemoveAllPayload discord.MessageReactionRemoveAll

	err := unmarshalPayload(msg, &messageReactionRemoveAllPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, messageReactionRemoveAllPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionRemoveEmoji(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionRemoveEmojiPayload discord.MessageReactionRemoveEmoji

	err := unmarshalPayload(msg, &messageReactionRemoveEmojiPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, messageReactionRemoveEmojiPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnPresenceUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var presenceUpdatePayload discord.PresenceUpdate

	err := unmarshalPayload(msg, &presenceUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onGuildDispatchEvent(msg.Type, presenceUpdatePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnStageInstanceCreate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var stageInstanceCreatePayload discord.StageInstanceCreate

	err := unmarshalPayload(msg, &stageInstanceCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnStageInstanceUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var stageInstanceUpdatePayload discord.StageInstanceUpdate

	err := unmarshalPayload(msg, &stageInstanceUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnStageInstanceDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var stageInstanceDeletePayload discord.StageInstanceDelete

	err := unmarshalPayload(msg, &stageInstanceDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnTypingStart(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var typingStartPayload discord.TypingStart

	err := unmarshalPayload(msg, &typingStartPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, typingStartPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnUserUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var userUpdatePayload discord.UserUpdate

	err := unmarshalPayload(msg, &userUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	beforeUser, _ := shard.sandwich.stateProvider.GetUser(ctx, userUpdatePayload.ID)

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeUser,
		},
	}, true, nil
}

func OnVoiceStateUpdate(ctx context.Context, shard *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var voiceStateUpdatePayload discord.VoiceStateUpdate

	err := unmarshalPayload(msg, &voiceStateUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer safeOnGuildDispatchEvent(msg.Type, voiceStateUpdatePayload.GuildID)

	var guildID discord.Snowflake

	if voiceStateUpdatePayload.GuildID != nil {
		guildID = *voiceStateUpdatePayload.GuildID
	}

	beforeVoiceState, _ := shard.sandwich.stateProvider.GetVoiceState(ctx, guildID, voiceStateUpdatePayload.UserID)

	if guildID.IsNil() {
		shard.sandwich.stateProvider.RemoveVoiceState(ctx, *voiceStateUpdatePayload.GuildID, voiceStateUpdatePayload.UserID)
	} else {
		shard.sandwich.stateProvider.SetVoiceState(ctx, *voiceStateUpdatePayload.GuildID, discord.VoiceState(voiceStateUpdatePayload))
	}

	return DispatchResult{
		Data: msg.Data,
		Extra: Extra{
			"before": beforeVoiceState,
		},
	}, true, nil
}

func OnVoiceServerUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var voiceServerUpdatePayload discord.VoiceServerUpdate

	err := unmarshalPayload(msg, &voiceServerUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnWebhookUpdate(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var webhookUpdatePayload discord.WebhookUpdate

	err := unmarshalPayload(msg, &webhookUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildJoinRequestDelete(_ context.Context, _ *Shard, msg discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(msg.Type)

	var guildJoinRequestDeletePayload discord.GuildJoinRequestDelete

	err := unmarshalPayload(msg, &guildJoinRequestDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func init() {
	registerDispatchHandler(discord.DiscordEventReady, OnReady)
	registerDispatchHandler(discord.DiscordEventResumed, OnResumed)
	registerDispatchHandler(discord.DiscordEventApplicationCommandCreate, OnApplicationCommandCreate)
	registerDispatchHandler(discord.DiscordEventApplicationCommandUpdate, OnApplicationCommandUpdate)
	registerDispatchHandler(discord.DiscordEventApplicationCommandDelete, OnApplicationCommandDelete)
	registerDispatchHandler(discord.DiscordEventGuildMembersChunk, OnGuildMembersChunk)
	registerDispatchHandler(discord.DiscordEventChannelCreate, OnChannelCreate)
	registerDispatchHandler(discord.DiscordEventChannelUpdate, OnChannelUpdate)
	registerDispatchHandler(discord.DiscordEventChannelDelete, OnChannelDelete)
	registerDispatchHandler(discord.DiscordEventChannelPinsUpdate, OnChannelPinsUpdate)
	registerDispatchHandler(discord.DiscordEventThreadCreate, OnThreadCreate)
	registerDispatchHandler(discord.DiscordEventThreadUpdate, OnThreadUpdate)
	registerDispatchHandler(discord.DiscordEventThreadDelete, OnThreadDelete)
	registerDispatchHandler(discord.DiscordEventThreadListSync, OnThreadListSync)
	registerDispatchHandler(discord.DiscordEventThreadMemberUpdate, OnThreadMemberUpdate)
	registerDispatchHandler(discord.DiscordEventThreadMembersUpdate, OnThreadMembersUpdate)
	registerDispatchHandler(discord.DiscordEventGuildAuditLogEntryCreate, OnGuildAuditLogEntryCreate)
	registerDispatchHandler(discord.DiscordEventEntitlementCreate, OnEntitlementCreate)
	registerDispatchHandler(discord.DiscordEventEntitlementUpdate, OnEntitlementUpdate)
	registerDispatchHandler(discord.DiscordEventEntitlementDelete, OnEntitlementDelete)
	registerDispatchHandler(discord.DiscordEventGuildCreate, OnGuildCreate)
	registerDispatchHandler(discord.DiscordEventGuildUpdate, OnGuildUpdate)
	registerDispatchHandler(discord.DiscordEventGuildDelete, OnGuildDelete)
	registerDispatchHandler(discord.DiscordEventGuildBanAdd, OnGuildBanAdd)
	registerDispatchHandler(discord.DiscordEventGuildBanRemove, OnGuildBanRemove)
	registerDispatchHandler(discord.DiscordEventGuildEmojisUpdate, OnGuildEmojisUpdate)
	registerDispatchHandler(discord.DiscordEventGuildStickersUpdate, OnGuildStickersUpdate)
	registerDispatchHandler(discord.DiscordEventGuildIntegrationsUpdate, OnGuildIntegrationsUpdate)
	registerDispatchHandler(discord.DiscordEventGuildMemberAdd, OnGuildMemberAdd)
	registerDispatchHandler(discord.DiscordEventGuildMemberRemove, OnGuildMemberRemove)
	registerDispatchHandler(discord.DiscordEventGuildMemberUpdate, OnGuildMemberUpdate)
	registerDispatchHandler(discord.DiscordEventGuildRoleCreate, OnGuildRoleCreate)
	registerDispatchHandler(discord.DiscordEventGuildRoleUpdate, OnGuildRoleUpdate)
	registerDispatchHandler(discord.DiscordEventGuildRoleDelete, OnGuildRoleDelete)
	registerDispatchHandler(discord.DiscordEventIntegrationCreate, OnIntegrationCreate)
	registerDispatchHandler(discord.DiscordEventIntegrationUpdate, OnIntegrationUpdate)
	registerDispatchHandler(discord.DiscordEventIntegrationDelete, OnIntegrationDelete)
	registerDispatchHandler(discord.DiscordEventInteractionCreate, OnInteractionCreate)
	registerDispatchHandler(discord.DiscordEventInviteCreate, OnInviteCreate)
	registerDispatchHandler(discord.DiscordEventInviteDelete, OnInviteDelete)
	registerDispatchHandler(discord.DiscordEventMessageCreate, OnMessageCreate)
	registerDispatchHandler(discord.DiscordEventMessageUpdate, OnMessageUpdate)
	registerDispatchHandler(discord.DiscordEventMessageDelete, OnMessageDelete)
	registerDispatchHandler(discord.DiscordEventMessageDeleteBulk, OnMessageDeleteBulk)
	registerDispatchHandler(discord.DiscordEventMessageReactionAdd, OnMessageReactionAdd)
	registerDispatchHandler(discord.DiscordEventMessageReactionRemove, OnMessageReactionRemove)
	registerDispatchHandler(discord.DiscordEventMessageReactionRemoveAll, OnMessageReactionRemoveAll)
	registerDispatchHandler(discord.DiscordEventMessageReactionRemoveEmoji, OnMessageReactionRemoveEmoji)
	registerDispatchHandler(discord.DiscordEventPresenceUpdate, OnPresenceUpdate)
	registerDispatchHandler(discord.DiscordEventStageInstanceCreate, OnStageInstanceCreate)
	registerDispatchHandler(discord.DiscordEventStageInstanceUpdate, OnStageInstanceUpdate)
	registerDispatchHandler(discord.DiscordEventStageInstanceDelete, OnStageInstanceDelete)
	registerDispatchHandler(discord.DiscordEventTypingStart, OnTypingStart)
	registerDispatchHandler(discord.DiscordEventUserUpdate, OnUserUpdate)
	registerDispatchHandler(discord.DiscordEventVoiceStateUpdate, OnVoiceStateUpdate)
	registerDispatchHandler(discord.DiscordEventVoiceServerUpdate, OnVoiceServerUpdate)
	registerDispatchHandler(discord.DiscordEventWebhookUpdate, OnWebhookUpdate)

	// Discord Undocumented
	registerDispatchHandler(discord.DiscordEventGuildJoinRequestDelete, OnGuildJoinRequestDelete)
}
