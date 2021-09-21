package internal

import (
	"context"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"golang.org/x/xerrors"
)

func tempReady(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
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

func init() {
	registerDispatch("READY", tempReady)
}
