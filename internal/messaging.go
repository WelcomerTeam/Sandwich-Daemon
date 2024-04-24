package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

type MQClient interface {
	String() string
	Channel() string
	Cluster() string

	Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error
	Publish(ctx context.Context, packet *sandwich_structs.SandwichPayload, channel string) error

	// IsClosed returns true if the connection is closed.
	IsClosed() bool

	// Close all connections for a specific shard, only supported by websocket producer
	CloseShard(shardID int32)

	// Close the connection
	Close()
}

func NewMQClient(mqType string) (MQClient, error) {
	switch mqType {
	case "stan":
		return &StanMQClient{}, nil
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
	identifier := sh.Manager.Configuration.ProducerIdentifier
	channelName := sh.Manager.Configuration.Messaging.ChannelName
	application := sh.Manager.Identifier.Load()
	sh.Manager.configurationMu.RUnlock()

	userID := sh.Manager.UserID.Load()

	packet.Metadata = sandwich_structs.SandwichMetadata{
		Version:       VERSION,
		Identifier:    identifier,
		Application:   application,
		ApplicationID: discord.Snowflake(userID),
		Shard: [3]int32{
			sh.ShardGroup.ID,
			sh.ShardID,
			sh.ShardGroup.ShardCount,
		},
	}

	if packet.Trace == nil {
		packet.Trace = csmap.Create(
			csmap.WithSize[string, discord.Int64](uint64(1)),
		)
	}

	packet.Trace.Store("publish", discord.Int64(time.Now().Unix()))

	err := sh.Manager.ProducerClient.Publish(
		ctx,
		packet,
		channelName,
	)
	if err != nil {
		return fmt.Errorf("publishEvent publish: %w", err)
	}

	return nil
}
