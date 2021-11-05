package internal

import (
	"net/http"
	"sort"
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
	r.GET("/api/dashboard", sg.requireDiscordAuthentication(sg.DashboardGetEndpoint))

	r.POST("/api/manager", sg.requireDiscordAuthentication(sg.ManagerUpdateEndpoint))
	r.POST("/api/manager/shardgroup", sg.requireDiscordAuthentication(sg.CreateManagerShardGroupEndpoint))
	r.POST("/api/sandwich", sg.requireDiscordAuthentication(sg.SandwichUpdateEndpoint))

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
		if userID == user.ID.String() {
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
	defer sg.managersMu.RUnlock()

	managers := make([]*structs.StatusEndpointManager, 0, len(sg.Managers))

	manager := gotils_strconv.B2S(ctx.QueryArgs().Peek("manager"))

	if manager == "" {
		for _, manager := range sg.Managers {
			manager.configurationMu.RLock()
			friendlyName := manager.Configuration.FriendlyName
			manager.configurationMu.RUnlock()

			managers = append(managers, &structs.StatusEndpointManager{
				DisplayName: friendlyName,
				ShardGroups: getManagerShardGroupStatus(manager),
			})
		}

		writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
			Ok: true,
			Data: structs.StatusEndpointResponse{
				Managers: managers,
			},
		})
	} else {
		manager, ok := sg.Managers[manager]
		if !ok {
			writeResponse(ctx, fasthttp.StatusBadRequest, structs.BaseRestResponse{
				Ok:    false,
				Error: ErrNoManagerPresent.Error(),
			})

			return
		}

		manager.configurationMu.RLock()
		friendlyName := manager.Configuration.FriendlyName
		manager.configurationMu.RUnlock()

		writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
			Ok: true,
			Data: &structs.StatusEndpointManager{
				DisplayName: friendlyName,
				ShardGroups: getManagerShardGroupStatus(manager),
			},
		})
	}
}

func getManagerShardGroupStatus(manager *Manager) (shardGroups []*structs.StatusEndpointShardGroup) {
	manager.shardGroupsMu.RLock()

	sortedShardGroupIDs := make([]int, 0)
	for shardGroupID, shardGroup := range manager.ShardGroups {
		shardGroup.statusMu.RLock()
		shardGroupStatus := shardGroup.Status
		shardGroup.statusMu.RUnlock()

		if shardGroupStatus != structs.ShardGroupStatusClosed {
			sortedShardGroupIDs = append(sortedShardGroupIDs, int(shardGroupID))
		}
	}

	sort.Ints(sortedShardGroupIDs)

	for _, shardGroupID := range sortedShardGroupIDs {
		shardGroup := manager.ShardGroups[int64(shardGroupID)]

		shardGroup.shardsMu.RLock()
		statusShardGroup := &structs.StatusEndpointShardGroup{
			ShardGroupID: int(shardGroup.ID),
			Shards:       make([][5]int, 0, len(shardGroup.Shards)),
			Status:       shardGroup.Status,
		}

		sortedShardIDs := make([]int, 0, len(shardGroup.Shards))
		for shardID := range shardGroup.Shards {
			sortedShardIDs = append(sortedShardIDs, shardID)
		}

		sort.Ints(sortedShardIDs)

		for _, shardID := range sortedShardIDs {
			shard := shardGroup.Shards[shardID]

			shard.statusMu.RLock()
			shardStatus := shard.Status
			shard.statusMu.RUnlock()

			shard.guildsMu.RLock()
			statusShardGroup.Shards = append(statusShardGroup.Shards, [5]int{
				shard.ShardID,
				int(shardStatus),
				int(shard.LastHeartbeatAck.Load().Sub(shard.LastHeartbeatSent.Load()).Milliseconds()),
				len(shard.Guilds),
				int(time.Since(shard.Start).Seconds()),
			})
			shard.guildsMu.RUnlock()
		}
		shardGroup.shardsMu.RUnlock()

		shardGroups = append(shardGroups, statusShardGroup)
	}
	manager.shardGroupsMu.RUnlock()

	return shardGroups
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
		sg.configurationMu.RLock()
		configuration := sg.Configuration
		sg.configurationMu.RUnlock()

		writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
			Ok: true,
			Data: structs.DashboardGetResponse{
				Configuration: configuration,
			},
		})
	})(ctx)
}

func (sg *Sandwich) SandwichUpdateEndpoint(ctx *fasthttp.RequestCtx) {
	sandwichConfiguration := SandwichConfiguration{}

	err := json.Unmarshal(ctx.PostBody(), &sandwichConfiguration)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	sg.configurationMu.Lock()
	sandwichConfiguration.Managers = sg.Configuration.Managers
	sg.Configuration = &sandwichConfiguration
	sg.configurationMu.Unlock()

	err = sg.SaveConfiguration(&sandwichConfiguration, sg.ConfigurationLocation)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
		Ok:   true,
		Data: "Changes applied.",
	})
}

func (sg *Sandwich) ManagerUpdateEndpoint(ctx *fasthttp.RequestCtx) {
	managerConfiguration := ManagerConfiguration{}

	err := json.Unmarshal(ctx.PostBody(), &managerConfiguration)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	sg.managersMu.RLock()
	manager, ok := sg.Managers[managerConfiguration.Identifier]
	defer sg.managersMu.RUnlock()

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

	sg.configurationMu.Lock()
	for _, configurationManager := range sg.Configuration.Managers {
		if managerConfiguration.Identifier == configurationManager.Identifier {
			configurationManager = manager.Configuration
		}
	}
	sg.configurationMu.Unlock()

	sg.configurationMu.RLock()
	defer sg.configurationMu.RUnlock()

	err = sg.SaveConfiguration(sg.Configuration, sg.ConfigurationLocation)
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
}

func (sg *Sandwich) CreateManagerShardGroupEndpoint(ctx *fasthttp.RequestCtx) {
	shardGroupArguments := structs.CreateManagerShardGroupArguments{}

	err := json.Unmarshal(ctx.PostBody(), &shardGroupArguments)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	sg.managersMu.RLock()
	manager, ok := sg.Managers[shardGroupArguments.Identifier]
	sg.managersMu.RUnlock()

	if !ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, structs.BaseRestResponse{
			Ok:    false,
			Error: ErrNoManagerPresent.Error(),
		})

		return
	}

	shardIDs, shardCount := manager.getInitialShardCount(
		shardGroupArguments.ShardCount,
		shardGroupArguments.ShardIDs,
	)

	shardGroup := manager.Scale(shardIDs, shardCount)

	_, err = shardGroup.Open()
	if err != nil {
		// Cleanup ShardGroups to remove failed ShardGroup.
		manager.shardGroupsMu.Lock()
		delete(manager.ShardGroups, shardGroup.ID)
		manager.shardGroupsMu.Unlock()

		writeResponse(ctx, fasthttp.StatusBadRequest, structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	writeResponse(ctx, fasthttp.StatusOK, structs.BaseRestResponse{
		Ok:   true,
		Data: "ShardGroup successfully created",
	})
}
