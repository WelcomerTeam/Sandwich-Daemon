package internal

import (
	"context"
	"fmt"
	discord_structs "github.com/WelcomerTeam/Discord/structs"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
	"strconv"
	"time"
)

const MagicDecimalBase = 10

func gatewayOpDispatch(ctx context.Context, sh *Shard, msg discord_structs.GatewayPayload) error {
	sh.Sequence.Store(msg.Sequence)

	go func(msg discord_structs.GatewayPayload) {
		sh.Sandwich.EventsInflight.Inc()
		defer sh.Sandwich.EventsInflight.Dec()

		err := sh.OnDispatch(ctx, msg)
		if err != nil && !xerrors.Is(err, ErrNoDispatchHandler) {
			sh.Logger.Error().Err(err).Msg("State dispatch failed")
		}
	}(msg)

	return nil
}

func gatewayOpHeartbeat(ctx context.Context, sh *Shard, msg discord_structs.GatewayPayload) (err error) {
	err = sh.SendEvent(ctx, discord_structs.GatewayOpHeartbeat, sh.Sequence.Load())
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

func gatewayOpReconnect(ctx context.Context, sh *Shard, msg discord_structs.GatewayPayload) (err error) {
	sh.Logger.Info().Msg("Reconnecting in response to gateway")

	err = sh.Reconnect(WebsocketReconnectCloseCode)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to reconnect")
	}

	return
}

func gatewayOpInvalidSession(ctx context.Context, sh *Shard, msg discord_structs.GatewayPayload) (err error) {
	resumable := jsoniter.Get(msg.Data, "d").ToBool()
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

func gatewayOpHello(ctx context.Context, sh *Shard, msg discord_structs.GatewayPayload) (err error) {
	hello := discord_structs.Hello{}

	err = sh.decodeContent(msg, &hello)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	sh.LastHeartbeatSent.Store(now)
	sh.LastHeartbeatAck.Store(now)

	sh.HeartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures
	sh.Heartbeater = time.NewTicker(sh.HeartbeatInterval)

	sh.Logger.Debug().
		Dur("interval", sh.HeartbeatInterval).
		Msg("Received HELLO event from discord")

	return
}

func gatewayOpHeartbeatACK(ctx context.Context, sh *Shard, msg discord_structs.GatewayPayload) (err error) {
	sh.LastHeartbeatAck.Store(time.Now().UTC())

	heartbeatRTT := sh.LastHeartbeatAck.Load().Sub(sh.LastHeartbeatSent.Load()).Milliseconds()

	sh.Logger.Debug().
		Int64("RTT", heartbeatRTT).
		Msg("Received heartbeat ACK")

	sandwichGatewayLatency.WithLabelValues(
		sh.Manager.Identifier.Load(),
		strconv.FormatInt(int64(sh.ShardGroup.ID), MagicDecimalBase),
		strconv.Itoa(int(sh.ShardID)),
	).Set(float64(heartbeatRTT))

	return
}

func init() {
	registerGatewayEvent(discord_structs.GatewayOpDispatch, gatewayOpDispatch)
	registerGatewayEvent(discord_structs.GatewayOpHeartbeat, gatewayOpHeartbeat)
	registerGatewayEvent(discord_structs.GatewayOpReconnect, gatewayOpReconnect)
	registerGatewayEvent(discord_structs.GatewayOpInvalidSession, gatewayOpInvalidSession)
	registerGatewayEvent(discord_structs.GatewayOpHello, gatewayOpHello)
	registerGatewayEvent(discord_structs.GatewayOpHeartbeatACK, gatewayOpHeartbeatACK)
}
