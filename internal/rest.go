package internal

import (
	"net/http"
	"strconv"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"github.com/fasthttp/router"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"golang.org/x/xerrors"
)

var (
	ErrForbidden = xerrors.New("You do not have permission to access this")

	DiscordUserMeEndpoint = "https://discord.com/api/users/@me"

	// When enabled, / will serve the dist folder.
	EnableDistHandling = true
	DistPath           = "sandwich/dist"
)

func (sg *Sandwich) NewRestRouter() (routerHandler fasthttp.RequestHandler, fsHandler fasthttp.RequestHandler) {
	r := router.New()
	r.GET("/api/status", sg.StatusEndpoint)

	fs := fasthttp.FS{
		IndexNames:     []string{"index.html"},
		Root:           DistPath,
		Compress:       true,
		CompressBrotli: true,
		CacheDuration:  time.Hour,
		PathNotFound: func(ctx *fasthttp.RequestCtx) {
			ctx.Response.Reset()
			ctx.SendFile(DistPath + "/index.html")
		},
	}

	return r.Handler, fs.NewRequestHandler()
}

// RequireDiscordAuthentication wraps a RequestHandler and
// redirects to oauth if not in session and raises Unauthorized
// if user is not permitted.
func (sg *Sandwich) requireDiscordAuthentication(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		h(ctx)
		return
	})
}

func (sg *Sandwich) HandleRequest(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	path := ctx.Request.URI().PathOriginal()

	defer func() {
		var log *zerolog.Event

		processingMS := time.Since(start).Milliseconds()
		statusCode := ctx.Response.StatusCode()

		switch {
		case (statusCode >= 400 && statusCode <= 499):
			log = sg.Logger.Warn()
		case (statusCode >= 500 && statusCode <= 599):
			log = sg.Logger.Error()
		default:
			log = sg.Logger.Info()
		}

		log.Msgf("%s %s %s %d %d %dms",
			ctx.RemoteAddr(),
			ctx.Request.Header.Method(),
			path,
			statusCode,
			len(ctx.Response.Body()),
			processingMS,
		)

		ctx.Response.Header.Set("X-Elapsed", strconv.FormatInt(processingMS, MagicDecimalBase))
	}()

	fasthttp.CompressHandlerBrotliLevel(
		func(ctx *fasthttp.RequestCtx) {
			sg.RouterHandler(ctx)

			if ctx.Response.StatusCode() == http.StatusNotFound {
				ctx.Response.Reset()
				sg.DistHandler(ctx)
			}
		},
		fasthttp.CompressBrotliDefaultCompression,
		fasthttp.CompressDefaultCompression,
	)(ctx)
}

// /api/status: Returns managers, shardgroups and shard status.
func (sg *Sandwich) StatusEndpoint(ctx *fasthttp.RequestCtx) {
	sg.managersMu.RLock()
	response := &structs.StatusEndpointResponse{
		Managers: make([]*structs.StatusEndpointManager, 0, len(sg.Managers)),
	}

	for _, manager := range sg.Managers {
		manager.shardGroupsMu.RLock()

		manager.configurationMu.RLock()
		friendlyName := manager.Configuration.FriendlyName
		manager.configurationMu.RUnlock()

		statusManager := &structs.StatusEndpointManager{
			DisplayName: friendlyName,
			ShardGroups: make([]*structs.StatusEndpointShardGroup, 0, len(manager.ShardGroups)),
		}

		for _, shardGroup := range manager.ShardGroups {
			shardGroup.shardsMu.RLock()
			statusShardGroup := &structs.StatusEndpointShardGroup{
				ShardGroupID: int(shardGroup.ID),
				Shards:       make([][5]int, 0, len(shardGroup.Shards)),
				Status:       shardGroup.Status,
			}

			for _, shard := range shardGroup.Shards {
				shard.channelMu.RLock()
				statusShardGroup.Shards = append(statusShardGroup.Shards, [5]int{
					shard.ShardID,
					int(shard.Status),
					int(shard.LastHeartbeatAck.Load().Sub(shard.LastHeartbeatSent.Load()).Milliseconds()),
					len(shard.Guilds),
					int(time.Since(shard.Start).Seconds()),
				})
				shard.channelMu.RUnlock()
			}
			shardGroup.shardsMu.RUnlock()

			statusManager.ShardGroups = append(statusManager.ShardGroups, statusShardGroup)
		}
		manager.shardGroupsMu.RUnlock()

		response.Managers = append(response.Managers, statusManager)
	}
	sg.managersMu.RUnlock()

	body, err := json.Marshal(response)
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)

		return
	}

	ctx.Write(body)
	ctx.SetStatusCode(http.StatusOK)
}

// func (sg *Sandwich) testEndpoint(ctx *fasthttp.RequestCtx) {
// 	b, _ := json.Marshal(sg)
// 	ctx.Write(b)
// 	ctx.SetStatusCode(http.StatusOK)
// }
