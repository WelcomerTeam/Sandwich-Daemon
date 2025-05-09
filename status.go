package sandwich

type ManagerStatus int

const (
	ManagerStatusIdle ManagerStatus = iota
	ManagerStatusFailed
	ManagerStatusStarting
	ManagerStatusConnecting
	ManagerStatusConnected
	ManagerStatusReady
	ManagerStatusStopping
	ManagerStatusStopped
)

func (status ManagerStatus) String() string {
	return []string{
		"Idle",
		"Failed",
		"Starting",
		"Connecting",
		"Connected",
		"Ready",
		"Stopping",
		"Stopped",
	}[status]
}

type ShardStatus int

const (
	ShardStatusIdle ShardStatus = iota
	ShardStatusFailed
	ShardStatusConnecting
	ShardStatusConnected
	ShardStatusReady
	ShardStatusStopping
	ShardStatusStopped
)

func (status ShardStatus) String() string {
	return []string{
		"Idle",
		"Failed",
		"Connecting",
		"Connected",
		"Ready",
		"Stopping",
		"Stopped",
	}[status]
}
