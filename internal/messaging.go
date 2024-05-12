package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	messaging "github.com/WelcomerTeam/Sandwich-Daemon/messaging"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
)

type MQClient interface {
	String() string
	Channel() string

	Connect(ctx context.Context, clientName string, args map[string]interface{}) error
	Publish(ctx context.Context, channel string, data []byte) error
	// Function to clean close
}

func NewMQClient(mqType string) (MQClient, error) {
	switch mqType {
	case "jetstream":
		return &messaging.JetStreamMQClient{}, nil
	case "kafka":
		return &messaging.KafkaMQClient{}, nil
	case "redis":
		return &messaging.RedisMQClient{}, nil
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

	packet.Trace["publish"] = discord.Int64(time.Now().Unix())

	payload, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = sh.Manager.ProducerClient.Publish(
		ctx,
		channelName,
		payload,
	)

	if err != nil {
		return fmt.Errorf("publishEvent publish: %w", err)
	}

	return nil
}
