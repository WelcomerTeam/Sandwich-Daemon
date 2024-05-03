package internal

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"golang.org/x/net/context"
	"nhooyr.io/websocket"
)

// ShardGroup represents a group of shards.
type ShardGroup struct {
	Error *atomic.String `json:"error"`

	Manager *Manager       `json:"-"`
	Logger  zerolog.Logger `json:"-"`

	Start *atomic.Time `json:"start"`

	WaitingFor *atomic.Int32 `json:"-"`

	userMu sync.RWMutex  `json:"-"`
	User   *discord.User `json:"user"`

	ID int32 `json:"id"`

	ShardCount int32 `json:"shard_count"`

	shardIdsMu sync.RWMutex `json:"-"`
	ShardIDs   []int32      `json:"shard_ids"`

	Shards *csmap.CsMap[int32, *Shard] `json:"shards"`

	Guilds *csmap.CsMap[discord.Snowflake, struct{}] `json:"guilds"`

	ReadyWait *sync.WaitGroup `json:"-"`

	statusMu sync.RWMutex                      `json:"-"`
	Status   sandwich_structs.ShardGroupStatus `json:"status"`

	// Used to override when events can be processed.
	// Used to orchestrate scaling of shardgroups.
	floodgateMu sync.RWMutex
	floodgate   bool
}

// NewShardGroup creates a new shardgroup.
func (mg *Manager) NewShardGroup(shardGroupID int32, shardIDs []int32, shardCount int32) (sg *ShardGroup) {
	sg = &ShardGroup{
		Error: &atomic.String{},

		Manager: mg,
		Logger:  mg.Logger,

		Start: atomic.NewTime(time.Now().UTC()),

		WaitingFor: &atomic.Int32{},

		ID: shardGroupID,

		ShardCount: shardCount,

		shardIdsMu: sync.RWMutex{},
		ShardIDs:   shardIDs,

		Shards: csmap.Create(
			csmap.WithSize[int32, *Shard](1),
		),

		Guilds: csmap.Create(
			csmap.WithSize[discord.Snowflake, struct{}](50),
		),

		statusMu: sync.RWMutex{},
		Status:   sandwich_structs.ShardGroupStatusIdle,

		floodgate:   false,
		floodgateMu: sync.RWMutex{},
	}

	return sg
}

// Open handles the startup of a shard group.
// On startup of a shard group, the first shard is connected and ran to confirm token and such are valid.
// If an issue occurs starting the first shard, open will return an error. Other shards will then connect
// concurrently and will attempt to reconnect on error.
// Once the shardgroup has fully finished connecting and are ready, then floodgate will be enabled allowing
// their events to be handled.
func (sg *ShardGroup) Open() (ready chan bool, err error) {
	sg.Start.Store(time.Now().UTC())

	sg.Manager.ShardGroups.Range(func(i int32, shardGroup *ShardGroup) bool {
		if shardGroup.GetStatus() != sandwich_structs.ShardGroupStatusErroring && shardGroup.ID != sg.ID {
			shardGroup.SetStatus(sandwich_structs.ShardGroupStatusMarkedForClosure)
		}
		return false
	})

	ready = make(chan bool, 1)

	sg.Logger.Info().
		Int32("shardCount", sg.ShardCount).
		Int("shardIds", len(sg.ShardIDs)).
		Msg("Starting shardgroup")

	sg.shardIdsMu.Lock()
	for _, shardID := range sg.ShardIDs {
		shard := sg.NewShard(shardID)
		sg.Shards.Store(shardID, shard)
	}
	sg.shardIdsMu.Unlock()

	if len(sg.ShardIDs) == 0 {
		return nil, ErrMissingShards
	}

	initialShard, ok := sg.Shards.Load(sg.ShardIDs[0])

	if !ok {
		return nil, ErrNoShardPresent
	}

	sg.SetStatus(sandwich_structs.ShardGroupStatusConnecting)

	for {
		err = initialShard.Connect()

		if err != nil && !errors.Is(err, context.Canceled) {
			retriesRemaining := initialShard.RetriesRemaining.Load()

			if retriesRemaining > 0 {
				sg.Logger.Error().Err(err).
					Int32("retries_remaining", retriesRemaining).
					Msg("Failed to connect shard. Retrying")
			} else {
				sg.Logger.Error().Err(err).
					Msg("Failed to connect shard. Cannot continue")

				sg.Error.Store(err.Error())

				sg.SetStatus(sandwich_structs.ShardGroupStatusErroring)

				sg.Close()

				return
			}

			initialShard.RetriesRemaining.Sub(1)
		} else {
			break
		}
	}

	go initialShard.Open()

	connectGroup := sync.WaitGroup{}

	for _, shardID := range sg.ShardIDs[1:] {
		connectGroup.Add(1)

		go func(shardID int32) {
			shard, ok := sg.Shards.Load(shardID)

			if !ok {
				sg.Logger.Error().
					Int32("shardId", shardID).
					Msg("Failed to load shard")
				connectGroup.Done()
				return
			}

			for {
				shardErr := shard.Connect()
				if shardErr != nil && !errors.Is(shardErr, context.Canceled) {
					sg.Logger.Warn().Err(shardErr).
						Int32("shardId", shardID).
						Msgf("Failed to connect shard. Retrying")
				} else {
					go shard.Open()

					break
				}
			}

			connectGroup.Done()
		}(shardID)
	}

	connectGroup.Wait()
	sg.Logger.Info().Msg("All shards have connected")

	sg.SetStatus(sandwich_structs.ShardGroupStatusConnected)

	go func(sg *ShardGroup) {
		sg.shardIdsMu.RLock()
		for _, shardID := range sg.ShardIDs {
			shard, ok := sg.Shards.Load(shardID)

			if !ok {
				sg.Logger.Error().Int32("shardId", shardID).Msg("Failed to load shard")
				continue
			}
			sg.WaitingFor.Store(shardID)
			shard.WaitForReady()
		}
		sg.shardIdsMu.RUnlock()

		sg.Logger.Info().Msg("All shards are now ready")

		sg.Manager.ShardGroups.Range(func(shardGroupID int32, shardGroup *ShardGroup) bool {
			if shardGroupID != sg.ID {
				sg.floodgateMu.Lock()
				sg.floodgate = false
				sg.floodgateMu.Unlock()

				shardGroup.Close()
			}
			return false
		})

		sg.floodgateMu.Lock()
		sg.floodgate = true
		sg.floodgateMu.Unlock()

		close(ready)
	}(sg)

	return ready, err
}

// Close closes all shards in a shardgroup.
func (sg *ShardGroup) Close() {
	sg.Logger.Info().Msgf("Closing shardgroup %d", sg.ID)

	sg.SetStatus(sandwich_structs.ShardGroupStatusClosing)

	closeWaiter := sync.WaitGroup{}

	sg.Shards.Range(func(i int32, sh *Shard) bool {
		closeWaiter.Add(1)

		go func(sh *Shard) {
			sh.Close(websocket.StatusNormalClosure, false)
			closeWaiter.Done()
		}(sh)

		return false
	})

	closeWaiter.Wait()

	sg.Guilds.Clear()

	sg.SetStatus(sandwich_structs.ShardGroupStatusClosed)
}

// SetStatus sets the status of the ShardGroup.
func (sg *ShardGroup) SetStatus(status sandwich_structs.ShardGroupStatus) {
	sg.statusMu.Lock()
	defer sg.statusMu.Unlock()

	sg.Logger.Debug().Int("status", int(status)).Msg("ShardGroup status changed")

	sg.Status = status

	payload, _ := sandwichjson.Marshal(sandwich_structs.ShardGroupStatusUpdate{
		Manager:    sg.Manager.Identifier.Load(),
		ShardGroup: sg.ID,
		Status:     sg.Status,
	})

	_ = sg.Manager.Sandwich.PublishGlobalEvent(sandwich_structs.SandwichEventShardGroupStatusUpdate, json.RawMessage(payload))
}

// GetStatus returns the status of a ShardGroup.
func (sg *ShardGroup) GetStatus() (status sandwich_structs.ShardGroupStatus) {
	sg.statusMu.RLock()
	defer sg.statusMu.RUnlock()

	return sg.Status
}
