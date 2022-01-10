package internal

import (
	"context"
	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"golang.org/x/xerrors"
	"time"
)

// OnReady handles the READY event.
// It will go and mark guilds as unavailable and go through
// any GUILD_CREATE events for the next few seconds.
func OnReady(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var readyPayload discord.Ready

	var guildCreatePayload discord.GuildCreate

	err = ctx.decodeContent(msg, &readyPayload)
	if err != nil {
		return
	}

	ctx.Logger.Info().Msg("Received READY payload")

	ctx.SessionID.Store(readyPayload.SessionID)

	ctx.ShardGroup.userMu.Lock()
	ctx.ShardGroup.User = &readyPayload.User
	ctx.Manager.UserID.Store(int64(readyPayload.User.ID))

	ctx.Manager.userMu.Lock()
	ctx.Manager.User = readyPayload.User
	ctx.Manager.userMu.Unlock()

	ctx.ShardGroup.userMu.Unlock()

	ctx.lazyMu.Lock()
	ctx.guildsMu.Lock()

	for _, guild := range readyPayload.Guilds {
		ctx.Lazy[guild.ID] = true
		ctx.Guilds[guild.ID] = true
	}

	ctx.guildsMu.Unlock()
	ctx.lazyMu.Unlock()

	guildCreateEvents := 0

	readyTimeout := time.NewTicker(ReadyTimeout)

ready:
	for {
		select {
		case <-ctx.ErrorCh:
			if !xerrors.Is(err, context.Canceled) {
				ctx.Logger.Error().Err(err).Msg("Encountered error during READY")
			}

			break ready
		case msg := <-ctx.MessageCh:
			if msg.Type == "GUILD_CREATE" {
				guildCreateEvents++

				err = ctx.decodeContent(msg, &guildCreatePayload)
				if err != nil {
					ctx.Logger.Error().Err(err).Str("type", msg.Type).Msg("Failed to decode event")
				}

				readyTimeout.Reset(ReadyTimeout)
			}

			err = ctx.OnDispatch(ctx.context, msg)
			if err != nil && !xerrors.Is(err, ErrNoDispatchHandler) {
				ctx.Logger.Error().Err(err).Msg("Failed to dispatch event")
			}
		case <-readyTimeout.C:
			ctx.Logger.Info().Int("guilds", guildCreateEvents).Msg("Finished lazy loading guilds")

			break ready
		}
	}

	select {
	case ctx.ready <- void{}:
	default:
	}

	ctx.SetStatus(structs.ShardStatusReady)

	return result, false, nil
}

func OnResumed(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	select {
	case ctx.ready <- void{}:
	default:
	}

	ctx.SetStatus(structs.ShardStatusReady)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnApplicationCommandCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var applicationCommandCreatePayload discord.ApplicationCommandCreate

	err = ctx.decodeContent(msg, &applicationCommandCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnApplicationCommandUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var applicationCommandUpdatePayload discord.ApplicationCommandUpdate

	err = ctx.decodeContent(msg, &applicationCommandUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnApplicationCommandDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var applicationCommandDeletePayload discord.ApplicationCommandDelete

	err = ctx.decodeContent(msg, &applicationCommandDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildCreatePayload discord.GuildCreate

	err = ctx.decodeContent(msg, &guildCreatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildCreatePayload.ID)

	ctx.Sandwich.State.SetGuild(ctx, guildCreatePayload.Guild)

	ctx.lazyMu.Lock()
	lazy := ctx.Lazy[guildCreatePayload.ID]
	delete(ctx.Lazy, guildCreatePayload.ID)
	ctx.lazyMu.Unlock()

	ctx.unavailableMu.Lock()
	unavailable := ctx.Unavailable[guildCreatePayload.ID]
	delete(ctx.Unavailable, guildCreatePayload.ID)
	ctx.unavailableMu.Unlock()

	extra, err := makeExtra(map[string]interface{}{
		"lazy":        lazy,
		"unavailable": unavailable,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildMembersChunk(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var guildMembersChunkPayload discord.GuildMembersChunk

	err = ctx.decodeContent(msg, &guildMembersChunkPayload)
	if err != nil {
		return
	}

	if ctx.CacheMembers {
		for _, member := range guildMembersChunkPayload.Members {
			ctx.Sandwich.State.SetGuildMember(ctx, guildMembersChunkPayload.GuildID, member)
		}
	}

	ctx.Logger.Debug().
		Int("memberCount", len(guildMembersChunkPayload.Members)).
		Int("chunkIndex", guildMembersChunkPayload.ChunkIndex).
		Int("chunkCount", guildMembersChunkPayload.ChunkCount).
		Int64("guildID", int64(guildMembersChunkPayload.GuildID)).
		Msg("Chunked guild members")

	return result, false, nil
}

func OnChannelCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var channelCreatePayload discord.ChannelCreate

	err = ctx.decodeContent(msg, &channelCreatePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, channelCreatePayload.GuildID)

	ctx.Sandwich.State.SetGuildChannel(ctx, channelCreatePayload.GuildID, channelCreatePayload)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnChannelUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var channelUpdatePayload discord.ChannelUpdate

	err = ctx.decodeContent(msg, &channelUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, channelUpdatePayload.GuildID)

	beforeChannel, _ := ctx.Sandwich.State.GetGuildChannel(channelUpdatePayload.GuildID, channelUpdatePayload.ID)
	ctx.Sandwich.State.SetGuildChannel(ctx, channelUpdatePayload.GuildID, channelUpdatePayload)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeChannel,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnChannelDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var channelDeletePayload discord.ChannelDelete

	err = ctx.decodeContent(msg, &channelDeletePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, channelDeletePayload.GuildID)

	beforeChannel, _ := ctx.Sandwich.State.GetGuildChannel(channelDeletePayload.GuildID, channelDeletePayload.ID)
	ctx.Sandwich.State.RemoveGuildChannel(channelDeletePayload.GuildID, channelDeletePayload.ID)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeChannel,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnChannelPinsUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var channelPinsUpdatePayload discord.ChannelPinsUpdate

	err = ctx.decodeContent(msg, &channelPinsUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, channelPinsUpdatePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnThreadCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var threadCreatePayload discord.ThreadCreate

	err = ctx.decodeContent(msg, &threadCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnThreadUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var threadUpdatePayload discord.ThreadUpdate

	err = ctx.decodeContent(msg, &threadUpdatePayload)
	if err != nil {
		return
	}

	beforeChannel, _ := ctx.Sandwich.State.GetGuildChannel(threadUpdatePayload.GuildID, threadUpdatePayload.ID)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeChannel,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnThreadDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var threadDeletePayload discord.ThreadDelete

	err = ctx.decodeContent(msg, &threadDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnThreadListSync(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var threadListSyncPayload discord.ThreadListSync

	err = ctx.decodeContent(msg, &threadListSyncPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnThreadMemberUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var threadMemberUpdatePayload discord.ThreadMemberUpdate

	err = ctx.decodeContent(msg, &threadMemberUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnThreadMembersUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var threadMembersUpdatePayload discord.ThreadMembersUpdate

	err = ctx.decodeContent(msg, &threadMembersUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildUpdatePayload discord.GuildUpdate

	err = ctx.decodeContent(msg, &guildUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildUpdatePayload.ID)

	beforeGuild, _ := ctx.Sandwich.State.GetGuild(guildUpdatePayload.ID)

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

	if guildUpdatePayload.MemberCount == nil {
		guildUpdatePayload.MemberCount = beforeGuild.MemberCount
	}

	if guildUpdatePayload.Unavailable == nil {
		guildUpdatePayload.Unavailable = beforeGuild.Unavailable
	}

	if guildUpdatePayload.Large == nil {
		guildUpdatePayload.Large = beforeGuild.Large
	}

	if guildUpdatePayload.JoinedAt == nil {
		guildUpdatePayload.JoinedAt = beforeGuild.JoinedAt
	}

	ctx.Sandwich.State.SetGuild(ctx, guildUpdatePayload)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeGuild,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildDeletePayload discord.GuildDelete

	err = ctx.decodeContent(msg, &guildDeletePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildDeletePayload.ID)

	beforeGuild, _ := ctx.Sandwich.State.GetGuild(guildDeletePayload.ID)

	if guildDeletePayload.Unavailable {
		ctx.Unavailable[guildDeletePayload.ID] = true
	} else {
		// We do not remove the actual guild as other managers may be using it.
		// Dereferencing it locally ensures that if other managers are using it,
		// it will stay.
		ctx.ShardGroup.guildsMu.Lock()
		delete(ctx.ShardGroup.Guilds, guildDeletePayload.ID)
		ctx.ShardGroup.guildsMu.Unlock()
	}

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeGuild,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildBanAdd(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildBanAddPayload discord.GuildBanAdd

	err = ctx.decodeContent(msg, &guildBanAddPayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildBanAddPayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildBanRemove(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildBanRemovePayload discord.GuildBanRemove

	err = ctx.decodeContent(msg, &guildBanRemovePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildBanRemovePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildEmojisUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildEmojisUpdatePayload discord.GuildEmojisUpdate

	err = ctx.decodeContent(msg, &guildEmojisUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildEmojisUpdatePayload.GuildID)

	beforeEmojis, _ := ctx.Sandwich.State.GetAllGuildEmojis(guildEmojisUpdatePayload.GuildID)

	ctx.Sandwich.State.RemoveAllGuildEmojis(guildEmojisUpdatePayload.GuildID)

	for _, emoji := range guildEmojisUpdatePayload.Emojis {
		ctx.Sandwich.State.SetGuildEmoji(ctx, guildEmojisUpdatePayload.GuildID, emoji)
	}

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeEmojis,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildStickersUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildStickersUpdatePayload discord.GuildStickersUpdate

	err = ctx.decodeContent(msg, &guildStickersUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildStickersUpdatePayload.GuildID)

	beforeGuild, _ := ctx.Sandwich.State.GetGuild(guildStickersUpdatePayload.GuildID)
	beforeStickers := beforeGuild.Stickers

	beforeGuild.Stickers = guildStickersUpdatePayload.Stickers

	ctx.Sandwich.State.SetGuild(ctx, beforeGuild)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeStickers,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildIntegrationsUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildIntegrationsUpdatePayload discord.GuildIntegrationsUpdate

	err = ctx.decodeContent(msg, &guildIntegrationsUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildIntegrationsUpdatePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildMemberAdd(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMemberAddPayload discord.GuildMemberAdd

	err = ctx.decodeContent(msg, &guildMemberAddPayload)
	if err != nil {
		return
	}

	ddRemoveKey := createDedupeMemberRemoveKey(guildMemberAddPayload.GuildID, guildMemberAddPayload.User.ID)
	ddAddKey := createDedupeMemberAddKey(guildMemberAddPayload.GuildID, guildMemberAddPayload.User.ID)

	if !ctx.Sandwich.CheckDedupe(ddAddKey) {
		ctx.Sandwich.AddDedupe(ddAddKey)
		ctx.Sandwich.RemoveDedupe(ddRemoveKey)

		ctx.Sandwich.State.guildsMu.Lock()
		guild, ok := ctx.Sandwich.State.Guilds[guildMemberAddPayload.GuildID]

		if ok {
			if guild.MemberCount != nil {
				memberCount := *guild.MemberCount
				memberCount++
				guild.MemberCount = &memberCount
				ctx.Sandwich.State.Guilds[guildMemberAddPayload.GuildID] = guild
			} else {
				ctx.Sandwich.Logger.Fatal().
					Int("guildID", int(guild.ID)).
					Msg("Guild does not reference member count")
			}
		}
		ctx.Sandwich.State.guildsMu.Unlock()
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildMemberAddPayload.GuildID)

	if ctx.CacheMembers {
		ctx.Sandwich.State.SetGuildMember(ctx, guildMemberAddPayload.GuildID, guildMemberAddPayload.GuildMember)
	}

	if ctx.StoreMutuals {
		ctx.Sandwich.State.AddUserMutualGuild(ctx, guildMemberAddPayload.User.ID, guildMemberAddPayload.GuildID)
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildMemberRemove(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMemberRemovePayload discord.GuildMemberRemove

	err = ctx.decodeContent(msg, &guildMemberRemovePayload)
	if err != nil {
		return
	}

	ddRemoveKey := createDedupeMemberRemoveKey(guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)
	ddAddKey := createDedupeMemberAddKey(guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)

	if !ctx.Sandwich.CheckDedupe(ddRemoveKey) {
		ctx.Sandwich.AddDedupe(ddRemoveKey)
		ctx.Sandwich.RemoveDedupe(ddAddKey)

		ctx.Sandwich.State.guildsMu.Lock()
		guild, ok := ctx.Sandwich.State.Guilds[guildMemberRemovePayload.GuildID]

		if ok {
			if guild.MemberCount != nil {
				memberCount := *guild.MemberCount
				memberCount--
				guild.MemberCount = &memberCount
				ctx.Sandwich.State.Guilds[guildMemberRemovePayload.GuildID] = guild
			} else {
				ctx.Sandwich.Logger.Fatal().
					Int("guildID", int(guild.ID)).
					Msg("Guild does not reference member count")
			}
		}
		ctx.Sandwich.State.guildsMu.Unlock()
	}
	defer ctx.OnGuildDispatchEvent(msg.Type, guildMemberRemovePayload.GuildID)

	guildMember, _ := ctx.Sandwich.State.GetGuildMember(guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)

	ctx.Sandwich.State.RemoveGuildMember(guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)
	ctx.Sandwich.State.RemoveUserMutualGuild(guildMemberRemovePayload.User.ID, guildMemberRemovePayload.GuildID)

	extra, err := makeExtra(map[string]interface{}{
		"before": guildMember,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildMemberUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMemberUpdatePayload discord.GuildMemberUpdate

	err = ctx.decodeContent(msg, &guildMemberUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildMemberUpdatePayload.GuildID)

	beforeGuildMember, _ := ctx.Sandwich.State.GetGuildMember(
		guildMemberUpdatePayload.GuildID, guildMemberUpdatePayload.User.ID)

	if ctx.CacheMembers {
		ctx.Sandwich.State.SetGuildMember(ctx, guildMemberUpdatePayload.GuildID, guildMemberUpdatePayload.GuildMember)
	}

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeGuildMember,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildRoleCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildRoleCreatePayload discord.GuildRoleCreate

	err = ctx.decodeContent(msg, &guildRoleCreatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildRoleCreatePayload.GuildID)

	ctx.Sandwich.State.SetGuildRole(ctx, guildRoleCreatePayload.GuildID, guildRoleCreatePayload.Role)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildRoleUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildRoleUpdatePayload discord.GuildRoleUpdate

	err = ctx.decodeContent(msg, &guildRoleUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildRoleUpdatePayload.GuildID)

	beforeRole, _ := ctx.Sandwich.State.GetGuildRole(
		guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role.ID)

	ctx.Sandwich.State.SetGuildRole(ctx, guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeRole,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnGuildRoleDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildRoleDeletePayload discord.GuildRoleDelete

	err = ctx.decodeContent(msg, &guildRoleDeletePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, guildRoleDeletePayload.GuildID)

	ctx.Sandwich.State.RemoveGuildRole(guildRoleDeletePayload.GuildID, guildRoleDeletePayload.RoleID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnIntegrationCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var integrationCreatePayload discord.IntegrationCreate

	err = ctx.decodeContent(msg, &integrationCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnIntegrationUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var integrationUpdatePayload discord.IntegrationUpdate

	err = ctx.decodeContent(msg, &integrationUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnIntegrationDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var integrationDeletePayload discord.IntegrationDelete

	err = ctx.decodeContent(msg, &integrationDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnInteractionCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var interactionCreatePayload discord.InteractionCreate

	err = ctx.decodeContent(msg, &interactionCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnInviteCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var inviteCreatePayload discord.InviteCreate

	err = ctx.decodeContent(msg, &inviteCreatePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, inviteCreatePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnInviteDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var inviteDeletePayload discord.InviteDelete

	err = ctx.decodeContent(msg, &inviteDeletePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, inviteDeletePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageCreatePayload discord.MessageCreate

	err = ctx.decodeContent(msg, &messageCreatePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, messageCreatePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageUpdatePayload discord.MessageUpdate

	err = ctx.decodeContent(msg, &messageUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, messageUpdatePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageDeletePayload discord.MessageDelete

	err = ctx.decodeContent(msg, &messageDeletePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, messageDeletePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageDeleteBulk(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageDeleteBulkPayload discord.MessageDeleteBulk

	err = ctx.decodeContent(msg, &messageDeleteBulkPayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, messageDeleteBulkPayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageReactionAdd(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionAddPayload discord.MessageReactionAdd

	err = ctx.decodeContent(msg, &messageReactionAddPayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, messageReactionAddPayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageReactionRemove(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionRemovePayload discord.MessageReactionRemove

	err = ctx.decodeContent(msg, &messageReactionRemovePayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, messageReactionRemovePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageReactionRemoveAll(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionRemoveAllPayload discord.MessageReactionRemoveAll

	err = ctx.decodeContent(msg, &messageReactionRemoveAllPayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, messageReactionRemoveAllPayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnMessageReactionRemoveEmoji(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionRemoveEmojiPayload discord.MessageReactionRemoveEmoji

	err = ctx.decodeContent(msg, &messageReactionRemoveEmojiPayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, messageReactionRemoveEmojiPayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnPresenceUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var presenceUpdatePayload discord.PresenceUpdate

	err = ctx.decodeContent(msg, &presenceUpdatePayload)
	if err != nil {
		return
	}

	defer ctx.OnGuildDispatchEvent(msg.Type, presenceUpdatePayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnStageInstanceCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var stageInstanceCreatePayload discord.StageInstanceCreate

	err = ctx.decodeContent(msg, &stageInstanceCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnStageInstanceUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var stageInstanceUpdatePayload discord.StageInstanceUpdate

	err = ctx.decodeContent(msg, &stageInstanceUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnStageInstanceDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var stageInstanceDeletePayload discord.StageInstanceDelete

	err = ctx.decodeContent(msg, &stageInstanceDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnTypingStart(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var typingStartPayload discord.TypingStart

	err = ctx.decodeContent(msg, &typingStartPayload)
	if err != nil {
		return
	}

	defer ctx.SafeOnGuildDispatchEvent(msg.Type, typingStartPayload.GuildID)

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnUserUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var userUpdatePayload discord.UserUpdate

	err = ctx.decodeContent(msg, &userUpdatePayload)
	if err != nil {
		return
	}

	beforeUser, _ := ctx.Sandwich.State.GetUser(userUpdatePayload.ID)

	extra, err := makeExtra(map[string]interface{}{
		"before": beforeUser,
	})
	if err != nil {
		return result, ok, xerrors.Errorf("Failed to marshal extras: %v", err)
	}

	return structs.StateResult{
		Data:  msg.Data,
		Extra: extra,
	}, true, nil
}

func OnVoiceStateUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var voiceStateUpdatePayload discord.VoiceStateUpdate

	err = ctx.decodeContent(msg, &voiceStateUpdatePayload)
	if err != nil {
		return
	}

	if voiceStateUpdatePayload.GuildID != nil {
		defer ctx.OnGuildDispatchEvent(msg.Type, *voiceStateUpdatePayload.GuildID)
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnVoiceServerUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var voiceServerUpdatePayload discord.VoiceServerUpdate

	err = ctx.decodeContent(msg, &voiceServerUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnWebhookUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var webhookUpdatePayload discord.WebhookUpdate

	err = ctx.decodeContent(msg, &webhookUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnGuildJoinRequestDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	defer ctx.OnDispatchEvent(msg.Type)

	var guildJoinRequestDeletePayload discord.GuildJoinRequestDelete

	err = ctx.decodeContent(msg, &guildJoinRequestDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func init() {
	registerDispatch("READY", OnReady)
	registerDispatch("RESUMED", OnResumed)
	registerDispatch("APPLICATION_COMMAND_CREATE", OnApplicationCommandCreate)
	registerDispatch("APPLICATION_COMMAND_UPDATE", OnApplicationCommandUpdate)
	registerDispatch("APPLICATION_COMMAND_DELETE", OnApplicationCommandDelete)
	registerDispatch("GUILD_MEMBERS_CHUNK", OnGuildMembersChunk)
	registerDispatch("CHANNEL_CREATE", OnChannelCreate)
	registerDispatch("CHANNEL_UPDATE", OnChannelUpdate)
	registerDispatch("CHANNEL_DELETE", OnChannelDelete)
	registerDispatch("CHANNEL_PINS_UPDATE", OnChannelPinsUpdate)
	registerDispatch("THREAD_CREATE", OnThreadCreate)
	registerDispatch("THREAD_UPDATE", OnThreadUpdate)
	registerDispatch("THREAD_DELETE", OnThreadDelete)
	registerDispatch("THREAD_LIST_SYNC", OnThreadListSync)
	registerDispatch("THREAD_MEMBER_UPDATE", OnThreadMemberUpdate)
	registerDispatch("THREAD_MEMBERS_UPDATE", OnThreadMembersUpdate)
	registerDispatch("GUILD_CREATE", OnGuildCreate)
	registerDispatch("GUILD_UPDATE", OnGuildUpdate)
	registerDispatch("GUILD_DELETE", OnGuildDelete)
	registerDispatch("GUILD_BAN_ADD", OnGuildBanAdd)
	registerDispatch("GUILD_BAN_REMOVE", OnGuildBanRemove)
	registerDispatch("GUILD_EMOJIS_UPDATE", OnGuildEmojisUpdate)
	registerDispatch("GUILD_STICKERS_UPDATE", OnGuildStickersUpdate)
	registerDispatch("GUILD_INTEGRATIONS_UPDATE", OnGuildIntegrationsUpdate)
	registerDispatch("GUILD_MEMBER_ADD", OnGuildMemberAdd)
	registerDispatch("GUILD_MEMBER_REMOVE", OnGuildMemberRemove)
	registerDispatch("GUILD_MEMBER_UPDATE", OnGuildMemberUpdate)
	registerDispatch("GUILD_ROLE_CREATE", OnGuildRoleCreate)
	registerDispatch("GUILD_ROLE_UPDATE", OnGuildRoleUpdate)
	registerDispatch("GUILD_ROLE_DELETE", OnGuildRoleDelete)
	registerDispatch("INTEGRATION_CREATE", OnIntegrationCreate)
	registerDispatch("INTEGRATION_UPDATE", OnIntegrationUpdate)
	registerDispatch("INTEGRATION_DELETE", OnIntegrationDelete)
	registerDispatch("INTERACTION_CREATE", OnInteractionCreate)
	registerDispatch("INVITE_CREATE", OnInviteCreate)
	registerDispatch("INVITE_DELETE", OnInviteDelete)
	registerDispatch("MESSAGE_CREATE", OnMessageCreate)
	registerDispatch("MESSAGE_UPDATE", OnMessageUpdate)
	registerDispatch("MESSAGE_DELETE", OnMessageDelete)
	registerDispatch("MESSAGE_DELETE_BULK", OnMessageDeleteBulk)
	registerDispatch("MESSAGE_REACTION_ADD", OnMessageReactionAdd)
	registerDispatch("MESSAGE_REACTION_REMOVE", OnMessageReactionRemove)
	registerDispatch("MESSAGE_REACTION_REMOVE_ALL", OnMessageReactionRemoveAll)
	registerDispatch("MESSAGE_REACTION_REMOVE_EMOJI", OnMessageReactionRemoveEmoji)
	registerDispatch("PRESENCE_UPDATE", OnPresenceUpdate)
	registerDispatch("STAGE_INSTANCE_CREATE", OnStageInstanceCreate)
	registerDispatch("STAGE_INSTANCE_UPDATE", OnStageInstanceUpdate)
	registerDispatch("STAGE_INSTANCE_DELETE", OnStageInstanceDelete)
	registerDispatch("TYPING_START", OnTypingStart)
	registerDispatch("USER_UPDATE", OnUserUpdate)
	registerDispatch("VOICE_STATE_UPDATE", OnVoiceStateUpdate)
	registerDispatch("VOICE_SERVER_UPDATE", OnVoiceServerUpdate)
	registerDispatch("WEBHOOKS_UPDATE", OnWebhookUpdate)

	// Discord Undocumented
	registerDispatch("GUILD_JOIN_REQUEST_DELETE", OnGuildJoinRequestDelete)
}
