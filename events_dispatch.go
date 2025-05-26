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

func onDispatchEvent(shard *Shard, eventType string) {
	RecordEvent(shard.Application.Identifier, eventType)
}

// OnReady handles the READY event.
// It will go and mark guilds as unavailable and go through
// any GUILD_CREATE events for the next few seconds.
func OnReady(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

	var readyPayload discord.Ready

	var readyGatewayURL struct {
		ReadyGatewayURL string `json:"resume_gateway_url"`
	}

	err := unmarshalPayload(msg, &readyPayload)
	if err != nil {
		shard.Logger.Error("Failed to unmarshal ready payload", "error", err)

		return DispatchResult{nil, nil}, false, err
	}

	err = unmarshalPayload(msg, &readyGatewayURL)
	if err != nil {
		shard.Logger.Error("Failed to unmarshal ready gateway url", "error", err)

		return DispatchResult{nil, nil}, false, err
	}

	shard.Logger.Debug("Received READY payload")

	shard.sessionID.Store(&readyPayload.SessionID)
	shard.resumeGatewayURL.Store(&readyGatewayURL.ReadyGatewayURL)

	shard.Application.SetUser(&readyPayload.User)

	for _, guild := range readyPayload.Guilds {
		shard.lazyGuilds.Store(guild.ID, true)
		shard.guilds.Store(guild.ID, true)
	}

	guildCreateEvents := 0

	readyTimeout := time.NewTicker(ReadyTimeout)

	shard.Logger.Debug("Starting lazy loading guilds")

ready:
	for {
		select {
		case <-readyTimeout.C:
			slog.Debug("Finished lazy loading guilds", "guilds", guildCreateEvents)

			break ready
		default:
		}

		msg, err := shard.read(ctx, shard.websocketConn)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				shard.Logger.Error("Encountered error during READY", "error", err)
			}

			break ready
		}

		if msg.Type == discord.DiscordEventGuildCreate {
			guildCreateEvents++

			shard.Logger.Debug("Received GUILD_CREATE event", "guilds", guildCreateEvents)

			readyTimeout.Reset(ReadyTimeout)
		}

		err = shard.OnEvent(ctx, msg, trace)
		if err != nil && !errors.Is(err, ErrNoDispatchHandler) {
			shard.Logger.Error("Failed to dispatch event", "error", err)
		}

		if msg != nil {
			shard.gatewayPayloadPool.Put(msg)
		} else {
			shard.Logger.Warn("Attempt to put nil message into pool", "loc", "OnReady")
		}
	}

	shard.Logger.Debug("Finished lazy loading guilds", "guilds", guildCreateEvents)

	shard.Logger.Debug("Shard is ready")

	select {
	case shard.ready <- struct{}{}:
	default:
	}

	// ctx.SetStatus(sandwich_structs.ShardStatusReady)

	configuration := shard.Application.Configuration.Load()

	if configuration.ChunkGuildsOnStart {
		shard.chunkAllGuilds(ctx)
	}

	return DispatchResult{nil, nil}, false, nil
}

func OnResumed(_ context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

	shard.Logger.Debug("Shard has resumed")

	select {
	case shard.ready <- struct{}{}:
	default:
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnApplicationCommandCreate(_ context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnApplicationCommandUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnApplicationCommandDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnGuildCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildCreatePayload discord.GuildCreate

	err := unmarshalPayload(msg, &guildCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildCreatePayload.ID.IsNil() {
		ctx = WithGuildID(ctx, guildCreatePayload.ID)
	}

	shard.Sandwich.stateProvider.SetGuild(ctx, guildCreatePayload.ID, discord.Guild(guildCreatePayload))

	lazy, exists := shard.lazyGuilds.Load(guildCreatePayload.ID)

	if exists {
		shard.lazyGuilds.Delete(guildCreatePayload.ID)
	}

	unavailable, exists := shard.unavailableGuilds.Load(guildCreatePayload.ID)

	if exists {
		shard.unavailableGuilds.Delete(guildCreatePayload.ID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("lazy", lazy).Set("unavailable", unavailable),
	}, true, nil
}

func OnGuildMembersChunk(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

	var guildMembersChunkPayload discord.GuildMembersChunk

	err := unmarshalPayload(msg, &guildMembersChunkPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	for _, member := range guildMembersChunkPayload.Members {
		shard.Sandwich.stateProvider.SetGuildMember(ctx, guildMembersChunkPayload.GuildID, member)
	}

	shard.Logger.Debug("Chunked guild members", "memberCount", len(guildMembersChunkPayload.Members), "chunkIndex", guildMembersChunkPayload.ChunkIndex, "chunkCount", guildMembersChunkPayload.ChunkCount, "guildID", guildMembersChunkPayload.GuildID)

	guildChunk, exists := shard.Sandwich.guildChunks.Load(guildMembersChunkPayload.GuildID)

	if !exists {
		shard.Logger.Warn("Received guild member chunk, but there is no record in the GuildChunks map", "guildID", guildMembersChunkPayload.GuildID)

		return DispatchResult{nil, nil}, false, nil
	}

	if guildChunk.complete.Load() {
		shard.Logger.Warn("Received guild member chunk, but it is already complete", "guildID", guildMembersChunkPayload.GuildID)
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

func OnChannelCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelCreatePayload discord.ChannelCreate

	err := unmarshalPayload(msg, &channelCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if channelCreatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *channelCreatePayload.GuildID)
	}

	shard.Sandwich.stateProvider.SetGuildChannel(ctx, *channelCreatePayload.GuildID, discord.Channel(channelCreatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnChannelUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelUpdatePayload discord.ChannelUpdate

	err := unmarshalPayload(msg, &channelUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if channelUpdatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *channelUpdatePayload.GuildID)
	}

	beforeChannel, ok := shard.Sandwich.stateProvider.GetGuildChannel(ctx, *channelUpdatePayload.GuildID, channelUpdatePayload.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventChannelUpdate+" event, but previous channel not present in state", "guild_id", *channelUpdatePayload.GuildID, "channel_id", channelUpdatePayload.ID)
	}

	shard.Sandwich.stateProvider.SetGuildChannel(ctx, *channelUpdatePayload.GuildID, discord.Channel(channelUpdatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeChannel),
	}, true, nil
}

func OnChannelDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelDeletePayload discord.ChannelDelete

	err := unmarshalPayload(msg, &channelDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if channelDeletePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *channelDeletePayload.GuildID)
	}

	beforeChannel, ok := shard.Sandwich.stateProvider.GetGuildChannel(ctx, *channelDeletePayload.GuildID, channelDeletePayload.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventChannelDelete+" event, but previous channel not present in state", "guild_id", *channelDeletePayload.GuildID, "channel_id", channelDeletePayload.ID)
	}

	shard.Sandwich.stateProvider.RemoveGuildChannel(ctx, *channelDeletePayload.GuildID, channelDeletePayload.ID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeChannel),
	}, true, nil
}

func OnChannelPinsUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var channelPinsUpdatePayload discord.ChannelPinsUpdate

	err := unmarshalPayload(msg, &channelPinsUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !channelPinsUpdatePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, channelPinsUpdatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnThreadCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnThreadUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

	var threadUpdatePayload discord.ThreadUpdate

	err := unmarshalPayload(msg, &threadUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	beforeChannel, ok := shard.Sandwich.stateProvider.GetGuildChannel(ctx, *threadUpdatePayload.GuildID, threadUpdatePayload.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventThreadUpdate+" event, but previous channel not present in state", "guild_id", *threadUpdatePayload.GuildID, "channel_id", threadUpdatePayload.ID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeChannel),
	}, true, nil
}

func OnThreadDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnThreadListSync(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnThreadMemberUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnThreadMembersUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnGuildAuditLogEntryCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnEntitlementCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnEntitlementUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnEntitlementDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnGuildUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildUpdatePayload discord.GuildUpdate

	err := unmarshalPayload(msg, &guildUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildUpdatePayload.ID.IsNil() {
		ctx = WithGuildID(ctx, guildUpdatePayload.ID)
	}

	beforeGuild, exists := shard.Sandwich.stateProvider.GetGuild(ctx, guildUpdatePayload.ID)

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
		shard.Logger.Warn("Received "+discord.DiscordEventGuildUpdate+" event, but previous guild not present in state", "guild_id", guildUpdatePayload.ID)
	}

	shard.Sandwich.stateProvider.SetGuild(ctx, guildUpdatePayload.ID, discord.Guild(guildUpdatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeGuild),
	}, true, nil
}

func OnGuildDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildDeletePayload discord.GuildDelete

	err := unmarshalPayload(msg, &guildDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildDeletePayload.ID.IsNil() {
		ctx = WithGuildID(ctx, guildDeletePayload.ID)
	}

	beforeGuild, ok := shard.Sandwich.stateProvider.GetGuild(ctx, guildDeletePayload.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventGuildDelete+" event, but previous guild not present in state", "guild_id", guildDeletePayload.ID)
	}

	if guildDeletePayload.Unavailable {
		shard.unavailableGuilds.Store(guildDeletePayload.ID, true)
	} else {
		// We do not remove the actual guild as other applications may be using it.
		// Dereferencing it locally ensures that if other applications are using it,
		// it will stay.
		shard.guilds.Delete(guildDeletePayload.ID)
		shard.Application.guilds.Delete(guildDeletePayload.ID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeGuild),
	}, true, nil
}

func OnGuildBanAdd(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildBanAddPayload discord.GuildBanAdd

	err := unmarshalPayload(msg, &guildBanAddPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if guildBanAddPayload.GuildID != nil {
		ctx = WithGuildID(ctx, *guildBanAddPayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildBanRemove(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildBanRemovePayload discord.GuildBanRemove

	err := unmarshalPayload(msg, &guildBanRemovePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if guildBanRemovePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *guildBanRemovePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildEmojisUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildEmojisUpdatePayload discord.GuildEmojisUpdate

	err := unmarshalPayload(msg, &guildEmojisUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildEmojisUpdatePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, guildEmojisUpdatePayload.GuildID)
	}

	beforeEmojis, ok := shard.Sandwich.stateProvider.GetGuildEmojis(ctx, guildEmojisUpdatePayload.GuildID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventGuildEmojisUpdate+" event, but previous emojis not present in state", "guild_id", guildEmojisUpdatePayload.GuildID)
	}

	shard.Sandwich.stateProvider.SetGuildEmojis(ctx, guildEmojisUpdatePayload.GuildID, guildEmojisUpdatePayload.Emojis)

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeEmojis),
	}, true, nil
}

func OnGuildStickersUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildStickersUpdatePayload discord.GuildStickersUpdate

	err := unmarshalPayload(msg, &guildStickersUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildStickersUpdatePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, guildStickersUpdatePayload.GuildID)
	}

	beforeGuild, exists := shard.Sandwich.stateProvider.GetGuild(ctx, guildStickersUpdatePayload.GuildID)
	beforeStickers := beforeGuild.Stickers

	if exists {
		beforeGuild.Stickers = guildStickersUpdatePayload.Stickers

		shard.Sandwich.stateProvider.SetGuildStickers(ctx, beforeGuild.ID, guildStickersUpdatePayload.Stickers)
	} else {
		shard.Logger.Warn("Received "+discord.DiscordEventGuildStickersUpdate+" event, however guild is not present in state", "guild_id", guildStickersUpdatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeStickers),
	}, true, nil
}

func OnGuildIntegrationsUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildIntegrationsUpdatePayload discord.GuildIntegrationsUpdate

	err := unmarshalPayload(msg, &guildIntegrationsUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildIntegrationsUpdatePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, guildIntegrationsUpdatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildMemberAdd(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildMemberAddPayload discord.GuildMemberAdd

	err := unmarshalPayload(msg, &guildMemberAddPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	guild, exists := shard.Sandwich.stateProvider.GetGuild(ctx, *guildMemberAddPayload.GuildID)

	if exists {
		guild.MemberCount++
		shard.Sandwich.stateProvider.SetGuild(ctx, *guildMemberAddPayload.GuildID, *guild)
	}

	defer onDispatchEvent(shard, msg.Type)

	if guildMemberAddPayload.GuildID != nil {
		ctx = WithGuildID(ctx, *guildMemberAddPayload.GuildID)
	}

	shard.Sandwich.stateProvider.SetGuildMember(ctx, *guildMemberAddPayload.GuildID, discord.GuildMember(guildMemberAddPayload))
	shard.Sandwich.stateProvider.AddUserMutualGuild(ctx, guildMemberAddPayload.User.ID, *guildMemberAddPayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildMemberRemove(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildMemberRemovePayload discord.GuildMemberRemove

	err := unmarshalPayload(msg, &guildMemberRemovePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	// TODO: Implement deduping.

	guild, exists := shard.Sandwich.stateProvider.GetGuild(ctx, guildMemberRemovePayload.GuildID)

	if exists {
		guild.MemberCount--
		shard.Sandwich.stateProvider.SetGuild(ctx, guildMemberRemovePayload.GuildID, *guild)
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildMemberRemovePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, guildMemberRemovePayload.GuildID)
	}

	guildMember, ok := shard.Sandwich.stateProvider.GetGuildMember(ctx, guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventGuildMemberRemove+" event, but previous guild member not present in state", "guild_id", guildMemberRemovePayload.GuildID, "user_id", guildMemberRemovePayload.User.ID)
	}

	shard.Sandwich.stateProvider.RemoveGuildMember(ctx, guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)
	shard.Sandwich.stateProvider.RemoveUserMutualGuild(ctx, guildMemberRemovePayload.User.ID, guildMemberRemovePayload.GuildID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", guildMember),
	}, true, nil
}

func OnGuildMemberUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildMemberUpdatePayload discord.GuildMemberUpdate

	err := unmarshalPayload(msg, &guildMemberUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if guildMemberUpdatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *guildMemberUpdatePayload.GuildID)
	}

	beforeGuildMember, ok := shard.Sandwich.stateProvider.GetGuildMember(
		ctx,
		*guildMemberUpdatePayload.GuildID,
		guildMemberUpdatePayload.User.ID,
	)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventGuildMemberUpdate+" event, but previous guild member not present in state", "guild_id", *guildMemberUpdatePayload.GuildID, "user_id", guildMemberUpdatePayload.User.ID)
	}

	shard.Sandwich.stateProvider.SetGuildMember(ctx, *guildMemberUpdatePayload.GuildID, discord.GuildMember(guildMemberUpdatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeGuildMember),
	}, true, nil
}

func OnGuildRoleCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildRoleCreatePayload discord.GuildRoleCreate

	err := unmarshalPayload(msg, &guildRoleCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if guildRoleCreatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *guildRoleCreatePayload.GuildID)
	}

	shard.Sandwich.stateProvider.SetGuildRole(ctx, *guildRoleCreatePayload.GuildID, discord.Role(guildRoleCreatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnGuildRoleUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildRoleUpdatePayload discord.GuildRoleUpdate

	err := unmarshalPayload(msg, &guildRoleUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildRoleUpdatePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, guildRoleUpdatePayload.GuildID)
	}

	beforeRole, ok := shard.Sandwich.stateProvider.GetGuildRole(ctx, guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventGuildRoleUpdate+" event, but previous guild role not present in state", "guild_id", guildRoleUpdatePayload.GuildID, "role_id", guildRoleUpdatePayload.Role.ID)
	}

	shard.Sandwich.stateProvider.SetGuildRole(ctx, guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role)

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeRole),
	}, true, nil
}

func OnGuildRoleDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var guildRoleDeletePayload discord.GuildRoleDelete

	err := unmarshalPayload(msg, &guildRoleDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !guildRoleDeletePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, guildRoleDeletePayload.GuildID)
	}

	shard.Sandwich.stateProvider.RemoveGuildRole(ctx, guildRoleDeletePayload.GuildID, guildRoleDeletePayload.RoleID)

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnIntegrationCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnIntegrationUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnIntegrationDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnInteractionCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnInviteCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var inviteCreatePayload discord.InviteCreate

	err := unmarshalPayload(msg, &inviteCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if inviteCreatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *inviteCreatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnInviteDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var inviteDeletePayload discord.InviteDelete

	err := unmarshalPayload(msg, &inviteDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if inviteDeletePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *inviteDeletePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageCreatePayload discord.MessageCreate

	err := unmarshalPayload(msg, &messageCreatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if messageCreatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *messageCreatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageUpdatePayload discord.MessageUpdate

	err := unmarshalPayload(msg, &messageUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if messageUpdatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *messageUpdatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageDeletePayload discord.MessageDelete

	err := unmarshalPayload(msg, &messageDeletePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if messageDeletePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *messageDeletePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageDeleteBulk(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageDeleteBulkPayload discord.MessageDeleteBulk

	err := unmarshalPayload(msg, &messageDeleteBulkPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if messageDeleteBulkPayload.GuildID != nil {
		ctx = WithGuildID(ctx, *messageDeleteBulkPayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionAdd(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionAddPayload discord.MessageReactionAdd

	err := unmarshalPayload(msg, &messageReactionAddPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !messageReactionAddPayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, messageReactionAddPayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionRemove(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionRemovePayload discord.MessageReactionRemove

	err := unmarshalPayload(msg, &messageReactionRemovePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if messageReactionRemovePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *messageReactionRemovePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionRemoveAll(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionRemoveAllPayload discord.MessageReactionRemoveAll

	err := unmarshalPayload(msg, &messageReactionRemoveAllPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !messageReactionRemoveAllPayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, messageReactionRemoveAllPayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnMessageReactionRemoveEmoji(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var messageReactionRemoveEmojiPayload discord.MessageReactionRemoveEmoji

	err := unmarshalPayload(msg, &messageReactionRemoveEmojiPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if messageReactionRemoveEmojiPayload.GuildID != nil {
		ctx = WithGuildID(ctx, *messageReactionRemoveEmojiPayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnPresenceUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var presenceUpdatePayload discord.PresenceUpdate

	err := unmarshalPayload(msg, &presenceUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if !presenceUpdatePayload.GuildID.IsNil() {
		ctx = WithGuildID(ctx, presenceUpdatePayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnStageInstanceCreate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnStageInstanceUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnStageInstanceDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnTypingStart(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var typingStartPayload discord.TypingStart

	err := unmarshalPayload(msg, &typingStartPayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if typingStartPayload.GuildID != nil {
		ctx = WithGuildID(ctx, *typingStartPayload.GuildID)
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: nil,
	}, true, nil
}

func OnUserUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

	var userUpdatePayload discord.UserUpdate

	err := unmarshalPayload(msg, &userUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	beforeUser, ok := shard.Sandwich.stateProvider.GetUser(ctx, userUpdatePayload.ID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventUserUpdate+" event, but previous user not present in state", "user_id", userUpdatePayload.ID)
	}

	shard.Sandwich.stateProvider.SetUser(ctx, userUpdatePayload.ID, discord.User(userUpdatePayload))

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeUser),
	}, true, nil
}

func OnVoiceStateUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	var voiceStateUpdatePayload discord.VoiceStateUpdate

	err := unmarshalPayload(msg, &voiceStateUpdatePayload)
	if err != nil {
		return DispatchResult{nil, nil}, false, err
	}

	defer onDispatchEvent(shard, msg.Type)

	if voiceStateUpdatePayload.GuildID != nil {
		ctx = WithGuildID(ctx, *voiceStateUpdatePayload.GuildID)
	}

	var guildID discord.Snowflake

	if voiceStateUpdatePayload.GuildID != nil {
		guildID = *voiceStateUpdatePayload.GuildID
	}

	beforeVoiceState, ok := shard.Sandwich.stateProvider.GetVoiceState(ctx, guildID, voiceStateUpdatePayload.UserID)
	if !ok {
		shard.Logger.Warn("Received "+discord.DiscordEventVoiceStateUpdate+" event, but previous voice state not present in state", "guild_id", guildID, "user_id", voiceStateUpdatePayload.UserID)
	}

	if voiceStateUpdatePayload.ChannelID.IsNil() {
		shard.Sandwich.stateProvider.RemoveVoiceState(ctx, *voiceStateUpdatePayload.GuildID, voiceStateUpdatePayload.UserID)
	} else {
		shard.Sandwich.stateProvider.SetVoiceState(ctx, *voiceStateUpdatePayload.GuildID, discord.VoiceState(voiceStateUpdatePayload))
	}

	return DispatchResult{
		Data:  msg.Data,
		Extra: NewExtra().Set("before", beforeVoiceState),
	}, true, nil
}

func OnVoiceServerUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnWebhookUpdate(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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

func OnGuildJoinRequestDelete(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) (DispatchResult, bool, error) {
	onDispatchEvent(shard, msg.Type)

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
	RegisterDispatchHandler(discord.DiscordEventReady, OnReady)
	RegisterDispatchHandler(discord.DiscordEventResumed, OnResumed)
	RegisterDispatchHandler(discord.DiscordEventApplicationCommandCreate, OnApplicationCommandCreate)
	RegisterDispatchHandler(discord.DiscordEventApplicationCommandUpdate, OnApplicationCommandUpdate)
	RegisterDispatchHandler(discord.DiscordEventApplicationCommandDelete, OnApplicationCommandDelete)
	RegisterDispatchHandler(discord.DiscordEventGuildMembersChunk, OnGuildMembersChunk)
	RegisterDispatchHandler(discord.DiscordEventChannelCreate, OnChannelCreate)
	RegisterDispatchHandler(discord.DiscordEventChannelUpdate, OnChannelUpdate)
	RegisterDispatchHandler(discord.DiscordEventChannelDelete, OnChannelDelete)
	RegisterDispatchHandler(discord.DiscordEventChannelPinsUpdate, OnChannelPinsUpdate)
	RegisterDispatchHandler(discord.DiscordEventThreadCreate, OnThreadCreate)
	RegisterDispatchHandler(discord.DiscordEventThreadUpdate, OnThreadUpdate)
	RegisterDispatchHandler(discord.DiscordEventThreadDelete, OnThreadDelete)
	RegisterDispatchHandler(discord.DiscordEventThreadListSync, OnThreadListSync)
	RegisterDispatchHandler(discord.DiscordEventThreadMemberUpdate, OnThreadMemberUpdate)
	RegisterDispatchHandler(discord.DiscordEventThreadMembersUpdate, OnThreadMembersUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildAuditLogEntryCreate, OnGuildAuditLogEntryCreate)
	RegisterDispatchHandler(discord.DiscordEventEntitlementCreate, OnEntitlementCreate)
	RegisterDispatchHandler(discord.DiscordEventEntitlementUpdate, OnEntitlementUpdate)
	RegisterDispatchHandler(discord.DiscordEventEntitlementDelete, OnEntitlementDelete)
	RegisterDispatchHandler(discord.DiscordEventGuildCreate, OnGuildCreate)
	RegisterDispatchHandler(discord.DiscordEventGuildUpdate, OnGuildUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildDelete, OnGuildDelete)
	RegisterDispatchHandler(discord.DiscordEventGuildBanAdd, OnGuildBanAdd)
	RegisterDispatchHandler(discord.DiscordEventGuildBanRemove, OnGuildBanRemove)
	RegisterDispatchHandler(discord.DiscordEventGuildEmojisUpdate, OnGuildEmojisUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildStickersUpdate, OnGuildStickersUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildIntegrationsUpdate, OnGuildIntegrationsUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildMemberAdd, OnGuildMemberAdd)
	RegisterDispatchHandler(discord.DiscordEventGuildMemberRemove, OnGuildMemberRemove)
	RegisterDispatchHandler(discord.DiscordEventGuildMemberUpdate, OnGuildMemberUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildRoleCreate, OnGuildRoleCreate)
	RegisterDispatchHandler(discord.DiscordEventGuildRoleUpdate, OnGuildRoleUpdate)
	RegisterDispatchHandler(discord.DiscordEventGuildRoleDelete, OnGuildRoleDelete)
	RegisterDispatchHandler(discord.DiscordEventIntegrationCreate, OnIntegrationCreate)
	RegisterDispatchHandler(discord.DiscordEventIntegrationUpdate, OnIntegrationUpdate)
	RegisterDispatchHandler(discord.DiscordEventIntegrationDelete, OnIntegrationDelete)
	RegisterDispatchHandler(discord.DiscordEventInteractionCreate, OnInteractionCreate)
	RegisterDispatchHandler(discord.DiscordEventInviteCreate, OnInviteCreate)
	RegisterDispatchHandler(discord.DiscordEventInviteDelete, OnInviteDelete)
	RegisterDispatchHandler(discord.DiscordEventMessageCreate, OnMessageCreate)
	RegisterDispatchHandler(discord.DiscordEventMessageUpdate, OnMessageUpdate)
	RegisterDispatchHandler(discord.DiscordEventMessageDelete, OnMessageDelete)
	RegisterDispatchHandler(discord.DiscordEventMessageDeleteBulk, OnMessageDeleteBulk)
	RegisterDispatchHandler(discord.DiscordEventMessageReactionAdd, OnMessageReactionAdd)
	RegisterDispatchHandler(discord.DiscordEventMessageReactionRemove, OnMessageReactionRemove)
	RegisterDispatchHandler(discord.DiscordEventMessageReactionRemoveAll, OnMessageReactionRemoveAll)
	RegisterDispatchHandler(discord.DiscordEventMessageReactionRemoveEmoji, OnMessageReactionRemoveEmoji)
	RegisterDispatchHandler(discord.DiscordEventPresenceUpdate, OnPresenceUpdate)
	RegisterDispatchHandler(discord.DiscordEventStageInstanceCreate, OnStageInstanceCreate)
	RegisterDispatchHandler(discord.DiscordEventStageInstanceUpdate, OnStageInstanceUpdate)
	RegisterDispatchHandler(discord.DiscordEventStageInstanceDelete, OnStageInstanceDelete)
	RegisterDispatchHandler(discord.DiscordEventTypingStart, OnTypingStart)
	RegisterDispatchHandler(discord.DiscordEventUserUpdate, OnUserUpdate)
	RegisterDispatchHandler(discord.DiscordEventVoiceStateUpdate, OnVoiceStateUpdate)
	RegisterDispatchHandler(discord.DiscordEventVoiceServerUpdate, OnVoiceServerUpdate)
	RegisterDispatchHandler(discord.DiscordEventWebhookUpdate, OnWebhookUpdate)

	// Discord Undocumented
	RegisterDispatchHandler(discord.DiscordEventGuildJoinRequestDelete, OnGuildJoinRequestDelete)
}
