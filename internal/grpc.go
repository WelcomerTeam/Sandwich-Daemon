package gateway

import (
	"context"

	pb "github.com/TheRockettek/Sandwich-Daemon/protobuf"
	structs "github.com/TheRockettek/Sandwich-Daemon/structs/discord"
)

type gRPCServer struct {
	pb.GatewayServer
	sw *Sandwich
}

func (s *gRPCServer) SendEvent(ctx context.Context, event *pb.SendEventRequest) (*pb.SendEventResponse, error) {
	var err error

	s.sw.ManagersMu.RLock()
	manager, ok := s.sw.Managers[event.Manager]
	s.sw.ManagersMu.RUnlock()

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
					event.Data,
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
