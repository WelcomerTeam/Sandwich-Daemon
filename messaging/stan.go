package mqclients

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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

func (stanMQ *StanMQClient) Connect(ctx context.Context, clientName string, args map[string]interface{}) (err error) {
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

func (stanMQ *StanMQClient) Publish(ctx context.Context, channelName string, data []byte) (err error) {
	if stanMQ.async {
		_, err = stanMQ.StanClient.PublishAsync(
			channelName,
			data,
			nil,
		)

		return
	}

	return stanMQ.StanClient.Publish(
		channelName,
		data,
	)
}
