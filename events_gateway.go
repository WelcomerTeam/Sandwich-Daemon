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

func GatewayOpDispatch(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, trace *Trace) error {
	shard.sequence.Store(msg.Sequence)

	trace.Set("dispatch", time.Now().UnixNano())

	return shard.OnDispatch(ctx, msg, trace)
}

func GatewayOpHeartbeat(ctx context.Context, shard *Shard, _ *discord.GatewayPayload, _ *Trace) error {
	err := shard.SendEvent(ctx, discord.GatewayOpHeartbeat, shard.sequence.Load())
	if err != nil {
		err = shard.reconnect(ctx, websocket.StatusNormalClosure)
		if err != nil {
			return fmt.Errorf("failed to reconnect due to heartbeat failure: %w", err)
		}
	}

	return nil
}

func GatewayOpReconnect(ctx context.Context, shard *Shard, _ *discord.GatewayPayload, _ *Trace) error {
	shard.Logger.Debug("Shard has been requested to reconnect")

	err := shard.reconnect(ctx, WebsocketReconnectCloseCode)
	if err != nil {
		return fmt.Errorf("failed to reconnect due to reconnect event: %w", err)
	}

	return nil
}

func GatewayOpInvalidSession(ctx context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) error {
	var resumable bool

	err := json.Unmarshal(msg.Data, &resumable)
	if err != nil {
		return fmt.Errorf("failed to unmarshal invalid session: %w", err)
	}

	shard.Logger.Warn("Shard has received an invalid session", "resumable", resumable)

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

func GatewayOpHello(_ context.Context, shard *Shard, msg *discord.GatewayPayload, _ *Trace) error {
	var hello discord.Hello

	err := json.Unmarshal(msg.Data, &hello)
	if err != nil {
		return fmt.Errorf("failed to unmarshal hello: %w", err)
	}

	now := time.Now()
	shard.LastHeartbeatSent.Store(&now)
	shard.LastHeartbeatAck.Store(&now)

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

func GatewayOpHeartbeatAck(_ context.Context, shard *Shard, _ *discord.GatewayPayload, _ *Trace) error {
	now := time.Now()
	shard.LastHeartbeatAck.Store(&now)

	if lastHeartbeatSent := shard.LastHeartbeatSent.Load(); lastHeartbeatSent != nil {
		gatewayLatency := now.Sub(*lastHeartbeatSent).Milliseconds()
		shard.GatewayLatency.Store(gatewayLatency)
		UpdateGatewayLatency(shard.Application.Identifier, shard.ShardID, float64(gatewayLatency))
	}

	return nil
}

func init() {
	RegisterGatewayEvent(discord.GatewayOpDispatch, GatewayOpDispatch)
	RegisterGatewayEvent(discord.GatewayOpHeartbeat, GatewayOpHeartbeat)
	RegisterGatewayEvent(discord.GatewayOpReconnect, GatewayOpReconnect)
	RegisterGatewayEvent(discord.GatewayOpInvalidSession, GatewayOpInvalidSession)
	RegisterGatewayEvent(discord.GatewayOpHello, GatewayOpHello)
	RegisterGatewayEvent(discord.GatewayOpHeartbeatACK, GatewayOpHeartbeatAck)
}
