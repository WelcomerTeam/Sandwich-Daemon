package sandwich

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	"github.com/coder/websocket"
)

const (
	WebsocketReconnectCloseCode = 4000
)

type GatewayHandler func(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) error

var gatewayEvents = make(map[discord.GatewayOp]GatewayHandler)

func RegisterGatewayEvent(eventType discord.GatewayOp, handler GatewayHandler) {
	gatewayEvents[eventType] = handler
}

func gatewayOpDispatch(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) error {
	shard.sequence.Store(msg.Sequence)

	trace.Set("dispatch", time.Now().UnixNano())

	return shard.OnDispatch(ctx, msg, trace)
}

func gatewayOpHeartbeat(ctx context.Context, shard *Shard, _ *discord.GatewayPayload, _ *Trace) error {
	err := shard.SendEvent(ctx, discord.GatewayOpHeartbeat, shard.sequence.Load())
	if err != nil {
		err = shard.reconnect(ctx, websocket.StatusNormalClosure)
		if err != nil {
			return fmt.Errorf("failed to reconnect due to heartbeat failure: %w", err)
		}
	}

	return nil
}

func gatewayOpReconnect(ctx context.Context, shard *Shard, _ *discord.GatewayPayload, _ *Trace) error {
	shard.logger.Debug("Shard has been requested to reconnect")

	err := shard.reconnect(ctx, WebsocketReconnectCloseCode)
	if err != nil {
		return fmt.Errorf("failed to reconnect due to reconnect event: %w", err)
	}

	return nil
}

func gatewayOpInvalidSession(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) error {
	var resumable bool

	err := json.Unmarshal(msg.Data, &resumable)
	if err != nil {
		return fmt.Errorf("failed to unmarshal invalid session: %w", err)
	}

	shard.logger.Warn("Shard has received an invalid session", "resumable", resumable)

	if !resumable {
		shard.sessionID.Store(nil)
		shard.sequence.Store(0)
	}

	err = shard.reconnect(ctx, WebsocketReconnectCloseCode)
	if err != nil {
		return fmt.Errorf("failed to reconnect due to invalid session: %w", err)
	}

	return nil
}

func gatewayOpHello(_ context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) error {
	var hello discord.Hello

	err := json.Unmarshal(msg.Data, &hello)
	if err != nil {
		return fmt.Errorf("failed to unmarshal hello: %w", err)
	}

	now := time.Now()
	shard.lastHeartbeatSent.Store(&now)
	shard.lastHeartbeatAck.Store(&now)

	if hello.HeartbeatInterval <= 0 {
		return ErrShardInvalidHeartbeatInterval
	}

	heartbeatInterval := time.Duration(hello.HeartbeatInterval) * time.Millisecond
	shard.heartbeatInterval.Store(&heartbeatInterval)

	heartbeatFailureInterval := heartbeatInterval * time.Duration(ShardMaxHeartbeatFailures)
	shard.heartbeatFailureInterval.Store(&heartbeatFailureInterval)

	shard.heartbeater.Reset(heartbeatInterval)

	return nil
}

func gatewayOpHeartbeatAck(_ context.Context, shard *Shard, _ *discord.GatewayPayload, _ *Trace) error {
	now := time.Now()
	shard.lastHeartbeatAck.Store(&now)

	if lastHeartbeatSent := shard.lastHeartbeatSent.Load(); lastHeartbeatSent != nil {
		UpdateGatewayLatency(
			shard.application.identifier,
			float64(now.Sub(*lastHeartbeatSent).Milliseconds()),
		)
	}

	return nil
}

func init() {
	RegisterGatewayEvent(discord.GatewayOpDispatch, gatewayOpDispatch)
	RegisterGatewayEvent(discord.GatewayOpHeartbeat, gatewayOpHeartbeat)
	RegisterGatewayEvent(discord.GatewayOpReconnect, gatewayOpReconnect)
	RegisterGatewayEvent(discord.GatewayOpInvalidSession, gatewayOpInvalidSession)
	RegisterGatewayEvent(discord.GatewayOpHello, gatewayOpHello)
	RegisterGatewayEvent(discord.GatewayOpHeartbeatACK, gatewayOpHeartbeatAck)
}
