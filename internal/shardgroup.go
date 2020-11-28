package gateway

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/rs/zerolog"
	"github.com/tevino/abool"
	"golang.org/x/net/context"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

// ShardGroup groups a selection of shards.
type ShardGroup struct {
	StatusMu sync.RWMutex             `json:"-"`
	Status   structs.ShardGroupStatus `json:"status"`

	ErrorMu sync.RWMutex `json:"-"`
	Error   string       `json:"error"`

	Start time.Time `json:"uptime"`

	WaitingFor *int32 `json:"waiting_for"`

	ID int32 `json:"id"` // track of shardgroups

	Manager *Manager       `json:"-"`
	Logger  zerolog.Logger `json:"-"`

	ShardCount int   `json:"shard_count"`
	ShardIDs   []int `json:"shard_ids"`

	ShardsMu sync.RWMutex   `json:"-"`
	Shards   map[int]*Shard `json:"shards"`

	// WaitGroup for detecting when all shards are ready
	Wait *sync.WaitGroup `json:"-"`

	// Mutex map to limit connections per max_concurrency connection
	IdentifyBucket map[int]*sync.Mutex `json:"-"`

	// Used for detecting errors during shard startup
	err chan error
	// Used to close active goroutines
	close chan void

	floodgate *abool.AtomicBool
}

// NewShardGroup creates a new shardgroup.
func (mg *Manager) NewShardGroup(id int32) *ShardGroup {
	return &ShardGroup{
		StatusMu: sync.RWMutex{},
		Status:   structs.ShardGroupIdle,
		ErrorMu:  sync.RWMutex{},
		Error:    "",

		WaitingFor: new(int32),

		ID: id,

		Manager: mg,
		Logger:  mg.Logger,

		ShardsMu:       sync.RWMutex{},
		Shards:         make(map[int]*Shard),
		Wait:           &sync.WaitGroup{},
		IdentifyBucket: make(map[int]*sync.Mutex),
		err:            make(chan error),
		close:          make(chan void),

		floodgate: abool.New(),
	}
}

// Open starts up the shardgroup.
func (sg *ShardGroup) Open(shardIDs []int, shardCount int) (ready chan bool, err error) {
	sg.Start = time.Now().UTC()

	sg.Manager.ShardGroupsMu.Lock()
	for _, _sg := range sg.Manager.ShardGroups {
		// We preferably do not want to mark an erroring shardgroup as replaced as it overwrites how it is displayed.
		if _sg.Status != structs.ShardGroupError {
			if err := _sg.SetStatus(structs.ShardGroupReplaced); err != nil {
				_sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
			}
		}
	}
	sg.Manager.ShardGroupsMu.Unlock()

	if err := sg.SetStatus(structs.ShardGroupStarting); err != nil {
		sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
	}

	sg.ShardCount = shardCount
	sg.ShardIDs = shardIDs

	ready = make(chan bool, 1)

	sg.Logger.Info().Msgf("Starting ShardGroup with %d shards", len(sg.ShardIDs))

	sg.ShardsMu.Lock()
	for _, shardID := range sg.ShardIDs {
		shard := sg.NewShard(shardID)
		sg.Shards[shardID] = shard
	}
	sg.ShardsMu.Unlock()

	for index, shardID := range sg.ShardIDs {
		sg.ShardsMu.RLock()
		shard := sg.Shards[shardID]
		sg.ShardsMu.RUnlock()

		for {
			err = shard.Connect()

			// We will only close down the entire shardgroup in the event that the first
			// shard fails to connect. This is to ensure that others are able to properly
			// connect and not just another generic connect issue which would be annoying
			// if 1 of your 250 shards die whilst starting up causing all the others to
			// also be killed.
			if index == 0 && err != nil && !xerrors.Is(err, context.Canceled) {
				retries := atomic.LoadInt32(shard.Retries)

				// In the event shard 0 does not successfully connect, we will attempt a
				// few more times in case it is one of those generic connection issues.
				if retries > 0 {
					sg.Logger.Error().Err(err).Int32("retries", retries).Msg("Failed to connect shard. Retrying...")
				} else {
					sg.Logger.Error().Err(err).Msg("Failed to connect shard. Cannot continue")

					sg.ErrorMu.Lock()
					sg.Error = err.Error()
					sg.ErrorMu.Unlock()

					sg.Close()

					if err := sg.SetStatus(structs.ShardGroupError); err != nil {
						sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
					}

					for _, shard := range sg.Shards {
						if err = shard.SetStatus(structs.ShardClosed); err != nil {
							shard.Logger.Error().Err(err).Msg("Encountered error setting shard status")
						}
					}

					err = xerrors.Errorf("ShardGroup open: %w", err)

					return
				}

				atomic.AddInt32(shard.Retries, -1)
				sg.Logger.Debug().Msgf("Shardgroup retries is now at %d", atomic.LoadInt32(shard.Retries))
			} else {
				break
			}
		}

		go shard.Open()
	}

	sg.Logger.Debug().Msgf("All shards are now listening")

	if err := sg.SetStatus(structs.ShardGroupConnecting); err != nil {
		sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
	}

	go func(sg *ShardGroup) {
		sg.ShardsMu.RLock()
		for _, shardID := range sg.ShardIDs {
			shard := sg.Shards[shardID]
			sg.Logger.Debug().Msgf("Waiting for shard %d to be ready", shard.ShardID)
			atomic.StoreInt32(sg.WaitingFor, int32(shardID))
			shard.WaitForReady()
		}
		sg.ShardsMu.RUnlock()

		sg.Logger.Debug().Msg("All shards in ShardGroup are ready")

		if err := sg.SetStatus(structs.ShardGroupReady); err != nil {
			sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
		}

		sg.Manager.ShardGroupsMu.RLock()
		for index, _sg := range sg.Manager.ShardGroups {
			if _sg != sg {
				_sg.floodgate.UnSet()
				sg.Manager.Logger.Debug().Int32("index", index).Msg("Killed ShardGroup")
				_sg.Close()
			}
		}
		sg.Manager.ShardGroupsMu.RUnlock()

		sg.floodgate.Set()
		close(ready)
	}(sg)

	return ready, nil
}

// SetStatus changes the ShardGroup status.
func (sg *ShardGroup) SetStatus(status structs.ShardGroupStatus) (err error) {
	sg.StatusMu.Lock()
	sg.Status = status
	sg.StatusMu.Unlock()

	return sg.Manager.PublishEvent("SHARD_STATUS", structs.MessagingStatusUpdate{Status: int32(status)})
}

// Close closes the shard group and finishes any shards.
func (sg *ShardGroup) Close() {
	sg.Logger.Info().Msg("Closing ShardGroup")

	if err := sg.SetStatus(structs.ShardGroupClosing); err != nil {
		sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
	}

	sg.ShardsMu.RLock()
	for _, shard := range sg.Shards {
		shard.Close(websocket.StatusNormalClosure)
	}
	sg.ShardsMu.RUnlock()

	if err := sg.SetStatus(structs.ShardGroupClosed); err != nil {
		sg.Logger.Error().Err(err).Msg("Encountered error setting shard group status")
	}
}
