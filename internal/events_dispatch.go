package internal

import (
	"context"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"golang.org/x/xerrors"
)

// OnReady handles the READY event.
// It will go and mark guilds as unavailable and go through
// any GUILD_CREATE events for the next few seconds.
func OnReady(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
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
	ctx.ShardGroup.userMu.Unlock()

	ctx.unavailableMu.Lock()
	ctx.guildsMu.Lock()

	for _, guild := range readyPayload.Guilds {
		ctx.Unavailable[guild.ID] = true
		ctx.Guilds[guild.ID] = true
	}

	ctx.guildsMu.Unlock()
	ctx.unavailableMu.Unlock()

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

	ctx.ready <- void{}

	return result, false, nil
}

func OnApplicationCommandCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
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

	ctx.Sandwich.State.SetGuild(guildCreatePayload.Guild)

	return structs.StateResult{
		Data: guildCreatePayload,
		Extra: map[string]interface{}{
			"lazy": guildCreatePayload.Lazy,
		},
	}, true, nil
}

func OnGuildMembersChunk(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMembersChunkPayload discord.GuildMembersChunk

	err = ctx.decodeContent(msg, &guildMembersChunkPayload)
	if err != nil {
		return
	}

	for _, member := range guildMembersChunkPayload.Members {
		ctx.Sandwich.State.SetGuildMember(guildMembersChunkPayload.GuildID, member)
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

	ctx.Sandwich.State.SetGuildChannel(channelCreatePayload.GuildID, channelCreatePayload.Channel)

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

	beforeChannel, _ := ctx.Sandwich.State.GetGuildChannel(channelUpdatePayload.GuildID, channelUpdatePayload.ID)
	ctx.Sandwich.State.SetGuildChannel(channelUpdatePayload.GuildID, channelUpdatePayload.Channel)

	return structs.StateResult{
		Data: msg.Data,
		Extra: map[string]interface{}{
			"before": beforeChannel,
		},
	}, true, nil
}

func OnChannelDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var channelDeletePayload discord.ChannelDelete

	err = ctx.decodeContent(msg, &channelDeletePayload)
	if err != nil {
		return
	}

	beforeChannel, _ := ctx.Sandwich.State.GetGuildChannel(channelDeletePayload.GuildID, channelDeletePayload.ID)
	ctx.Sandwich.State.RemoveGuildChannel(channelDeletePayload.GuildID, channelDeletePayload.ID)

	return structs.StateResult{
		Data: msg.Data,
		Extra: map[string]interface{}{
			"before": beforeChannel,
		},
	}, true, nil
}

func OnChannelPinsUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var channelPinsUpdatePayload discord.ChannelPinsUpdate

	err = ctx.decodeContent(msg, &channelPinsUpdatePayload)
	if err != nil {
		return
	}

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
	var threadUpdatePayload discord.ThreadUpdate

	err = ctx.decodeContent(msg, &threadUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg.Data,
	}, true, nil
}

func OnThreadDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
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

	beforeGuild, _ := ctx.Sandwich.State.GetGuild(guildUpdatePayload.ID)
	ctx.Sandwich.State.SetGuild(guildUpdatePayload)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeGuild,
		},
	}, true, nil
}

func OnGuildDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildDeletePayload discord.GuildDelete

	err = ctx.decodeContent(msg, &guildDeletePayload)
	if err != nil {
		return
	}

	beforeGuild, _ := ctx.Sandwich.State.GetGuild(guildDeletePayload.ID)
	ctx.Sandwich.State.RemoveGuild(guildDeletePayload.ID)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeGuild,
		},
	}, true, nil
}

func OnGuildBanAdd(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildBanAddPayload discord.GuildBanAdd

	err = ctx.decodeContent(msg, &guildBanAddPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnGuildBanRemove(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildBanRemovePayload discord.GuildBanRemove

	err = ctx.decodeContent(msg, &guildBanRemovePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnGuildEmojisUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildEmojisUpdatePayload discord.GuildEmojisUpdate

	err = ctx.decodeContent(msg, &guildEmojisUpdatePayload)
	if err != nil {
		return
	}

	beforeEmojis, _ := ctx.Sandwich.State.GetAllGuildEmojis(guildEmojisUpdatePayload.GuildID)

	ctx.Sandwich.State.RemoveAllGuildEmojis(guildEmojisUpdatePayload.GuildID)

	for _, emoji := range guildEmojisUpdatePayload.Emojis {
		ctx.Sandwich.State.SetGuildEmoji(guildEmojisUpdatePayload.GuildID, emoji)
	}

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeEmojis,
		},
	}, true, nil
}

func OnGuildStickersUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildStickersUpdatePayload discord.GuildStickersUpdate

	err = ctx.decodeContent(msg, &guildStickersUpdatePayload)
	if err != nil {
		return
	}

	beforeGuild, _ := ctx.Sandwich.State.GetGuild(guildStickersUpdatePayload.GuildID)
	beforeStickers := beforeGuild.Stickers

	beforeGuild.Stickers = guildStickersUpdatePayload.Stickers

	ctx.Sandwich.State.SetGuild(beforeGuild)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeStickers,
		},
	}, true, nil
}

func OnGuildIntegrationsUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildIntegrationsUpdatePayload discord.GuildIntegrationsUpdate

	err = ctx.decodeContent(msg, &guildIntegrationsUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnGuildMemberAdd(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMemberAddPayload discord.GuildMemberAdd

	err = ctx.decodeContent(msg, &guildMemberAddPayload)
	if err != nil {
		return
	}

	ctx.Sandwich.State.SetGuildMember(guildMemberAddPayload.GuildID, guildMemberAddPayload.GuildMember)

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnGuildMemberRemove(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMemberRemovePayload discord.GuildMemberRemove

	err = ctx.decodeContent(msg, &guildMemberRemovePayload)
	if err != nil {
		return
	}

	guildMember, _ := ctx.Sandwich.State.GetGuildMember(guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)

	ctx.Sandwich.State.RemoveGuildMember(guildMemberRemovePayload.GuildID, guildMemberRemovePayload.User.ID)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": guildMember,
		},
	}, true, nil
}

func OnGuildMemberUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildMemberUpdatePayload discord.GuildMemberUpdate

	err = ctx.decodeContent(msg, &guildMemberUpdatePayload)
	if err != nil {
		return
	}

	beforeGuildMember, _ := ctx.Sandwich.State.GetGuildMember(
		guildMemberUpdatePayload.GuildID, guildMemberUpdatePayload.User.ID)

	ctx.Sandwich.State.SetGuildMember(guildMemberUpdatePayload.GuildID, guildMemberUpdatePayload.GuildMember)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeGuildMember,
		},
	}, true, nil
}

func OnGuildRoleCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildRoleCreatePayload discord.GuildRoleCreate

	err = ctx.decodeContent(msg, &guildRoleCreatePayload)
	if err != nil {
		return
	}

	ctx.Sandwich.State.SetGuildRole(guildRoleCreatePayload.GuildID, guildRoleCreatePayload.Role)

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnGuildRoleUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildRoleUpdatePayload discord.GuildRoleUpdate

	err = ctx.decodeContent(msg, &guildRoleUpdatePayload)
	if err != nil {
		return
	}

	beforeRole, _ := ctx.Sandwich.State.GetGuildRole(
		guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role.ID)

	ctx.Sandwich.State.SetGuildRole(guildRoleUpdatePayload.GuildID, guildRoleUpdatePayload.Role)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeRole,
		},
	}, true, nil
}

func OnGuildRoleDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildRoleDeletePayload discord.GuildRoleDelete

	err = ctx.decodeContent(msg, &guildRoleDeletePayload)
	if err != nil {
		return
	}

	ctx.Sandwich.State.RemoveGuildRole(guildRoleDeletePayload.GuildID, guildRoleDeletePayload.RoleID)

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnIntegrationCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var integrationCreatePayload discord.IntegrationCreate

	err = ctx.decodeContent(msg, &integrationCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnIntegrationUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var integrationUpdatePayload discord.IntegrationUpdate

	err = ctx.decodeContent(msg, &integrationUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnIntegrationDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var integrationDeletePayload discord.IntegrationDelete

	err = ctx.decodeContent(msg, &integrationDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnInteractionCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var interactionCreatePayload discord.InteractionCreate

	err = ctx.decodeContent(msg, &interactionCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnInviteCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var inviteCreatePayload discord.InviteCreate

	err = ctx.decodeContent(msg, &inviteCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnInviteDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var inviteDeletePayload discord.InviteDelete

	err = ctx.decodeContent(msg, &inviteDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageCreatePayload discord.MessageCreate

	err = ctx.decodeContent(msg, &messageCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageUpdatePayload discord.MessageUpdate

	err = ctx.decodeContent(msg, &messageUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageDeletePayload discord.MessageDelete

	err = ctx.decodeContent(msg, &messageDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageDeleteBulk(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageDeleteBulkPayload discord.MessageDeleteBulk

	err = ctx.decodeContent(msg, &messageDeleteBulkPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageReactionAdd(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionAddPayload discord.MessageReactionAdd

	err = ctx.decodeContent(msg, &messageReactionAddPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageReactionRemove(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionRemovePayload discord.MessageReactionRemove

	err = ctx.decodeContent(msg, &messageReactionRemovePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageReactionRemoveAll(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionRemoveAllPayload discord.MessageReactionRemoveAll

	err = ctx.decodeContent(msg, &messageReactionRemoveAllPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnMessageReactionRemoveEmoji(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var messageReactionRemoveEmojiPayload discord.MessageReactionRemoveEmoji

	err = ctx.decodeContent(msg, &messageReactionRemoveEmojiPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnPresenceUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var presenceUpdatePayload discord.PresenceUpdate

	err = ctx.decodeContent(msg, &presenceUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

// TODO: Implement.
func OnStageInstanceCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var stageInstanceCreatePayload discord.StageInstanceCreate

	err = ctx.decodeContent(msg, &stageInstanceCreatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

// TODO: Implement.
func OnStageInstanceUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var stageInstanceUpdatePayload discord.StageInstanceUpdate

	err = ctx.decodeContent(msg, &stageInstanceUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

// TODO: Implement.
func OnStageInstanceDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var stageInstanceDeletePayload discord.StageInstanceDelete

	err = ctx.decodeContent(msg, &stageInstanceDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnTypingStart(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var typingStartPayload discord.TypingStart

	err = ctx.decodeContent(msg, &typingStartPayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnUserUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var userUpdatePayload discord.UserUpdate

	err = ctx.decodeContent(msg, &userUpdatePayload)
	if err != nil {
		return
	}

	beforeUser, _ := ctx.Sandwich.State.GetUser(userUpdatePayload.ID)

	return structs.StateResult{
		Data: msg,
		Extra: map[string]interface{}{
			"before": beforeUser,
		},
	}, true, nil
}

func OnVoiceStateUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var voiceStateUpdatePayload discord.VoiceStateUpdate

	err = ctx.decodeContent(msg, &voiceStateUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnVoiceServerUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var voiceServerUpdatePayload discord.VoiceServerUpdate

	err = ctx.decodeContent(msg, &voiceServerUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnWebhookUpdate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var webhookUpdatePayload discord.WebhookUpdate

	err = ctx.decodeContent(msg, &webhookUpdatePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func OnGuildJoinRequestDelete(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildJoinRequestDeletePayload discord.GuildJoinRequestDelete

	err = ctx.decodeContent(msg, &guildJoinRequestDeletePayload)
	if err != nil {
		return
	}

	return structs.StateResult{
		Data: msg,
	}, true, nil
}

func init() {
	registerDispatch("READY", OnReady)
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
