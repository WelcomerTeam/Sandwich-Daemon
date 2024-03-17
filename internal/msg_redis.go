package internal

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/WelcomerTeam/Sandwich-Daemon/structs"
	"github.com/go-redis/redis/v8"
)

func init() {
	MQClients = append(MQClients, "redis")
}

type RedisMQClient struct {
	redisClient *redis.Client

	channel string
	cluster string
}

func (redisMQ *RedisMQClient) String() string {
	return "redis"
}

func (redisMQ *RedisMQClient) Channel() string {
	return redisMQ.channel
}

func (redisMQ *RedisMQClient) Cluster() string {
	return redisMQ.cluster
}

func (redisMQ *RedisMQClient) Connect(ctx context.Context, manager *Manager, clientName string, args map[string]interface{}) error {
	var ok bool

	var address string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return errors.New("redisMQ connect: string type assertion failed for Address")
	}

	var password string

	if password, ok = GetEntry(args, "Password").(string); !ok {
		return errors.New("redisMQ connect: string type assertion failed for Password")
	}

	var db int
	var err error

	if dbStr, ok := GetEntry(args, "DB").(string); !ok {
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return fmt.Errorf("redisMQ connect db atoi: %w", err)
		}
	}

	redisMQ.redisClient = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	err = redisMQ.redisClient.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("redisMQ connect ping: %w", err)
	}

	return nil
}

func (redisMQ *RedisMQClient) Publish(ctx context.Context, packet *structs.SandwichPayload, channelName string, data []byte) error {
	return redisMQ.redisClient.Publish(
		ctx,
		channelName,
		data,
	).Err()
}

func (redisMQ *RedisMQClient) IsClosed() bool {
	return redisMQ.redisClient == nil
}

func (redisMQ *RedisMQClient) CloseShard(shardID int32) {
	// No-op
}

func (redisMQ *RedisMQClient) Close() {
	redisMQ.redisClient.Close()
	redisMQ.redisClient = nil
}
