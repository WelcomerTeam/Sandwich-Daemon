package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/coder/websocket"
)

const MagicDecimalBase = 10

func gatewayOpDispatch(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
	sh.Sequence.Store(msg.Sequence)

	trace["dispatch"] = discord.Int64(time.Now().Unix())

	sh.Sandwich.EventsInflight.Inc()
	defer sh.Sandwich.EventsInflight.Dec()

	err := sh.OnDispatch(ctx, msg, trace)
	if err != nil && !errors.Is(err, ErrNoDispatchHandler) {
		sh.Logger.Error().Err(err).Msg("State dispatch failed")
	}

	return nil
}

func gatewayOpHeartbeat(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
	err := sh.SendEvent(ctx, discord.GatewayOpHeartbeat, sh.Sequence.Load())
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

			return err
		}
	}

	return nil
}

func gatewayOpReconnect(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
	sh.Logger.Info().Msg("Reconnecting in response to gateway")

	err := sh.Reconnect(WebsocketReconnectCloseCode)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to reconnect")

		return err
	}

	return nil
}

func gatewayOpInvalidSession(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
	var resumable bool

	sh.Logger.Warn().Str("data", string(msg.Data)).Msg("Received invalid session")

	err := json.Unmarshal(msg.Data, &resumable)
	if err != nil {
		sh.Logger.Error().Err(err).Msg("Failed to unmarshal invalid session")

		return err
	}

	sh.Logger.Info().Bool("resumable", resumable).Msg("Invalid session received")

	if !resumable {
		sh.SessionID.Store("")
		sh.Sequence.Store(0)
	}

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

		return err
	}

	return nil
}

func gatewayOpHello(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
	var hello discord.Hello

	err := sh.decodeContent(msg, &hello)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	sh.LastHeartbeatSent.Store(now)
	sh.LastHeartbeatAck.Store(now)

	sh.Logger.Debug().
		Int32("interval", hello.HeartbeatInterval).
		Msg("Received HELLO event from discord")

	if hello.HeartbeatInterval <= 0 {
		sh.Logger.Error().
			Int32("interval", hello.HeartbeatInterval).
			Str("event_type", msg.Type).
			Str("event_data", string(msg.Data)).
			Msg("Invalid heartbeat interval")

		return ErrInvalidHeartbeatInterval
	}

	sh.HeartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
	sh.HeartbeatFailureInterval = sh.HeartbeatInterval * ShardMaxHeartbeatFailures
	sh.Heartbeater.Reset(sh.HeartbeatInterval)

	return nil
}

func gatewayOpHeartbeatACK(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error {
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

	return nil
}

func init() {
	registerGatewayEvent(discord.GatewayOpDispatch, gatewayOpDispatch)
	registerGatewayEvent(discord.GatewayOpHeartbeat, gatewayOpHeartbeat)
	registerGatewayEvent(discord.GatewayOpReconnect, gatewayOpReconnect)
	registerGatewayEvent(discord.GatewayOpInvalidSession, gatewayOpInvalidSession)
	registerGatewayEvent(discord.GatewayOpHello, gatewayOpHello)
	registerGatewayEvent(discord.GatewayOpHeartbeatACK, gatewayOpHeartbeatACK)
}
