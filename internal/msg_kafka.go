package internal

import (
	"context"
	"errors"
	"strconv"

	"github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
	"github.com/segmentio/kafka-go"
)

func init() {
	MQClients = append(MQClients, "kafka")
}

type KafkaMQClient struct {
	KafkaClient *kafka.Writer

	channel string
	cluster string
}

func parseKafkaBalancer(balancer string) kafka.Balancer {
	switch balancer {
	case "crc32":
		return &kafka.CRC32Balancer{}
	case "hash":
		return &kafka.Hash{}
	case "murmur2":
		return &kafka.Murmur2Balancer{}
	case "roundrobin":
		return &kafka.RoundRobin{}
	case "leastbytes":
		return &kafka.LeastBytes{}
	default:
		return nil
	}
}

func (kafkaMQ *KafkaMQClient) String() string {
	return "kafka"
}

func (kafkaMQ *KafkaMQClient) Channel() string {
	return kafkaMQ.channel
}

func (kafkaMQ *KafkaMQClient) Cluster() string {
	return kafkaMQ.cluster
}

func (kafkaMQ *KafkaMQClient) Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error {
	var ok bool

	var address string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return errors.New("kafkaMQ connect: string type assertion failed for Address")
	}

	var balancer kafka.Balancer

	if balancerStr, ok := GetEntry(args, "Balancer").(string); ok {
		balancer = parseKafkaBalancer(balancerStr)
	} else {
		return errors.New("kafkaMQ connect: string type assertion failed for Balancer")
	}

	var async bool

	if asyncStr, ok := GetEntry(args, "Async").(string); ok {
		async, _ = strconv.ParseBool(asyncStr)
	} else {
		async = false
	}

	kafkaMQ.KafkaClient = &kafka.Writer{
		Addr:     kafka.TCP(address),
		Balancer: balancer,
		Async:    async,
	}

	return nil
}

func (kafkaMQ *KafkaMQClient) Publish(ctx context.Context, packet *structs.SandwichPayload, channelName string) error {
	data, err := jsoniter.Marshal(packet)

	if err != nil {
		return err
	}

	return kafkaMQ.KafkaClient.WriteMessages(
		ctx,
		kafka.Message{
			Topic: channelName,
			Value: data,
		},
	)
}

func (kafkaMQ *KafkaMQClient) IsClosed() bool {
	return kafkaMQ.KafkaClient == nil
}

func (kafkaMQ *KafkaMQClient) CloseShard(shardID int32) {
	// No-op for kafka
}

func (kafkaMQ *KafkaMQClient) Close() {
	kafkaMQ.KafkaClient.Close()
	kafkaMQ.KafkaClient = nil
}
