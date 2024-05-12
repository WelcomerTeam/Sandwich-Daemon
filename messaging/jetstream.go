package mqclients

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func init() {
	MQClients = append(MQClients, "jetstream")
}

type JetStreamMQClient struct {
	JetStreamClient jetstream.JetStream `json:"-"`

	channel string
	cluster string
}

func (jetstreamMQ *JetStreamMQClient) String() string {
	return "jetstream"
}

func (jetstreamMQ *JetStreamMQClient) Channel() string {
	return jetstreamMQ.channel
}

func (jetstreamMQ *JetStreamMQClient) Cluster() string {
	return jetstreamMQ.cluster
}

func (jetstreamMQ *JetStreamMQClient) Connect(ctx context.Context, clientName string, args map[string]interface{}) error {
	var ok bool

	var address string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return errors.New("jetstreamMQ connect: string type assertion failed for Address")
	}

	var cluster string

	if cluster, ok = GetEntry(args, "Cluster").(string); !ok {
		return errors.New("jetstreamMQ connect: string type assertion failed for Cluster")
	}

	var channel string

	if channel, ok = GetEntry(args, "Channel").(string); !ok {
		return errors.New("jetstreamMQ connect: string type assertion failed for Channel")
	}

	jetstreamMQ.cluster = cluster
	jetstreamMQ.channel = channel

	nc, err := nats.Connect(address)
	if err != nil {
		return fmt.Errorf("jetstreamMQ connect nats: %w", err)
	}

	jetstreamMQ.JetStreamClient, err = jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("jetstreamMQ new: %w", err)
	}

	return nil
}

func (jetstreamMQ *JetStreamMQClient) Publish(ctx context.Context, channelName string, data []byte) error {
	_, err := jetstreamMQ.JetStreamClient.PublishAsync(
		channelName,
		data,
		nil,
	)

	return err
}
