package internal

import (
	"context"

	messaging "github.com/WelcomerTeam/Sandwich-Daemon/next/messaging"
	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"golang.org/x/xerrors"
)

const (
	minPayloadCompressionSize = 1000000 // Apply higher level compression to payloads >1 Mb
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

	sh.ShardGroup.userMu.RLock()
	user := sh.ShardGroup.User
	sh.ShardGroup.userMu.RUnlock()

	packet.Metadata = structs.SandwichMetadata{
		Version:       VERSION,
		Identifier:    identifier,
		Application:   application,
		ApplicationID: int64(user.ID),
		Shard: [3]int{
			int(sh.ShardGroup.ID),
			sh.ShardID,
			sh.ShardGroup.ShardCount,
		},
	}

	payload, err := json.Marshal(packet)
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
