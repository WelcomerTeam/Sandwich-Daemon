package gateway

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/rs/zerolog"
	"golang.org/x/net/context"
	"golang.org/x/xerrors"
)

// ShardGroup groups a selection of shards
type ShardGroup struct {
	StatusMu sync.RWMutex
	Status   structs.ShardGroupStatus

	Manager *Manager
	Logger  zerolog.Logger

	Events *int64

	ShardCount int
	ShardIDs   []int
	Shards     map[int]*Shard

	// WaitGroup for detecting when all shards are ready
	Wait *sync.WaitGroup

	// Used for detecting errors during shard startup
	err chan error
	// Used to close active goroutines
	close chan void
}

// NewShardGroup creates a new shardgroup
func (mg *Manager) NewShardGroup() *ShardGroup {
	return &ShardGroup{
		Status:   structs.ShardGroupIdle,
		StatusMu: sync.RWMutex{},

		Manager: mg,
		Logger:  mg.Logger,

		Events: new(int64),

		Shards: make(map[int]*Shard),
		Wait:   &sync.WaitGroup{},
		err:    make(chan error),
		close:  make(chan void),
	}
}

// Open starts up the shardgroup
func (sg *ShardGroup) Open(ShardIDs []int, ShardCount int) (ready chan bool, err error) {
	sg.SetStatus(structs.ShardGroupStarting)

	sg.ShardCount = ShardCount
	sg.ShardIDs = ShardIDs

	ready = make(chan bool, 1)

	// TEMP
	// Maybe make this more fleshed out and push to a central event stats thing?
	go func() {
		start := time.Now().UTC()
		t := time.NewTicker(1 * time.Second)
		totalCount := int64(0)
		for {
			<-t.C
			count := int64(0)
			execution := int64(0)
			for _, shard := range sg.Shards {
				count += atomic.SwapInt64(shard.events, 0)
				execution += atomic.SwapInt64(shard.executionTime, 0)
			}
			totalCount += count
			since := time.Now().UTC().Sub(start)
			exec := ((float64(execution) / float64(len(sg.Shards)*1000000000)) * 100)
			sg.Logger.Debug().Msgf("%d events/s | %d total | %d avg/second | %s elapsed | %f%% execution",
				count, totalCount, int(float64(totalCount)/since.Seconds()), since, exec)
		}
	}()

	sg.Logger.Info().Msgf("Starting ShardGroup with %d shards", len(sg.ShardIDs))

	for _, shardID := range sg.ShardIDs {
		shard := sg.NewShard(shardID)
		sg.Shards[shardID] = shard

		err = shard.Connect()
		if err != nil && !xerrors.Is(err, context.Canceled) {
			sg.Logger.Error().Err(err).Msg("Failed to connect shard. Cannot continue")
			sg.Close()
			err = xerrors.Errorf("ShardGroup open: %w", err)
			return
		}

		go shard.Open()
	}

	sg.Logger.Debug().Msgf("All shards are now listening")
	sg.SetStatus(structs.ShardGroupConnecting)

	go func(sg *ShardGroup) {
		for _, shard := range sg.Shards {
			sg.Logger.Debug().Msgf("Waiting for shard %d to be ready", shard.ShardID)
			shard.WaitForReady()
		}
		sg.Logger.Debug().Msg("All shards in ShardGroup are ready")
		sg.SetStatus(structs.ShardGroupReady)
		close(ready)
	}(sg)

	return
}

// SetStatus changes the ShardGroup status
func (sg *ShardGroup) SetStatus(status structs.ShardGroupStatus) {
	sg.StatusMu.Lock()
	sg.Status = status
	sg.StatusMu.Unlock()
	sg.Manager.PublishEvent("SHARD_STATUS", structs.MessagingStatusUpdate{Status: int32(status)})
}

// Close closes the shard group and finishes any shards
func (sg *ShardGroup) Close() {
	sg.Logger.Info().Msg("Closing ShardGroup")
	sg.SetStatus(structs.ShardGroupClosing)
	for _, shard := range sg.Shards {
		shard.Close(1000)
	}
	sg.SetStatus(structs.ShardGroupClosed)
}
