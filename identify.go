package sandwich

import (
	"context"
	"time"
)

var (
	StandardIdentifyLimit = 5 * time.Second
	IdentifyRetry         = 5 * time.Second
	IdentifyRateLimit     = StandardIdentifyLimit + (time.Millisecond * 500)
)

type IdentifyProvider interface {
	Identify(ctx context.Context, shard *Shard) error
}
