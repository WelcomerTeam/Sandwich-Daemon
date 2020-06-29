package gateway

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// ShardGroupStatus represents a shardgroups status
type ShardGroupStatus int32

// Status Codes for ShardGroups
const (
	ShardGroupIdle     ShardGroupStatus = iota // Represents a ShardGroup that has been created but not opened yet
	ShardGroupStarting                         // Represent a ShardGroup that is still starting up clients
	ShardGroupReady                            // Represent a ShardGroup that has all its shards ready
	ShardGroupReplaced                         // Represent a ShardGroup that is going to be replaced soon by a new ShardGroup
	ShardGroupClosing                          // Represent a ShardGroup in the process of closing
	ShardGroupClosed                           // Represent a closed ShardGroup
)

// ShardGroup groups a selection of shards
type ShardGroup struct {
	StatusMu sync.RWMutex
	Status   ShardGroupStatus

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
		Status:  ShardGroupIdle,
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
func (sg *ShardGroup) Open(ShardIDs []int, ShardCount int) (err error) {
	sg.SetStatus(ShardGroupStarting)

	sg.ShardCount = ShardCount
	sg.ShardIDs = ShardIDs

	// TEMP
	// Maybe make this more fleshed out and push to a central event stats thing?
	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			<-t.C
			ttl := atomic.LoadInt64(sg.Events)
			println(ttl, "events/s")
			atomic.StoreInt64(sg.Events, 0)
		}
	}()

	for _, shardID := range sg.ShardIDs {
		shard := sg.NewShard(shardID)
		sg.Shards[shardID] = shard

		go func(shard *Shard) {
			for {
				errs := shard.Open()
				err := <-errs
				if !shard.Recoverable(err) {
					shard.Logger.Error().Err(err).Msg("Received error starting ShardGroup. Closing.")
					sg.Close()
					return
				}
			}
		}(shard)
	}

	// Wait for all shards to finish
	for _, shard := range sg.Shards {
		println("Waiting for ", shard.ShardID)
		shard.WaitUntilReady()
	}

	sg.SetStatus(ShardGroupReady)
	sg.Logger.Info().Msg("All shards are ready")

	// Make goroutine that waits for the Wait to be done which closes the err channel

	// Read from err channel, if it is empty were fine as all shards are ready

	return
}

// SetStatus changes the ShardGroup status
func (sg *ShardGroup) SetStatus(status ShardGroupStatus) {
	sg.StatusMu.Lock()
	sg.Status = status
	sg.StatusMu.Unlock()
}

// Close closes the shard group and finishes any shards
func (sg *ShardGroup) Close() {
	println("Closing shardgroup")
	sg.SetStatus(ShardGroupClosing)
	for _, shard := range sg.Shards {
		shard.Close()
	}
	sg.SetStatus(ShardGroupClosed)
}
