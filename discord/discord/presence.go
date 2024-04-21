package discord

// PresenceStatus represents a presence's status.
type PresenceStatus string

// Presence statuses.
const (
	PresenceStatusIdle    PresenceStatus = "idle"
	PresenceStatusDND     PresenceStatus = "dnd"
	PresenceStatusOnline  PresenceStatus = "online"
	PresenceStatusOffline PresenceStatus = "offline"
)

// ActivityType represents an activity's type.
type ActivityType int

// Activity types.
const (
	ActivityTypeGame ActivityType = iota
	ActivityTypeStreaming
	ActivityTypeListening
)

// ActivityFlag represents an activity's flags.
type ActivityFlag uint16

// Activity flags.
const (
	ActivityFlagInstance ActivityFlag = 1 << iota
	ActivityFlagJoin
	ActivityFlagSpectate
	ActivityFlagJoinRequest
	ActivityFlagSync
	ActivityFlagPlay
)

// Activity represents an activity as sent as part of other packets.
type Activity struct {
	Timestamps    *Timestamps   `json:"timestamps,omitempty"`
	ApplicationID *Snowflake    `json:"application_id,omitempty"`
	Party         *Party        `json:"party,omitempty"`
	Assets        *Assets       `json:"assets,omitempty"`
	Secrets       *Secrets      `json:"secrets,omitempty"`
	Flags         *ActivityFlag `json:"flags,omitempty"`
	Name          string        `json:"name"`
	URL           string        `json:"url"`
	Details       string        `json:"details"`
	State         string        `json:"state"`
	Type          ActivityType  `json:"type"`
	Instance      bool          `json:"instance"`
}

// Timestamps represents the starting and ending timestamp of an activity.
type Timestamps struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// Party represents an activity's current party information.
type Party struct {
	ID   string  `json:"id,omitempty"`
	Size []int32 `json:"size,omitempty"`
}

// Assets represents an activity's images and their hover texts.
type Assets struct {
	LargeImage string `json:"large_image"`
	LargeText  string `json:"large_text"`
	SmallImage string `json:"small_image"`
	SmallText  string `json:"small_text"`
}

// Secrets represents an activity's secrets for Rich Presence joining and spectating.
type Secrets struct {
	Join     string `json:"join"`
	Spectate string `json:"spectate"`
	Match    string `json:"match"`
}

// ClientStatus represent's the status of a client.
type ClientStatus struct {
	Desktop string `json:"desktop"`
	Mobile  string `json:"mobile"`
	Web     string `json:"web"`
}
