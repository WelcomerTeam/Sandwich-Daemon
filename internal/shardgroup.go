package internal

import (
	"errors"
	"sync"
	"time"

	"github.com/WelcomerTeam/Discord/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/structs"
	jsoniter "github.com/json-iterator/go"
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

	ShardCount int32   `json:"shard_count"`
	ShardIDs   []int32 `json:"shard_ids"`

	shardsMu sync.RWMutex     `json:"-"`
	Shards   map[int32]*Shard `json:"shards"`

	guildsMu sync.RWMutex               `json:"-"`
	Guilds   map[discord.Snowflake]bool `json:"guilds"`

	ReadyWait *sync.WaitGroup `json:"-"`

	statusMu sync.RWMutex                      `json:"-"`
	Status   sandwich_structs.ShardGroupStatus `json:"status"`

	// MemberChunksCallback is used to signal when a guild is chunking.
	memberChunksCallbackMu sync.RWMutex                          `json:"-"`
	MemberChunksCallback   map[discord.Snowflake]*sync.WaitGroup `json:"-"`

	// MemberChunksComplete is used to signal if a guild has recently
	// been chunked. It is up to the guild task to remove this bool
	// a few seconds after finishing chunking.
	memberChunksCompleteMu sync.RWMutex                       `json:"-"`
	MemberChunksComplete   map[discord.Snowflake]*atomic.Bool `json:"-"`

	// MemberChunkCallbacks is used to signal when any MEMBER_CHUNK
	// events are received for the specific guild.
	memberChunkCallbacksMu sync.RWMutex                    `json:"-"`
	MemberChunkCallbacks   map[discord.Snowflake]chan bool `json:"-"`

	// Used to override when events can be processed.
	// Used to orchestrate scaling of shardgroups.
	floodgate *atomic.Bool
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
		ShardIDs:   shardIDs,

		shardsMu: sync.RWMutex{},
		Shards:   make(map[int32]*Shard),

		guildsMu: sync.RWMutex{},
		Guilds:   make(map[discord.Snowflake]bool),

		statusMu: sync.RWMutex{},
		Status:   sandwich_structs.ShardGroupStatusIdle,

		memberChunksCallbackMu: sync.RWMutex{},
		MemberChunksCallback:   make(map[discord.Snowflake]*sync.WaitGroup),

		memberChunksCompleteMu: sync.RWMutex{},
		MemberChunksComplete:   make(map[discord.Snowflake]*atomic.Bool),

		memberChunkCallbacksMu: sync.RWMutex{},
		MemberChunkCallbacks:   make(map[discord.Snowflake]chan bool),

		floodgate: &atomic.Bool{},
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

	sg.Manager.shardGroupsMu.RLock()
	for _, shardGroup := range sg.Manager.ShardGroups {
		if shardGroup.GetStatus() != sandwich_structs.ShardGroupStatusErroring && shardGroup.ID != sg.ID {
			shardGroup.SetStatus(sandwich_structs.ShardGroupStatusMarkedForClosure)
		}
	}
	sg.Manager.shardGroupsMu.RUnlock()

	ready = make(chan bool, 1)

	sg.Logger.Info().
		Int32("shardCount", sg.ShardCount).
		Int("shardIds", len(sg.ShardIDs)).
		Msg("Starting shardgroup")

	sg.shardsMu.Lock()
	for _, shardID := range sg.ShardIDs {
		shard := sg.NewShard(shardID)
		sg.Shards[shardID] = shard
	}
	sg.shardsMu.Unlock()

	if len(sg.ShardIDs) == 0 {
		return nil, ErrMissingShards
	}

	initialShard := sg.Shards[sg.ShardIDs[0]]

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
			sg.shardsMu.RLock()
			shard := sg.Shards[shardID]
			sg.shardsMu.RUnlock()

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
		sg.shardsMu.RLock()
		for _, shardID := range sg.ShardIDs {
			shard := sg.Shards[shardID]
			sg.WaitingFor.Store(shardID)
			shard.WaitForReady()
		}
		sg.shardsMu.RUnlock()

		sg.Logger.Info().Msg("All shards are now ready")

		sg.Manager.shardGroupsMu.RLock()
		for sgID, _sg := range sg.Manager.ShardGroups {
			if sgID != sg.ID {
				_sg.floodgate.Store(false)
				_sg.Close()
			}
		}
		sg.Manager.shardGroupsMu.RUnlock()

		sg.floodgate.Store(true)
		close(ready)
	}(sg)

	return ready, err
}

// Close closes all shards in a shardgroup.
func (sg *ShardGroup) Close() {
	sg.Logger.Info().Msgf("Closing shardgroup %d", sg.ID)

	sg.SetStatus(sandwich_structs.ShardGroupStatusClosing)

	closeWaiter := sync.WaitGroup{}

	sg.shardsMu.RLock()
	for _, sh := range sg.Shards {
		closeWaiter.Add(1)

		go func(sh *Shard) {
			sh.Close(websocket.StatusNormalClosure)
			closeWaiter.Done()
		}(sh)
	}
	sg.shardsMu.RUnlock()

	closeWaiter.Wait()

	sg.guildsMu.Lock()
	sg.Guilds = make(map[discord.Snowflake]bool)
	sg.guildsMu.Unlock()

	sg.SetStatus(sandwich_structs.ShardGroupStatusClosed)
}

// SetStatus sets the status of the ShardGroup.
func (sg *ShardGroup) SetStatus(status sandwich_structs.ShardGroupStatus) {
	sg.statusMu.Lock()
	defer sg.statusMu.Unlock()

	sg.Logger.Debug().Int("status", int(status)).Msg("ShardGroup status changed")

	sg.Status = status

	payload, _ := jsoniter.Marshal(sandwich_structs.ShardGroupStatusUpdate{
		Manager:    sg.Manager.Identifier.Load(),
		ShardGroup: sg.ID,
		Status:     sg.Status,
	})

	_ = sg.Manager.Sandwich.PublishGlobalEvent(sandwich_structs.SandwichEventShardGroupStatusUpdate, jsoniter.RawMessage(payload))
}

// GetStatus returns the status of a ShardGroup.
func (sg *ShardGroup) GetStatus() (status sandwich_structs.ShardGroupStatus) {
	sg.statusMu.RLock()
	defer sg.statusMu.RUnlock()

	return sg.Status
}
