package sandwich

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/WelcomerTeam/RealRock/bucketstore"
)

// IdentifyViaBuckets is a bare minimum identify provider that uses buckets to identify shards.
// This will work for most use cases, but it's not the most efficient way to identify shards when dealing with multiple processes.
type IdentifyViaBuckets struct {
	bucketStore *bucketstore.BucketStore
}

func NewIdentifyViaBuckets() *IdentifyViaBuckets {
	return &IdentifyViaBuckets{
		bucketStore: bucketstore.NewBucketStore(),
	}
}

func (i *IdentifyViaBuckets) Identify(_ context.Context, shard *Shard) error {
	method := sha256.New()
	method.Write([]byte(shard.application.configuration.Load().BotToken))
	tokenHash := hex.EncodeToString(method.Sum(nil))

	bucketName := fmt.Sprintf(
		"identify:%s:%d",
		tokenHash,
		shard.shardID%shard.application.gateway.Load().SessionStartLimit.MaxConcurrency,
	)

	// Create the bucket if it doesn't exist with a limit of 1 request per IdentifyRateLimit.
	i.bucketStore.CreateBucket(bucketName, 1, IdentifyRateLimit)

	err := i.bucketStore.WaitForBucket(bucketName)
	if err != nil {
		return fmt.Errorf("failed to wait for bucket: %w", err)
	}

	return nil
}
