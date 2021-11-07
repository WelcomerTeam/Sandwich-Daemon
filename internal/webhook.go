package internal

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"golang.org/x/xerrors"
)

// Embed colours for webhooks.
const (
	EmbedColourSandwich = 16701571
	EmbedColourWarning  = 16760839
	EmbedColourDanger   = 14431557

	WebhookRateLimitDuration = 5 * time.Second
	WebhookRateLimitLimit    = 5
)

// PublishSimpleWebhook is a helper function for creating quicker webhook messages.
func (sg *Sandwich) PublishSimpleWebhook(title string, description string, footer string, colour int) {
	sg.PublishWebhook(discord.WebhookMessage{
		Embeds: []*discord.Embed{
			{
				Title:       title,
				Description: description,
				Color:       colour,
				Timestamp:   webhookTime(time.Now().UTC()),
				Footer: &discord.EmbedFooter{
					Text: footer,
				},
			},
		},
	})
}

// PublishWebhook sends a webhook message to all added webhooks in the configuration.
func (sg *Sandwich) PublishWebhook(message discord.WebhookMessage) {
	for _, webhook := range sg.Configuration.Webhooks {
		_, err := sg.SendWebhook(webhook, message)
		if err != nil && !xerrors.Is(err, context.Canceled) {
			sg.Logger.Warn().Err(err).Str("url", webhook).Msg("Failed to send webhook")
		}
	}
}

func (sg *Sandwich) SendWebhook(webhookUrl string, message discord.WebhookMessage) (status int, err error) {
	webhookUrl = strings.TrimSpace(webhookUrl)

	_, err = url.Parse(webhookUrl)
	if err != nil {
		return -1, xerrors.Errorf("failed to parse webhook URL: %w", err)
	}

	res, err := json.Marshal(message)
	if err != nil {
		return -1, xerrors.Errorf("failed to marshal webhook message: %w", err)
	}

	sg.webhookBuckets.CreateWaitForBucket(webhookUrl, WebhookRateLimitLimit, WebhookRateLimitDuration)

	_, status, err = sg.Client.Fetch(sg.ctx, "POST", webhookUrl, bytes.NewBuffer(res), map[string]string{
		"Content-Type": "application/json",
	})

	return status, err
}
