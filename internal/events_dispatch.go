package internal

import (
	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
)

func tempReady(ctx *StateCtx, msg discord.GatewayPayload) (result structs.StateResult, ok bool, err error) {
	ctx.ready <- void{}

	ctx.Logger.Info().Msg("TEMP READY!")

	return structs.StateResult{
		Data: 1,
	}, true, nil
}

func init() {
	registerDispatch("READY", tempReady)
}
