package internal

// ShardGroup Connect() and opens (in goroutine) first shard
// continue with all others and wait for all are done


// NewShard
// Open starts listen to message and error channel and processes events.
// Connect to gateway and setup message channels
// Listen handles reading from websocket, errors and basic reconnection
// Feed reads from the gateway and decompresses messages and push to message channel
// OnEvent handles gateway ops and dispatch
// OnDispatch handles cheking blacklists, handling dispatch and publishing
// Heartbeat maintains Heartbeat
// Reconnect reconnects to gateway
// Close sends a close code

// Resume
// Identify
// Reconnect

// SendEvent sends a sentpayload packet
// WriteJSON sends a message to discord respecting ratelimits

// WaitForReady returns when shard is ready

// SetStatus

// ChunkGuild chunks a guild

// readMessage returns a message or error from channels
// decodeContent unmarshals received payload
