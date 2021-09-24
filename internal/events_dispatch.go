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
		ctx.Logger.Error().Err(err).Msg("Failed to decode READY event")

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
					ctx.Logger.Error().Err(err).Msg("Failed to decode GUILD_CREATE event")
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

func OnGuildCreate(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	var guildCreatePayload discord.GuildCreate

	err = ctx.decodeContent(msg, &guildCreatePayload)
	if err != nil {
		ctx.Logger.Error().Err(err).Msg("Failed to decode GUILD_CREATE event")

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
		ctx.Logger.Error().Err(err).Msg("Failed to decode GUILD_MEMBERS_CHUNK event")

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

func init() {
	registerDispatch("READY", OnReady)
	registerDispatch("GUILD_CREATE", OnGuildCreate)
	registerDispatch("GUILD_MEMBERS_CHUNK", OnGuildMembersChunk)
}
