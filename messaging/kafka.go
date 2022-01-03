package mqclients

import (
	"context"
	"github.com/segmentio/kafka-go"
	"golang.org/x/xerrors"
	"strconv"
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

func (kafkaMQ *KafkaMQClient) Connect(ctx context.Context, clientName string, args map[string]interface{}) (err error) {
	var ok bool

	var address string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return xerrors.New("kafkaMQ connect: string type assertion failed for Address")
	}

	var balancer kafka.Balancer

	if balancerStr, ok := GetEntry(args, "Balancer").(string); ok {
		balancer = parseKafkaBalancer(balancerStr)
	} else {
		return xerrors.New("kafkaMQ connect: string type assertion failed for Balancer")
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

func (kafkaMQ *KafkaMQClient) Publish(ctx context.Context, channelName string, data []byte) (err error) {
	return kafkaMQ.KafkaClient.WriteMessages(
		ctx,
		kafka.Message{
			Topic: channelName,
			Value: data,
		},
	)
}
