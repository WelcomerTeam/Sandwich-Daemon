package internal

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/xerrors"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (sg *Sandwich) SendWebhook(ctx context.Context, webhookUrl string,
	message discord.WebhookMessage) (status int, err error) {
	var c *Client

	// We will trim whitespace just in case.
	webhookUrl = strings.TrimSpace(webhookUrl)

	_, err = url.Parse(webhookUrl)
	if err != nil {
		return -1, xerrors.Errorf("failed to parse webhook URL: %w", err)
	}

	c = sg.NewClient()

	res, err := json.Marshal(message)
	if err != nil {
		return -1, xerrors.Errorf("failed to marshal webhook message: %w", err)
	}

	_, status, err = c.Fetch(ctx, "POST", webhookUrl, bytes.NewBuffer(res), map[string]string{
		"Content-Type": "application/json",
	})

	return status, err
}
