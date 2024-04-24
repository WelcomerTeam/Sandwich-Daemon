package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	"github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

func init() {
	MQClients = append(MQClients, "stan")
}

type StanMQClient struct {
	NatsClient *nats.Conn `json:"-"`
	StanClient stan.Conn  `json:"-"`

	async bool

	channel string
	cluster string
}

func (stanMQ *StanMQClient) String() string {
	return "stan"
}

func (stanMQ *StanMQClient) Channel() string {
	return stanMQ.channel
}

func (stanMQ *StanMQClient) Cluster() string {
	return stanMQ.cluster
}

func (stanMQ *StanMQClient) Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error {
	var ok bool

	var address string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return errors.New("stanMQ connect: string type assertion failed for Address")
	}

	var cluster string

	if cluster, ok = GetEntry(args, "Cluster").(string); !ok {
		return errors.New("stanMQ connect: string type assertion failed for Cluster")
	}

	var channel string

	if channel, ok = GetEntry(args, "Channel").(string); !ok {
		return errors.New("stanMQ connect: string type assertion failed for Channel")
	}

	stanMQ.cluster = cluster
	stanMQ.channel = channel

	var useNatsConnection bool
	var err error

	if useNatsConnectionStr, ok := GetEntry(args, "UseNATSConnection").(string); ok {
		if useNatsConnection, err = strconv.ParseBool(useNatsConnectionStr); err != nil {
			useNatsConnection = true
		}
	} else {
		useNatsConnection = true
	}

	if asyncStr, ok := GetEntry(args, "Async").(string); ok {
		stanMQ.async, _ = strconv.ParseBool(asyncStr)
	} else {
		stanMQ.async = false
	}

	var option stan.Option

	if useNatsConnection {
		stanMQ.NatsClient, err = nats.Connect(address)
		if err != nil {
			return fmt.Errorf("stanMQ connect nats: %w", err)
		}

		option = stan.NatsConn(stanMQ.NatsClient)
	} else {
		option = stan.NatsURL(address)
	}

	stanMQ.StanClient, err = stan.Connect(
		cluster,
		clientName,
		option,
	)
	if err != nil {
		return fmt.Errorf("stanMQ connect stan: %w", err)
	}

	return nil
}

func (stanMQ *StanMQClient) Publish(ctx context.Context, packet *structs.SandwichPayload, channelName string) error {
	data, err := sandwichjson.Marshal(packet)

	if err != nil {
		return err
	}

	if stanMQ.async {
		_, err := stanMQ.StanClient.PublishAsync(
			channelName,
			data,
			nil,
		)

		return err
	}

	return stanMQ.StanClient.Publish(
		channelName,
		data,
	)
}

func (stanMQ *StanMQClient) IsClosed() bool {
	return stanMQ.StanClient == nil
}

func (stanMQ *StanMQClient) CloseShard(shardID int32, reason MQCloseShardReason) {
	// No-op
}

func (stanMQ *StanMQClient) Close() {
	stanMQ.StanClient.Close()
	stanMQ.NatsClient.Close()
	stanMQ.StanClient = nil
	stanMQ.NatsClient = nil
}
