package gateway

import (
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	discord "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
	"golang.org/x/xerrors"
)

// StateGuildCreate handles the GUILD_CREATE event.
func StateGuildCreate(ctx *StateCtx, msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	var packet discord.GuildCreate

	err = json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return result, false, xerrors.Errorf("Failed to unmarshal message: %w", err)
	}

	ctx.Sg.State.AddGuildShardGroup(ctx, &packet.Guild)

	lazy, _ := ctx.Vars["lazy"].(bool)

	ctx.Sh.UnavailableMu.RLock()
	ok, unavailable := ctx.Sh.Unavailable[packet.Guild.ID]
	ctx.Sh.UnavailableMu.RUnlock()

	// Check if the guild is unavailable.
	if ok {
		if unavailable {
			ctx.Sh.Logger.Trace().Str("id", packet.Guild.ID.String()).Msg("Lazy loaded guild")

			lazy = true || lazy
		}

		ctx.Sh.UnavailableMu.Lock()
		delete(ctx.Sh.Unavailable, packet.Guild.ID)
		ctx.Sh.UnavailableMu.Unlock()
	}

	return structs.StateResult{
		Data: packet.Guild,
		Extra: map[string]interface{}{
			"lazy": lazy,
		},
	}, true, nil
}

// StateGuildMembersChunk handles the GUILD_MEMBERS_CHUNK event.
func StateGuildMembersChunk(ctx *StateCtx, msg discord.ReceivedPayload) (result structs.StateResult, ok bool, err error) {
	if !ctx.Mg.Configuration.Caching.CacheMembers {
		return
	}

	var packet discord.GuildMembersChunk

	err = json.Unmarshal(msg.Data, &packet)
	if err != nil {
		return result, false, xerrors.Errorf("Failed to unmarshal message: %w", err)
	}

	ctx.Sh.ShardGroup.MemberChunkCallbacksMu.RLock()
	callback, ok := ctx.Sh.ShardGroup.MemberChunkCallbacks[packet.GuildID]

	if ok {
		callback <- true
	}
	ctx.Sh.ShardGroup.MemberChunkCallbacksMu.RUnlock()

	g, o := ctx.Sg.State.GetGuild(ctx, packet.GuildID, false)
	if !o {
		ctx.Sh.Logger.Warn().Msgf("StateGuildMembersChunk referenced guild ID %d that was not in state", packet.GuildID)
		return
	}

	for _, member := range packet.Members {
		ctx.Sg.State.AddMember(ctx, g, member)
	}

	// TODO: Handle storing mutuals
	// TODO: Handle Caching Users

	// We do not want to send member chunks to
	// consumers as they will have no use.

	return result, false, nil
}
