package internal

import (
	"context"
	"encoding/json"
	"errors"
	"runtime/debug"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	gotils_strings "github.com/savsgio/gotils/strings"
)

// EventDispatch represents the data returned by an event handler after processing state etc.
type EventDispatch struct {
	Extra                   map[string]json.RawMessage
	EventDispatchIdentifier *sandwich_structs.EventDispatchIdentifier
	Data                    json.RawMessage
}

// List of handlers for gateway events.
var gatewayHandlers = make(map[discord.GatewayOp]func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error)

// List of handlers for dispatch events.
var dispatchHandlers = make(map[string]func(ctx StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result EventDispatch, ok bool, err error))

func (sh *Shard) OnEvent(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) {
	err := GatewayDispatch(ctx, sh, msg, trace)
	if err != nil {
		if errors.Is(err, ErrNoGatewayHandler) {
			sh.Logger.Warn().
				Int("op", int(msg.Op)).
				Str("type", msg.Type).
				Msg("Gateway sent unknown packet")
		}

		sh.Logger.Error().Err(err).Msg("Gateway dispatch failed")
	}
}

// OnDispatch handles routing of discord event.
func (sh *Shard) OnDispatch(ctx context.Context, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (err error) {
	defer func() {
		if r := recover(); r != nil {
			errorMessage, ok := r.(error)

			if ok {
				sh.Logger.Error().
					Err(errorMessage).
					Int("op", int(msg.Op)).
					Str("type", msg.Type).
					Int("seq", int(msg.Sequence)).
					Bytes("data", msg.Data).
					Msg("Recovered panic in OnDispatch")
			} else {
				sh.Logger.Error().
					Str("err", "[unknown]").
					Int("op", int(msg.Op)).
					Str("type", msg.Type).
					Int("seq", int(msg.Sequence)).
					Bytes("data", msg.Data).
					Msg("Recovered panic in OnDispatch")
			}

			println(string(debug.Stack()))
		}
	}()

	if sh.Manager.ProducerClient == nil {
		return ErrProducerMissing
	}

	sh.Manager.eventBlacklistMu.RLock()
	contains := gotils_strings.Include(sh.Manager.eventBlacklist, msg.Type)
	sh.Manager.eventBlacklistMu.RUnlock()

	if contains {
		return nil
	}

	sh.Manager.configurationMu.RLock()
	cacheUsers := sh.Manager.Configuration.Caching.CacheUsers
	cacheMembers := sh.Manager.Configuration.Caching.CacheMembers
	storeMutuals := sh.Manager.Configuration.Caching.StoreMutuals
	disableTrace := sh.Manager.Configuration.DisableTrace
	sh.Manager.configurationMu.RUnlock()

	if !disableTrace {
		if trace == nil {
			trace = make(map[string]discord.Int64)
		}

		trace["state"] = discord.Int64(time.Now().Unix())
	}

	result, continuable, err := StateDispatch(StateCtx{
		context:      ctx,
		Shard:        sh,
		CacheUsers:   cacheUsers,
		CacheMembers: cacheMembers,
		StoreMutuals: storeMutuals,
	}, msg, trace)

	if err != nil {
		if !errors.Is(err, ErrNoDispatchHandler) {
			sh.Logger.Error().Err(err).Str("data", gotils_strconv.B2S(msg.Data)).Msg("Encountered error whilst handling " + msg.Type)
		}

		return err
	}

	if result.EventDispatchIdentifier == nil {
		sh.Logger.Error().Str("type", msg.Type).Str("data", gotils_strconv.B2S(msg.Data)).Msg("EventDispatchIdentifier is nil")
		result.EventDispatchIdentifier = &sandwich_structs.EventDispatchIdentifier{
			GloballyRouted: true, // Ensure they are globally routed for now
		}
	}

	sh.ShardGroup.floodgateMu.RLock()
	floodgate := sh.ShardGroup.floodgate
	sh.ShardGroup.floodgateMu.RUnlock()

	if !floodgate || !continuable {
		return nil
	}

	sh.Manager.produceBlacklistMu.RLock()
	contains = gotils_strings.Include(sh.Manager.produceBlacklist, msg.Type)
	sh.Manager.produceBlacklistMu.RUnlock()

	if contains {
		return nil
	}

	packet := &sandwich_structs.SandwichPayload{
		Op:                      msg.Op,
		Sequence:                msg.Sequence,
		Type:                    msg.Type,
		Data:                    result.Data,
		Extra:                   result.Extra,
		Trace:                   trace,
		EventDispatchIdentifier: result.EventDispatchIdentifier,
	}

	return sh.PublishEvent(ctx, packet)
}

func registerGatewayEvent(op discord.GatewayOp, handler func(ctx context.Context, sh *Shard, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) error) {
	gatewayHandlers[op] = handler
}

func registerDispatch(eventType string, handler func(ctx StateCtx, msg discord.GatewayPayload, trace sandwich_structs.SandwichTrace) (result EventDispatch, ok bool, err error)) {
	dispatchHandlers[eventType] = handler
}

// GatewayDispatch handles selecting the proper gateway handler and executing it.
func GatewayDispatch(ctx context.Context, sh *Shard,
	event discord.GatewayPayload, trace sandwich_structs.SandwichTrace,
) error {
	if f, ok := gatewayHandlers[event.Op]; ok {
		return f(ctx, sh, event, trace)
	}

	sh.Logger.Warn().Int("op", int(event.Op)).Msg("No gateway handler found")

	return ErrNoGatewayHandler
}

// StateDispatch handles selecting the proper state handler and executing it.
func StateDispatch(ctx StateCtx,
	event discord.GatewayPayload, trace sandwich_structs.SandwichTrace,
) (result EventDispatch, ok bool, err error) {
	if f, ok := dispatchHandlers[event.Type]; ok {
		return f(ctx, event, trace)
	}

	return WildcardEvent(ctx, event, trace)
}
