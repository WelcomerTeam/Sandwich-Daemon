package mqclients

import (
	"context"
	"strconv"

	"github.com/go-redis/redis/v8"
	"golang.org/x/xerrors"
)

func init() { //nolint
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

func (redisMQ *RedisMQClient) Connect(ctx context.Context, clientName string, args map[string]interface{}) (err error) {
	var ok bool

	var address string

	if address, ok = GetEntry(args, "Address").(string); !ok {
		return xerrors.New("redisMQ connect: string type assertion failed for Address")
	}

	var password string

	if password, ok = GetEntry(args, "Password").(string); !ok {
		return xerrors.New("redisMQ connect: string type assertion failed for Password")
	}

	var db int

	if dbStr, ok := GetEntry(args, "DB").(string); !ok {
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return xerrors.Errorf("redisMQ connect db atoi: %w", err)
		}
	}

	redisMQ.redisClient = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	err = redisMQ.redisClient.Ping(ctx).Err()
	if err != nil {
		return xerrors.Errorf("redisMQ connect ping: %w", err)
	}

	return nil
}

func (redisMQ *RedisMQClient) Publish(ctx context.Context, channelName string, data []byte) (err error) {
	return redisMQ.redisClient.Publish(
		ctx,
		channelName,
		data,
	).Err()
}
