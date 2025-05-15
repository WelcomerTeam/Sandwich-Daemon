package sandwich

import (
	"context"

	"github.com/WelcomerTeam/Discord/discord"
)

type contextKey string

var guildIDKey contextKey = "guildID"

// WithGuildID adds a guild ID to the context
func WithGuildID(ctx context.Context, guildID discord.Snowflake) context.Context {
	return context.WithValue(ctx, guildIDKey, guildID)
}

// GuildIDFromContext retrieves the guild ID from the context if present
func GuildIDFromContext(ctx context.Context) (discord.Snowflake, bool) {
	guildID, ok := ctx.Value(guildIDKey).(discord.Snowflake)

	return guildID, ok
}
