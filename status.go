package sandwich

type ApplicationStatus int32

const (
	ApplicationStatusIdle ApplicationStatus = iota
	ApplicationStatusFailed
	ApplicationStatusStarting
	ApplicationStatusConnecting
	ApplicationStatusConnected
	ApplicationStatusReady
	ApplicationStatusStopping
	ApplicationStatusStopped
)

func (status ApplicationStatus) String() string {
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

type ShardStatus int32

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
