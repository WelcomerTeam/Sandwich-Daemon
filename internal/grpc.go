package gateway

import (
	"context"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
	pb "github.com/TheRockettek/Sandwich-Daemon/protobuf"
	structs "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
	jsoniter "github.com/json-iterator/go"
)

// Creates the gRPC Gateway Server for use in Sandwich Initialization.
func (sg *Sandwich) NewGatewayServer() *RouteGatewayServer {
	return &RouteGatewayServer{
		sg: sg,
	}
}

type RouteGatewayServer struct {
	pb.GatewayServer

	sg *Sandwich
}

// SendEventToGateway is a straighforward gRPC request to send some data
// to a specific shard's gateway connection. This is low level and should
// not be necessary.
func (s *RouteGatewayServer) SendEventToGateway(ctx context.Context, event *pb.SendEventRequest) (*pb.SendEventResponse, error) {
	var err error

	s.sg.ManagersMu.RLock()
	manager, ok := s.sg.Managers[event.Manager]
	s.sg.ManagersMu.RUnlock()

	if ok {
		manager.ShardGroupsMu.RLock()
		shardgroup, ok := manager.ShardGroups[event.ShardGroup]
		manager.ShardGroupsMu.RUnlock()

		if ok {
			shardgroup.ShardsMu.RLock()
			shard, ok := shardgroup.Shards[int(event.ShardID)]
			shardgroup.ShardsMu.RUnlock()

			if ok {
				err = shard.SendEvent(
					structs.GatewayOp(event.GatewayOPCode),
					jsoniter.RawMessage(event.Data),
				)

				return &pb.SendEventResponse{
					FoundShard: true,
					Error:      ReturnError(err),
				}, err
			}

			err = ErrInvalidShard
		} else {
			err = ErrInvalidShardGroup
		}
	} else {
		err = ErrInvalidManager
	}

	return &pb.SendEventResponse{
		FoundShard: false,
		Error:      ReturnError(err),
	}, err
}

// RequestGuildChunks is a gRPC request which yields until
// the guild id specified has chunked and will wait if multiple chunk
// requests on the same guild is in progress. This is disabled with
// Wait.
func (s *RouteGatewayServer) RequestGuildChunks(ctx context.Context, event *pb.RequestGuildChunksRequest) (*pb.StandardResponse, error) {
	var err error

	s.sg.ManagersMu.RLock()
	manager, ok := s.sg.Managers[event.Manager]
	s.sg.ManagersMu.RUnlock()

	if ok {
		manager.ShardGroupsMu.RLock()
		shardgroup, ok := manager.ShardGroups[event.ShardGroup]
		manager.ShardGroupsMu.RUnlock()

		if ok {
			shardgroup.ShardsMu.RLock()
			shard, ok := shardgroup.Shards[(int(event.GuildID)>>22)%shardgroup.ShardCount]
			shardgroup.ShardsMu.RUnlock()

			if ok {
				err = shard.ChunkGuild(snowflake.ParseInt64(event.GuildID), event.Wait)

				return &pb.StandardResponse{
					Success: err == nil,
					Error:   ReturnError(err),
				}, err
			}

			err = ErrInvalidShard
		} else {
			err = ErrInvalidShardGroup
		}
	} else {
		err = ErrInvalidManager
	}

	return &pb.StandardResponse{
		Success: false,
		Error:   ReturnError(err),
	}, err
}
