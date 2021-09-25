package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

const MagicDecimalBase = 10

func gatewayOpDispatch(ctx context.Context, sh *Shard, msg discord.GatewayPayload) error {
	go func(msg discord.GatewayPayload) {
		sh.Sandwich.EventPoolWaiting.Inc()

		ticket := sh.Sandwich.EventPool.Wait()
		defer sh.Sandwich.EventPool.FreeTicket(ticket)

		sh.Sandwich.EventPoolWaiting.Dec()

		err := sh.OnDispatch(ctx, msg)
		if err != nil && !xerrors.Is(err, ErrNoDispatchHandler) {
			sh.Logger.Error().Err(err).Msg("State dispatch failed")
		}
	}(msg)

	return nil
}

func gatewayOpHeartbeat(ctx context.Context, sh *Shard, msg discord.GatewayPayload) (err error) {
	err = sh.SendEvent(ctx, discord.GatewayOpHeartbeat, sh.Sequence.Load())
	if err != nil {
		go sh.Sandwich.PublishSimpleWebhook(
			"Failed to send heartbeat",
			"`"+err.Error()+"`",
			fmt.Sprintf(
				"Manager: %s ShardGroup: %d ShardID: %d/%d",
				sh.Manager.Configuration.Identifier,
				sh.ShardGroup.ID,
				sh.ShardID,
				sh.ShardGroup.ShardCount,
			),
			EmbedColourDanger,
		)

		err = sh.Reconnect(websocket.StatusNormalClosure)
		if err != nil {
			sh.Logger.Error().Err(err).Msg("Failed to reconnect")
		}
	}

	return
}

func gatewayOpReconnect(ctx context.Context, sh *Shard, msg discord.GatewayPayload) (err error) {
	sh.Logger.Info().Msg("Reconnecting in response to gateway")

	err = sh.Reconnect(WebsocketReconnectCloseCode)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to reconnect")
	}

	return
}

func gatewayOpInvalidSession(ctx context.Context, sh *Shard, msg discord.GatewayPayload) (err error) {
	resumable := json.Get(msg.Data, "d").ToBool()
	if !resumable {
		sh.SessionID.Store("")
		sh.Sequence.Store(0)
	}

	sh.Logger.Warn().Bool("resumable", resumable).Msg("Received invalid session")

	go sh.Sandwich.PublishSimpleWebhook(
		"Received invalid session from gateway",
		"",
		fmt.Sprintf(
			"Manager: %s ShardGroup: %d ShardID: %d/%d",
			sh.Manager.Configuration.Identifier,
			sh.ShardGroup.ID,
			sh.ShardID,
			sh.ShardGroup.ShardCount,
		),
		EmbedColourSandwich,
	)

	err = sh.Reconnect(WebsocketReconnectCloseCode)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to reconnect")
	}

	return
}

func gatewayOpHello(ctx context.Context, sh *Shard, msg discord.GatewayPayload) (err error) {
	hello := discord.Hello{}

	err = sh.decodeContent(msg, &hello)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	sh.LastHeartbeatSent.Store(now)
	sh.LastHeartbeatAck.Store(now)

	sh.HeartbeatInterval = hello.HeartbeatInterval * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures
	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Msg("Received HELLO event from discord")

	return
}

func gatewayOpHeartbeatACK(ctx context.Context, sh *Shard, msg discord.GatewayPayload) (err error) {
	sh.LastHeartbeatAck.Store(time.Now().UTC())

	heartbeatRTT := sh.LastHeartbeatAck.Load().Sub(sh.LastHeartbeatSent.Load()).Milliseconds()

	sh.Logger.Debug().
		Int64("RTT", heartbeatRTT).
		Msg("Received heartbeat ACK")

	sandwichGatewayLatency.WithLabelValues(
		sh.Manager.Configuration.Identifier,
		strconv.FormatInt(sh.ShardGroup.ID, MagicDecimalBase),
		strconv.Itoa(sh.ShardID),
	).Set(float64(heartbeatRTT))

	return
}

func init() {
	registerGatewayEvent(discord.GatewayOpDispatch, gatewayOpDispatch)
	registerGatewayEvent(discord.GatewayOpHeartbeat, gatewayOpHeartbeat)
	registerGatewayEvent(discord.GatewayOpReconnect, gatewayOpReconnect)
	registerGatewayEvent(discord.GatewayOpInvalidSession, gatewayOpInvalidSession)
	registerGatewayEvent(discord.GatewayOpHello, gatewayOpHello)
	registerGatewayEvent(discord.GatewayOpHeartbeatACK, gatewayOpHeartbeatACK)
}
