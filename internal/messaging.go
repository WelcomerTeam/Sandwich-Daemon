package internal

import (
	"context"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/discord/structs"
	messaging "github.com/WelcomerTeam/Sandwich-Daemon/messaging"
	structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/xerrors"
)

type MQClient interface {
	String() string
	Channel() string
	Cluster() string

	Connect(ctx context.Context, clientName string, args map[string]interface{}) (err error)
	Publish(ctx context.Context, channel string, data []byte) (err error)
	// Function to clean close
}

func NewMQClient(mqType string) (MQClient, error) {
	switch mqType {
	case "stan":
		return &messaging.StanMQClient{}, nil
	case "kafka":
		return &messaging.KafkaMQClient{}, nil
	case "redis":
		return &messaging.RedisMQClient{}, nil
	default:
		return nil, xerrors.New("No MQ client named " + mqType)
	}
}

// PublishEvent publishes a SandwichPayload.
func (sh *Shard) PublishEvent(ctx context.Context, packet *structs.SandwichPayload) (err error) {
	sh.Manager.configurationMu.RLock()
	identifier := sh.Manager.Configuration.ProducerIdentifier
	channelName := sh.Manager.Configuration.Messaging.ChannelName
	application := sh.Manager.Identifier.Load()
	sh.Manager.configurationMu.RUnlock()

	userID := sh.Manager.UserID.Load()

	packet.Metadata = structs.SandwichMetadata{
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

	payload, err := jsoniter.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("failed to marshal payload: %w", err)
	}

	err = sh.Manager.ProducerClient.Publish(
		ctx,
		channelName,
		payload,
	)

	if err != nil {
		return xerrors.Errorf("publishEvent publish: %w", err)
	}

	return nil
}
