package sandwich

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// IdentifyViaURL is a bare minimum identify provider that uses a URL to identify shards.
// This will send a POST request to the URL with the shard_id, shard_count, token, token_hash and max_concurrency in the body, or in the URL.

// This is done using formatting tags:
// - {shard_id}
// - {shard_count}
// - {token}
// - {token_hash}
// - {max_concurrency}

// This will expect a 200 or 204 response.
// If the response is a 402, it will retry based on the header `X-Retry-After-Ms` or 5000 milliseconds.
// If the response is anything else, it will return an error.
type IdentifyViaURL struct {
	URL     string
	Headers map[string]string
}

func NewIdentifyViaURL(url string, headers map[string]string) *IdentifyViaURL {
	return &IdentifyViaURL{
		URL:     url,
		Headers: headers,
	}
}

func (i *IdentifyViaURL) Identify(ctx context.Context, shard *Shard) error {
	method := sha256.New()
	method.Write([]byte(shard.Application.Configuration.Load().BotToken))
	tokenHash := hex.EncodeToString(method.Sum(nil))

	identifyURL := i.URL
	identifyURL = strings.Replace(identifyURL, "{shard_id}", strconv.Itoa(int(shard.ShardID)), 1)
	identifyURL = strings.Replace(identifyURL, "{shard_count}", strconv.Itoa(int(shard.Application.ShardCount.Load())), 1)
	identifyURL = strings.Replace(identifyURL, "{token}", shard.Application.Configuration.Load().BotToken, 1)
	identifyURL = strings.Replace(identifyURL, "{token_hash}", tokenHash, 1)
	identifyURL = strings.Replace(identifyURL, "{max_concurrency}", strconv.Itoa(int(shard.Application.Gateway.Load().SessionStartLimit.MaxConcurrency)), 1)

	_, err := url.Parse(identifyURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	identifyPayload := struct {
		ShardID        int    `json:"shard_id"`
		ShardCount     int    `json:"shard_count"`
		MaxConcurrency int    `json:"max_concurrency"`
		Token          string `json:"token"`
		TokenHash      string `json:"token_hash"`
	}{
		ShardID:        int(shard.ShardID),
		ShardCount:     int(shard.Application.Configuration.Load().ShardCount),
		MaxConcurrency: int(shard.Application.Gateway.Load().SessionStartLimit.MaxConcurrency),
		Token:          shard.Application.Configuration.Load().BotToken,
		TokenHash:      tokenHash,
	}

	body, err := json.Marshal(identifyPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal identify payload: %w", err)
	}

	client := http.DefaultClient

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, identifyURL, bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("failed to create identify request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		for key, value := range i.Headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			defer resp.Body.Close()
		}

		var retryAfter time.Duration

		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent) {
			return nil
		}

		if err != nil {
			retryAfter = StandardIdentifyLimit
		} else {
			retryAfterHeader := resp.Header.Get("X-Retry-After-Ms")
			retryAfterInt, _ := strconv.Atoi(retryAfterHeader)

			if retryAfterInt > 0 {
				retryAfter = time.Duration(retryAfterInt) * time.Millisecond
			} else {
				retryAfter = StandardIdentifyLimit
			}
		}

		time.Sleep(retryAfter)
	}
}
