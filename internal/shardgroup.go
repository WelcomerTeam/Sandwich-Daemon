package internal

import (
	"sync"
	"time"

	snowflake "github.com/WelcomerTeam/RealRock/snowflake"
	"github.com/rs/zerolog"
	"go.uber.org/atomic"
	"nhooyr.io/websocket"
)

// ShardGroup represents a group of shards
type ShardGroup struct {
	Error atomic.String `json:"error"`

	Manager *Manager       `json:"-"`
	Logger  zerolog.Logger `json:"-"`

	Start time.Time `json:"start"`

	WaitingFor atomic.Int32 `json:"-"`

	ID int32 `json:"id"`

	ShardCount int   `json:"shard_count"`
	ShardIDs   []int `json:"shard_ids"`

	shardsMu sync.RWMutex   `json:"-"`
	Shards   map[int]*Shard `json:"shards"`

	ReadyWait *sync.WaitGroup `json:"-"`

	// MemberChunksCallback is used to signal when a guild is chunking.
	memberChunksCallbackMu sync.RWMutex                     `json:"-"`
	MemberChunksCallback   map[snowflake.ID]*sync.WaitGroup `json:"-"`

	// MemberChunksComplete is used to signal if a guild has recently
	// been chunked. It is up to the guild task to remove this bool
	// a few seconds after finishing chunking.
	memberChunksCompleteMu sync.RWMutex                  `json:"-"`
	MemberChunksComplete   map[snowflake.ID]*atomic.Bool `json:"-"`

	// MemberChunkCallbacks is used to signal when any MEMBER_CHUNK
	// events are received for the specific guild.
	memberChunkCallbacksMu sync.RWMutex               `json:"-"`
	MemberChunkCallbacks   map[snowflake.ID]chan bool `json:"-"`

	// Used to override when events can be processed.
	// Used to orchestrate scaling of shardgroups.
	floodgate *atomic.Bool
}

// NewShardGroup creates a new shardgroup.
func (mg *Manager) NewShardGroup(shardGroupID int32, shardIDs []int, shardCount int) (sg *ShardGroup) {
	sg = &ShardGroup{
		Error: atomic.String{},

		Manager: mg,
		Logger:  mg.Logger,

		Start: time.Now().UTC(),

		WaitingFor: atomic.Int32{},

		ID: shardGroupID,

		ShardCount: shardCount,
		ShardIDs:   shardIDs,

		shardsMu: sync.RWMutex{},
		Shards:   make(map[int]*Shard),

		memberChunksCallbackMu: sync.RWMutex{},
		MemberChunksCallback:   make(map[snowflake.ID]*sync.WaitGroup),

		memberChunksCompleteMu: sync.RWMutex{},
		MemberChunksComplete:   make(map[snowflake.ID]*atomic.Bool),

		memberChunkCallbacksMu: sync.RWMutex{},
		MemberChunkCallbacks:   make(map[snowflake.ID]chan bool),

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
	// TODO

	// Create Shards
	// Try connect first Shard. Retry on error and set sg.Error after too many retries.
	// Open first shard in goroutine
	// Create new goroutine for each shard that attempts to constantly retry then shard.Open after and Done a waitgroup
	// Wait for all shards to connect and open.
	// Create new goroutine that calls shard.WaitforReady
	// Enable floodgate and close ready.

	return
}

// Close closes all shards in a shardgroup.
func (sg *ShardGroup) Close() {
	sg.Logger.Info().Msgf("Closing shardgroup %d", sg.ID)

	sg.shardsMu.RLock()
	for _, sh := range sg.Shards {
		sh.Close(websocket.StatusNormalClosure)
	}
	sg.shardsMu.RUnlock()
}

// SetStatus
