package gateway

import (
	"bytes"
	"context"

	"github.com/TheRockettek/Sandwich-Daemon/internal/mqclients"
	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/andybalholm/brotli"
	"github.com/savsgio/gotils"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/xerrors"
)

type MQClient interface {
	String() string
	Channel() string
	Cluster() string

	Connect(ctx context.Context, clientName string, args map[string]interface{}) (err error)
	Publish(ctx context.Context, channel string, data []byte) (err error)
	// Function to receive a channel with messages
	// Function to close
}

func NewMQClient(mqType string) (MQClient, error) {
	switch mqType {
	case "stan":
		return &mqclients.StanMQClient{}, nil
	case "kafka":
		return &mqclients.KafkaMQClient{}, nil
	case "redis":
		return &mqclients.RedisMQClient{}, nil
	default:
		return nil, xerrors.New("No MQ client named " + mqType)
	}
}

// PublishEvent publishes a SandwichPayload.
func (sh *Shard) PublishEvent(packet *structs.SandwichPayload) (err error) {
	sh.Manager.ConfigurationMu.RLock()
	defer sh.Manager.ConfigurationMu.RUnlock()

	packet.Metadata = structs.SandwichMetadata{
		Version:    VERSION,
		Identifier: sh.Manager.Configuration.Identifier,
		Shard: [3]int{
			int(sh.ShardGroup.ID),
			sh.ShardID,
			sh.ShardGroup.ShardCount,
		},
	}

	payload, err := msgpack.Marshal(packet)
	if err != nil {
		return xerrors.Errorf("failed to marshal payload: %w", err)
	}

	sh.Logger.Trace().Str("event", gotils.B2S(payload)).Msgf("Processed %s event", packet.Type)

	// Compression testing of large payloads. In the future this *may* be
	// added however in its current state it is uncertain. With using a 1mb
	// msgpack payload, compression can be brought down to 48kb using brotli
	// level 11 however will take around 1.5 seconds. However, it is likely
	// level 0 or 6 will be used which produce 95kb in 3ms and 54kb in 20ms
	// respectively. It is likely the actual data portion of the payload will
	// be compressed so the metadata and the rest of the data can be preserved
	// then pass in the metadata it is compressed instead of using magic bytes
	// or guessing by consumers.

	// Whilst compression can prove a benefit, having it enabled for all events
	// do not provide any benefit and only affect larger payloads which is
	// not common apart from GUILD_CREATE events.

	// Sample testing of a GUILD_CREATE event:

	// METHOD | Level        | Ms   | Resulting Payload Size
	// -------|--------------|------|-----------------------
	// NONE   |              |      | 1011967
	// BROTLI | 0  (speed)   | 3    | 95908   ( 9.5%)
	// BROTLI | 6  (default) | 20   | 54545   ( 5.4%)
	// BROTLI | 11 (best)    | 1245 | 47044   ( 4.6%)
	// GZIP   | 1  (speed)   | 3    | 115799  (11.5%)
	// GZIP   | -1 (default) | 8    | 82336   ( 8.1%)
	// GZIP   | 9  (best)    | 19   | 78253   ( 7.7%)

	// Compression stats

	// RAW      |  1.12Mbit
	// BROTLI 0 | 64.33Kbit | 152ms
	// BROTLI 6 | 31.04Kbit | 841ms
	// BROTLI 9 | 30.71KBit | 695ms
	// GZIP   1 | 92.40KBit | 92ms
	// GZIP  -1 | 64.52KBit | 290ms
	// GZIP   9 | 61.01KBit | 720ms

	// This may not be the most efficient way but it was useful for testing many
	// payloads. More cohesive benchmarking will take place if this is ever properly
	// implemented and may be a 1.0 feature however it is unlikely to be necessary..

	// Payloads larger than 1MB will default to using Level 6 brotli compression.
	// For consistency sake, we also compress smaller payloads on the lowest level
	// which should not affect performance too much as they are still fairly fast.

	// a := time.Now()

	compressedPayload := sh.cp.Get().(*bytes.Buffer)

	if len(payload) > minPayloadCompressionSize {
		dc := sh.DefaultCompressor.Get().(*brotli.Writer)
		dc.Reset(compressedPayload)

		_, err = dc.Write(payload)
		if err != nil {
			sh.Logger.Warn().Err(err).Msg("Failed to write payload to brotli compressor")
		}

		dc.Flush()
		sh.DefaultCompressor.Put(dc)
	} else {
		fc := sh.FastCompressor.Get().(*brotli.Writer)
		fc.Reset(compressedPayload)

		_, err = fc.Write(payload)
		if err != nil {
			sh.Logger.Warn().Err(err).Msg("Failed to write payload to brotli compressor")
		}

		fc.Flush()
		sh.FastCompressor.Put(fc)
	}

	err = sh.Manager.ProducerClient.Publish(
		sh.ctx,
		sh.Manager.Configuration.Messaging.ChannelName,
		compressedPayload.Bytes(),
	)

	compressedPayload.Reset()
	sh.cp.Put(compressedPayload)

	if err != nil {
		return xerrors.Errorf("publishEvent publish: %w", err)
	}

	return nil
}
