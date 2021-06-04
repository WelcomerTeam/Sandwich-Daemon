package gateway

import (
	"context"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	discord "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
	"golang.org/x/xerrors"
)

// StateReady handles the READY event.
func StateReady(ctx *StateCtx, msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	var packet discord.Ready

	var guildPayload discord.GuildCreate

	err = json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return result, false, xerrors.Errorf("Failed to unmarshal message: %w", err)
	}

	ctx.Sh.Logger.Info().Msg("Received READY payload")

	ctx.Sh.Lock()
	ctx.Sh.sessionID = packet.SessionID
	ctx.Sh.User = packet.User
	ctx.Sh.Unlock()

	events := make([]discord.ReceivedPayload, 0)
	guildIDs := make([]snowflake.ID, 0, len(packet.Guilds))

	ctx.Sh.UnavailableMu.Lock()
	ctx.Sh.Unavailable = make(map[snowflake.ID]bool)

	for _, guild := range packet.Guilds {
		ctx.Sh.Unavailable[guild.ID] = guild.Unavailable
	}
	ctx.Sh.UnavailableMu.Unlock()

	guildCreateEvents := 0

	// If true will only run events once finished loading.
	// TODO: Add to sandwich configuration.
	preemptiveEvents := false

	t := time.NewTicker(timeoutDuration)

ready:
	for {
		select {
		case err := <-ctx.Sh.ErrorCh:
			if !xerrors.Is(err, context.Canceled) {
				ctx.Sh.Logger.Error().Err(err).Msg("Encountered error whilst waiting lazy loading")
			}

			break ready
		case msg := <-ctx.Sh.MessageCh:
			if msg.Type == "GUILD_CREATE" {
				guildCreateEvents++

				if err = ctx.Sh.decodeContent(msg, &guildPayload); err != nil {
					ctx.Sh.Logger.Error().Err(err).Msg("Failed to unmarshal GUILD_CREATE")
				} else {
					guildIDs = append(guildIDs, guildPayload.ID)
				}

				t.Reset(timeoutDuration)
			}

			if preemptiveEvents {
				events = append(events, msg)
			} else if err = ctx.Sh.OnDispatch(msg); err != nil && !xerrors.Is(err, NoHandler) {
				ctx.Sh.Logger.Error().Err(err).Msg("Failed dispatching event")
			}
		case <-t.C:
			ctx.Sh.Manager.Sandwich.ConfigurationMu.RLock()

			if ctx.Sh.Manager.Configuration.Caching.RequestMembers {
				for _, guildID := range guildIDs {
					ticket := ctx.Sh.ShardGroup.ChunkLimiter.Wait()

					go func(guildID snowflake.ID, ticket int) {
						err := ctx.Sh.ChunkGuild(guildID, true)
						if err != nil {
							ctx.Sh.Logger.Error().Err(err).Msgf("Failed to request guild members")
						}

						ctx.Sh.ShardGroup.ChunkLimiter.FreeTicket(ticket)
					}(guildID, ticket)
				}
			}
			ctx.Sh.Manager.Sandwich.ConfigurationMu.RUnlock()

			break ready
		}
	}

	ctx.Sh.ready <- void{}
	if err := ctx.Sh.SetStatus(structs.ShardReady); err != nil {
		ctx.Sh.Logger.Error().Err(err).Msg("Encountered error setting shard status")
	}

	if preemptiveEvents {
		ctx.Sh.Logger.Debug().Int("events", len(events)).Msg("Dispatching preemptive events")

		for _, event := range events {
			ctx.Sh.Logger.Debug().Str("type", event.Type).Send()

			if err = ctx.Sh.OnDispatch(event); err != nil {
				ctx.Sh.Logger.Error().Err(err).Msg("Failed whilst dispatching preemptive events")
			}
		}

		ctx.Sh.Logger.Debug().Msg("Finished dispatching events")
	}

	return result, false, nil
}
