package internal

import (
	"net/http"
	"strconv"
	"time"

	discord "github.com/WelcomerTeam/Sandwich-Daemon/next/discord/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/next/structs"
	"github.com/fasthttp/router"
	"github.com/fasthttp/session/v2"
	"github.com/rs/zerolog"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	"github.com/valyala/fasthttp"
	"golang.org/x/xerrors"
)

var (
	ErrUserMissingAccess = xerrors.New("You are missing access")
	ErrUserNotLoggedIn   = xerrors.New("You are not logged in")

	discordUserMeEndpoint = "https://discord.com/api/users/@me"

	// When enabled, / will serve the dist folder.
	EnableDistHandling = true
	DistPath           = "sandwich/dist"

	loggedInAttrKey      = "isLoggedIn"
	authenticatedAttrKey = "isAuthenticated"
	userAttrKey          = "user"
)

func (sg *Sandwich) NewRestRouter() (routerHandler fasthttp.RequestHandler, fsHandler fasthttp.RequestHandler) {
	r := router.New()
	r.GET("/api/status", sg.StatusEndpoint)
	r.GET("/api/user", sg.UserEndpoint)
	r.GET("/api/dashboard", sg.DashboardGetEndpoint)

	r.POST("/api/manager", sg.ManagerUpdateEndpoint)

	r.GET("/login", sg.LoginEndpoint)
	r.GET("/logout", sg.LogoutEndpoint)
	r.GET("/callback", sg.CallbackEndpoint)

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
		isLoggedIn, ok := ctx.UserValue(loggedInAttrKey).(bool)
		if !ok {
			return
		}

		isAuthenticated, ok := ctx.UserValue(authenticatedAttrKey).(bool)
		if !ok {
			return
		}

		if !isLoggedIn {
			writeResponse(ctx, fasthttp.StatusUnauthorized, structs.BaseRestResponse{
				Ok:    false,
				Error: ErrUserNotLoggedIn.Error(),
			})

			return
		}

		sg.configurationMu.RLock()
		httpAccessEnabled := sg.Configuration.HTTP.Enabled
		sg.configurationMu.RUnlock()

		if !isAuthenticated && httpAccessEnabled {
			writeResponse(ctx, fasthttp.StatusForbidden, structs.BaseRestResponse{
				Ok:    false,
				Error: ErrUserMissingAccess.Error(),
			})

			return
		}

		h(ctx)

		return
	})
}

func writeResponse(ctx *fasthttp.RequestCtx, statusCode int, i interface{}) {
	body, err := json.Marshal(i)
	if err == nil {
		ctx.Write(body)
		ctx.SetStatusCode(statusCode)
	} else {
		ctx.SetStatusCode(http.StatusInternalServerError)
	}
}

// Returns if a user is authenticated.
// isLoggedIn: Has a valid user in session.
// isAuthenticated: User is in the UserAccess.
func (sg *Sandwich) authenticateValue(ctx *fasthttp.RequestCtx) (store *session.Store, err error) {
	var isLoggedIn bool

	var isAuthenticated bool

	var user discord.User

	defer func() {
		ctx.SetUserValue(loggedInAttrKey, isLoggedIn)
		ctx.SetUserValue(authenticatedAttrKey, isAuthenticated)
		ctx.SetUserValue(userAttrKey, user)
	}()

	store, err = sg.SessionProvider.Get(ctx)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to get session provider")

		return
	}

	userData, ok := store.Get(userAttrKey).([]byte)
	if !ok {
		return
	}

	err = json.Unmarshal(userData, &user)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to unmarshal user object")

		return
	}

	isLoggedIn = true

	sg.configurationMu.RLock()
	defer sg.configurationMu.RUnlock()

	for _, userID := range sg.Configuration.HTTP.UserAccess {
		if userID == int64(user.ID) {
			isAuthenticated = true

			return
		}
	}

	return store, err
}

func (sg *Sandwich) HandleRequest(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	path := ctx.Request.URI().PathOriginal()

	_, err := sg.authenticateValue(ctx)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

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

			if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
				ctx.Response.Reset()
				sg.DistHandler(ctx)
			}
		},
		fasthttp.CompressBrotliDefaultCompression,
		fasthttp.CompressDefaultCompression,
	)(ctx)
}

// /login: Handles logging in a user.
func (sg *Sandwich) LoginEndpoint(ctx *fasthttp.RequestCtx) {
	redirectURI := sg.Configuration.HTTP.OAuth.AuthCodeURL("")

	ctx.Redirect(redirectURI, fasthttp.StatusTemporaryRedirect)
}

// /callback: Handles oauth callback.
func (sg *Sandwich) CallbackEndpoint(ctx *fasthttp.RequestCtx) {
	var err error

	defer func() {
		if err != nil {
			ctx.Redirect("/", fasthttp.StatusTemporaryRedirect)
		}
	}()

	queryArgs := ctx.QueryArgs()

	code := gotils_strconv.B2S(queryArgs.Peek("code"))

	token, err := sg.Configuration.HTTP.OAuth.Exchange(ctx, code)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to exchange code")

		return
	}

	client := sg.Configuration.HTTP.OAuth.Client(ctx, token)

	resp, err := client.Get(discordUserMeEndpoint)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to fetch user")

		return
	}

	defer resp.Body.Close()

	user := discord.User{}

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to decode body")

		return
	}

	sg.Logger.Info().
		Str("username", user.Username+"#"+user.Discriminator).
		Int64("id", int64(user.ID)).Msg("New OAuth login")

	// Set user into session.

	store, err := sg.SessionProvider.Get(ctx)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to retrieve store")

		return
	}

	userData, err := json.Marshal(user)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to marshal user object")

		return
	}

	store.Set(userAttrKey, userData)

	err = sg.SessionProvider.Save(ctx, store)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to save session")

		return
	}

	ctx.Redirect("/", fasthttp.StatusTemporaryRedirect)
}

// /logout: Clears session.
func (sg *Sandwich) LogoutEndpoint(ctx *fasthttp.RequestCtx) {
	store, err := sg.SessionProvider.Get(ctx)
	if err != nil {
		return
	}

	store.Flush()

	err = sg.SessionProvider.Save(ctx, store)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to save session")

		return
	}

	ctx.Redirect("/", fasthttp.StatusTemporaryRedirect)
}

// /api/status: Returns managers, shardgroups and shard status.
func (sg *Sandwich) StatusEndpoint(ctx *fasthttp.RequestCtx) {
	sg.managersMu.RLock()
	managers := make([]*structs.StatusEndpointManager, 0, len(sg.Managers))

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

		managers = append(managers, statusManager)
	}
	sg.managersMu.RUnlock()

	writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
		Ok: true,
		Data: structs.StatusEndpointResponse{
			Managers: managers,
		},
	})
}

func (sg *Sandwich) UserEndpoint(ctx *fasthttp.RequestCtx) {
	user, _ := ctx.UserValue(userAttrKey).(discord.User)
	isLoggedIn, _ := ctx.UserValue(loggedInAttrKey).(bool)
	isAuthenticated, _ := ctx.UserValue(authenticatedAttrKey).(bool)

	writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
		Ok: true,
		Data: structs.UserResponse{
			User:            user,
			IsLoggedIn:      isLoggedIn,
			IsAuthenticated: isAuthenticated,
		},
	})
}

func (sg *Sandwich) DashboardGetEndpoint(ctx *fasthttp.RequestCtx) {
	sg.requireDiscordAuthentication(func(ctx *fasthttp.RequestCtx) {
		sg.managersMu.Lock()
		defer sg.managersMu.Unlock()

		sg.configurationMu.RLock()
		configuration := sg.Configuration
		defer sg.configurationMu.RUnlock()

		writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
			Ok: true,
			Data: structs.DashboardGetResponse{
				Configuration: configuration,
			},
		})
	})(ctx)
}

func (sg *Sandwich) ManagerUpdateEndpoint(ctx *fasthttp.RequestCtx) {
	sg.requireDiscordAuthentication(func(ctx *fasthttp.RequestCtx) {
		sg.managersMu.Lock()
		defer sg.managersMu.Unlock()

		managerConfiguration := ManagerConfiguration{}

		err := json.Unmarshal(ctx.PostBody(), &managerConfiguration)
		if err != nil {
			writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})

			return
		}

		manager, ok := sg.Managers[managerConfiguration.Identifier]
		if !ok {
			writeResponse(ctx, fasthttp.StatusBadRequest, structs.BaseRestResponse{
				Ok:    false,
				Error: ErrNoManagerPresent.Error(),
			})

			return
		}

		manager.configurationMu.Lock()
		manager.Configuration = &managerConfiguration
		manager.configurationMu.Unlock()

		err = manager.Initialize()
		if err != nil {
			writeResponse(ctx, fasthttp.StatusBadRequest, structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})

			return
		}

		manager.configurationMu.RLock()
		manager.Client = NewClient(manager.Configuration.Token)
		manager.configurationMu.RUnlock()

		manager.Sandwich.configurationMu.Lock()
		for _, configurationManager := range manager.Sandwich.Configuration.Managers {
			if managerConfiguration.Identifier == configurationManager.Identifier {
				configurationManager = manager.Configuration
			}
		}
		manager.Sandwich.configurationMu.Unlock()

		err = manager.Sandwich.SaveConfiguration(manager.Sandwich.Configuration, manager.Sandwich.ConfigurationLocation)
		if err != nil {
			writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})

			return
		}

		writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
			Ok:   true,
			Data: "Changes applied. You may need to make a new shard group to apply changes",
		})
	})(ctx)
}
