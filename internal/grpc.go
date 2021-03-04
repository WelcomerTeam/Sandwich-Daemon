package gateway

import (
	"context"

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
