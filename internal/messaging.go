package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
)

type MQCloseShardReason int

const (
	// CloseShardReasonGateway means that the gateway connection was closed.
	MQCloseShardReasonGateway MQCloseShardReason = iota
	MQCloseShardReasonOther   MQCloseShardReason = iota
)

type MQClient interface {
	String() string
	Channel() string

	Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error
	Publish(ctx context.Context, packet *sandwich_structs.SandwichPayload, channel string) error

	// IsClosed returns true if the connection is closed.
	IsClosed() bool

	// Close all connections for a specific shard, only supported by websocket producer
	CloseShard(shardID int32, reason MQCloseShardReason)

	// Stop a session id, only supported by websocket producer
	StopSession(sessionID string)

	// Close the connection
	Close()
}

func NewMQClient(mqType string) (MQClient, error) {
	switch mqType {
	case "jetstream":
		return &JetStreamMQClient{}, nil
	case "kafka":
		return &KafkaMQClient{}, nil
	case "redis":
		return &RedisMQClient{}, nil
	case "websocket":
		return &WebsocketClient{}, nil
	default:
		return nil, fmt.Errorf("%s is not a valid MQClient", mqType)
	}
}

// PublishEvent publishes a SandwichPayload.
func (sh *Shard) PublishEvent(ctx context.Context, packet *sandwich_structs.SandwichPayload) error {
	sh.Manager.configurationMu.RLock()
	channelName := sh.Manager.Configuration.Messaging.ChannelName
	disableTrace := sh.Manager.Configuration.DisableTrace
	sh.Manager.configurationMu.RUnlock()

	packet.Metadata = sh.metadata

	if !disableTrace {
		if packet.Trace == nil {
			packet.Trace = make(map[string]discord.Int64)
		}

		packet.Trace["publish"] = discord.Int64(time.Now().Unix())
	}

	err := sh.Manager.RoutePayloadToConsumer(packet)

	if err != nil {
		return fmt.Errorf("publishEvent RoutePayloadToConsumer: %w", err)
	}

	err = sh.Manager.ProducerClient.Publish(
		ctx,
		packet,
		channelName,
	)
	if err != nil {
		return fmt.Errorf("publishEvent publish: %w", err)
	}

	return nil
}
