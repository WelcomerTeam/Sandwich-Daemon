package internal

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/WelcomerTeam/Sandwich-Daemon/discord"
	sandwich_structs "github.com/WelcomerTeam/Sandwich-Daemon/internal/structs"
	"github.com/WelcomerTeam/Sandwich-Daemon/sandwichjson"
	"github.com/fasthttp/router"
	"github.com/fasthttp/session/v2"
	"github.com/rs/zerolog"
	gotils_strconv "github.com/savsgio/gotils/strconv"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/pprofhandler"
)

var (
	ErrUserMissingAccess = errors.New("you are missing access")
	ErrUserNotLoggedIn   = errors.New("you are not logged in")

	discordUserMeEndpoint = "https://discord.com/api/users/@me"

	EnableDistHandling = true
	DistPath           = "sandwich/dist"

	loggedInAttrKey      = "isLoggedIn"
	authenticatedAttrKey = "isAuthenticated"
	userAttrKey          = "user"

	StatusCacheDuration = time.Second * 30
)

func (sg *Sandwich) NewRestRouter() (routerHandler fasthttp.RequestHandler, fsHandler fasthttp.RequestHandler) {
	r := router.New()

	// OAuth2
	r.GET("/login", sg.LoginEndpoint)
	r.GET("/logout", sg.LogoutEndpoint)
	r.GET("/callback", sg.CallbackEndpoint)

	// Anonymous routes
	r.GET("/api/status", sg.StatusEndpoint)
	r.GET("/api/user", sg.UserEndpoint)

	// State route
	r.GET("/api/state", sg.internalEndpoint(sg.StateEndpoint))
	r.POST("/api/state", sg.internalEndpoint(sg.StateEndpoint))

	// Sandwich related endpoints
	r.GET("/api/sandwich", sg.requireDiscordAuthentication(sg.SandwichGetEndpoint))
	r.PATCH("/api/sandwich", sg.requireDiscordAuthentication(sg.SandwichUpdateEndpoint))
	r.GET("/debug/pprof/{profile:*}", sg.requireDiscordAuthentication(pprofhandler.PprofHandler))
	r.POST("/api/manager", sg.requireDiscordAuthentication(sg.ManagerCreateEndpoint))
	r.POST("/api/manager/initialize", sg.requireDiscordAuthentication(sg.ManagerInitializeEndpoint))
	r.PATCH("/api/manager", sg.requireDiscordAuthentication(sg.ManagerUpdateEndpoint))
	r.DELETE("/api/manager", sg.requireDiscordAuthentication(sg.ManagerDeleteEndpoint))

	r.POST("/api/manager/shardgroup", sg.requireDiscordAuthentication(sg.ShardGroupCreateEndpoint))
	r.DELETE("/api/manager/shardgroup", sg.requireDiscordAuthentication(sg.ShardGroupStopEndpoint))

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

// internalEndpoint wraps a RequestHandler and blocks requests made to
// such endpoints if the X-Forwarded-For header is set
func (sg *Sandwich) internalEndpoint(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		if len(ctx.Request.Header.Peek("X-Forwarded-For")) > 0 {
			writeResponse(ctx, fasthttp.StatusForbidden, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: "Forbidden",
			})

			return
		}

		h(ctx)
	})
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
			writeResponse(ctx, fasthttp.StatusUnauthorized, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: ErrUserNotLoggedIn.Error(),
			})

			return
		}

		if !isAuthenticated && sg.Options.HTTPEnabled {
			writeResponse(ctx, fasthttp.StatusForbidden, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: ErrUserMissingAccess.Error(),
			})

			return
		}

		h(ctx)
	})
}

func writeResponse(ctx *fasthttp.RequestCtx, statusCode int, i interface{}) {
	body, err := sandwichjson.Marshal(i)
	if err == nil {
		_, _ = ctx.Write(body)
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

	err = sandwichjson.Unmarshal(userData, &user)
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

	return store, nil
}

func (sg *Sandwich) HandleRequest(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	path := ctx.Request.URI().PathOriginal()

	_, err := sg.authenticateValue(ctx)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
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

	err = sandwichjson.UnmarshalReader(resp.Body, &user)

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

	userData, err := sandwichjson.Marshal(user)
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
	managers := make([]*sandwich_structs.StatusEndpointManager, 0, sg.Managers.Count())
	unsortedManagers := make(map[string]*sandwich_structs.StatusEndpointManager)

	manager := gotils_strconv.B2S(ctx.QueryArgs().Peek("manager"))

	if manager == "" {
		statusData := sg.statusCache.Result(StatusCacheDuration, func() interface{} {
			sg.Managers.Range(func(key string, manager *Manager) bool {
				manager.configurationMu.RLock()
				friendlyName := manager.Configuration.FriendlyName
				keyName := manager.Configuration.FriendlyName + ":" + manager.Configuration.Identifier
				manager.configurationMu.RUnlock()

				unsortedManagers[keyName] = &sandwich_structs.StatusEndpointManager{
					DisplayName: friendlyName,
					ShardGroups: getManagerShardGroupStatus(manager),
				}

				return false
			})

			// Sort manager list by friendly name.

			managerList := []string{}

			for managerName := range unsortedManagers {
				managerList = append(managerList, managerName)
			}

			sort.Strings(managerList)

			for _, keyName := range managerList {
				managers = append(managers, unsortedManagers[keyName])
			}

			return sandwich_structs.StatusEndpointResponse{
				Uptime:   int(time.Since(sg.StartTime).Seconds()),
				Managers: managers,
			}
		})

		writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
			Ok:   true,
			Data: statusData,
		})
	} else {
		manager, ok := sg.Managers.Load(manager)
		if !ok {
			writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: ErrNoManagerPresent.Error(),
			})

			return
		}

		manager.configurationMu.RLock()
		friendlyName := manager.Configuration.FriendlyName
		manager.configurationMu.RUnlock()

		writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
			Ok: true,
			Data: &sandwich_structs.StatusEndpointManager{
				DisplayName: friendlyName,
				ShardGroups: getManagerShardGroupStatus(manager),
			},
		})
	}
}

// /api/state?col={collection}&id={id}: Returns data from the sandwich state
func (sg *Sandwich) StateEndpoint(ctx *fasthttp.RequestCtx) {
	col := ctx.QueryArgs().Peek("col")
	id := ctx.QueryArgs().Peek("id")

	if len(col) == 0 || len(id) == 0 {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: "Missing col or id",
		})

		return
	}

	switch gotils_strconv.B2S(col) {
	case "users":
		idInt64, err := strconv.ParseInt(gotils_strconv.B2S(id), 10, 64)

		if err != nil {
			writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})

			return
		}

		if ctx.IsGet() {
			user, ok := sg.State.GetUser(discord.Snowflake(idInt64))

			if !ok {
				writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
					Ok:    false,
					Error: "User not found",
				})

				return
			}

			writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
				Ok:   true,
				Data: *user,
			})
		} else {
			// Read request body as a user
			var user discord.User

			err := sandwichjson.Unmarshal(ctx.PostBody(), &user)

			if err != nil {
				writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
					Ok:    false,
					Error: err.Error(),
				})

				return
			}

			sg.State.Users.Store(user.ID, *sg.State.UserToState(&user))
		}
	case "members":
		idInt64, err := strconv.ParseInt(gotils_strconv.B2S(id), 10, 64)

		if err != nil {
			writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})

			return
		}

		guildId := ctx.QueryArgs().Peek("guild_id")

		if len(guildId) == 0 {
			writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: "Missing guild_id",
			})

			return
		}

		guildIdInt64, err := strconv.ParseInt(gotils_strconv.B2S(guildId), 10, 64)

		if err != nil {
			writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})
			return
		}

		if ctx.IsGet() {
			member, ok := sg.State.GetGuildMember(discord.Snowflake(guildIdInt64), discord.Snowflake(idInt64))

			if !ok {
				writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
					Ok:    false,
					Error: "Member not found",
				})

				return
			}

			sg.Logger.Info().Any("member", member).Msg("Getting member")

			writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
				Ok:   true,
				Data: *member,
			})
		} else {
			// Read request body as a member
			var member discord.GuildMember

			err := sandwichjson.Unmarshal(ctx.PostBody(), &member)

			if err != nil {
				writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
					Ok:    false,
					Error: err.Error(),
				})

				return
			}

			sg.Logger.Info().Any("member", member).Msg("Adding member")

			sg.State.SetGuildMember(
				&StateCtx{
					CacheUsers:   true,
					CacheMembers: true,
					Shard: &Shard{
						Manager: &Manager{},
					},
				},
				discord.Snowflake(guildIdInt64),
				&member,
			)

			writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
				Ok:   true,
				Data: nil,
			})
		}
	case "guilds":
		idInt64, err := strconv.ParseInt(gotils_strconv.B2S(id), 10, 64)

		if err != nil {
			writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
				Ok:    false,
				Error: err.Error(),
			})

			return
		}

		if ctx.IsGet() {
			guild, ok := sg.State.GetGuild(discord.Snowflake(idInt64))

			if !ok {
				writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
					Ok:    false,
					Error: "Guild not found",
				})

				return
			}

			writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
				Ok:   true,
				Data: *guild,
			})
		} else {
			// Read request body as a guild
			var guild discord.Guild

			err := sandwichjson.Unmarshal(ctx.PostBody(), &guild)

			if err != nil {
				writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
					Ok:    false,
					Error: err.Error(),
				})

				return
			}

			sg.State.Guilds.Store(guild.ID, &guild)

			writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
				Ok:   true,
				Data: nil,
			})
		}
	}
}

func getManagerShardGroupStatus(manager *Manager) (shardGroups []*sandwich_structs.StatusEndpointShardGroup) {
	sortedShardGroupIDs := make([]int, 0)

	manager.ShardGroups.Range(func(shardGroupID int32, shardGroup *ShardGroup) bool {
		shardGroup.statusMu.RLock()
		shardGroupStatus := shardGroup.Status
		shardGroup.statusMu.RUnlock()

		if shardGroupStatus != sandwich_structs.ShardGroupStatusClosed {
			sortedShardGroupIDs = append(sortedShardGroupIDs, int(shardGroupID))
		}
		return false
	})

	sort.Ints(sortedShardGroupIDs)

	for _, _shardGroupID := range sortedShardGroupIDs {
		shardGroupID := int32(_shardGroupID)
		shardGroup, ok := manager.ShardGroups.Load(shardGroupID)

		if !ok {
			continue
		}

		statusShardGroup := &sandwich_structs.StatusEndpointShardGroup{
			ShardGroupID: shardGroup.ID,
			Shards:       make([][6]int, 0, shardGroup.Shards.Count()),
			Status:       shardGroup.Status,
			Uptime:       int(time.Since(shardGroup.Start.Load()).Seconds()),
		}

		sortedShardIDs := make([]int, 0, shardGroup.Shards.Count())
		shardGroup.Shards.Range(func(i int32, shardId *Shard) bool {
			sortedShardIDs = append(sortedShardIDs, int(i))
			return false
		})

		sort.Ints(sortedShardIDs)

		for _, intShardID := range sortedShardIDs {
			shardID := int32(intShardID)

			shard, ok := shardGroup.Shards.Load(shardID)

			if !ok {
				manager.Logger.Error().Int32("shardID", shardID).Msg("Failed to load shard [getManagerShardGroupStatus]")
				continue
			}

			shard.statusMu.RLock()
			shardStatus := shard.Status
			shard.statusMu.RUnlock()

			statusShardGroup.Shards = append(statusShardGroup.Shards, [6]int{
				int(shard.ShardID),
				int(shardStatus),
				int(shard.LastHeartbeatAck.Load().Sub(shard.LastHeartbeatSent.Load()).Milliseconds()),
				shard.Guilds.Count(),
				int(time.Since(shard.Start.Load()).Seconds()),
				int(time.Since(shard.Init.Load()).Seconds()),
			})
		}

		shardGroups = append(shardGroups, statusShardGroup)
	}

	return shardGroups
}

func (sg *Sandwich) UserEndpoint(ctx *fasthttp.RequestCtx) {
	user, _ := ctx.UserValue(userAttrKey).(discord.User)
	isLoggedIn, _ := ctx.UserValue(loggedInAttrKey).(bool)
	isAuthenticated, _ := ctx.UserValue(authenticatedAttrKey).(bool)

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok: true,
		Data: sandwich_structs.UserResponse{
			User:            user,
			IsLoggedIn:      isLoggedIn,
			IsAuthenticated: isAuthenticated,
		},
	})
}

func (sg *Sandwich) SandwichGetEndpoint(ctx *fasthttp.RequestCtx) {
	sg.configurationMu.RLock()
	configuration := sg.Configuration
	sg.configurationMu.RUnlock()

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok: true,
		Data: sandwich_structs.DashboardGetResponse{
			Configuration: configuration,
		},
	})
}

func (sg *Sandwich) SandwichUpdateEndpoint(ctx *fasthttp.RequestCtx) {
	sandwichConfiguration := SandwichConfiguration{}

	err := sandwichjson.Unmarshal(ctx.PostBody(), &sandwichConfiguration)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
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
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	go sg.PublishSimpleWebhook(
		"Updated sandwich config",
		"",
		fmt.Sprintf(
			"User: %s",
			ctx.UserValue(userAttrKey).(discord.User).Username,
		),
		EmbedColourSandwich,
	)

	// Save event blacklist/producer blacklist to managers.
	go func() {
		for _, manager := range sandwichConfiguration.Managers {
			m, ok := sg.Managers.Load(manager.Identifier)

			if !ok {
				continue
			}

			if manager.Events.EventBlacklist != nil {
				m.eventBlacklistMu.Lock()
				m.eventBlacklist = manager.Events.EventBlacklist
				m.eventBlacklistMu.Unlock()
			}

			if manager.Events.ProduceBlacklist != nil {
				m.produceBlacklistMu.Lock()
				m.produceBlacklist = manager.Events.ProduceBlacklist
				m.produceBlacklistMu.Unlock()
			}

			m.metadataMu.Lock()
			m.metadata = &sandwich_structs.SandwichMetadata{
				Version:       VERSION,
				Identifier:    manager.Identifier,
				Application:   m.Identifier.Load(),
				ApplicationID: discord.Snowflake(m.UserID.Load()),
			}
			m.metadataMu.Unlock()

			/*if manager.Bot.DefaultPresence.Status != "" {
				// Update presence.
				m.ShardGroups.Range(func(shardGroupID int32, shardGroup *ShardGroup) bool {
					shardGroup.Shards.Range(func(shardID int32, shard *Shard) bool {
						shard.UpdatePresence(ctx, &manager.Bot.DefaultPresence)
						return false
					})

					return false
				})
			}*/
		}

		sg.Logger.Info().Msg("Updated event blacklist and producer blacklist")
	}()

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: "Changes applied.",
	})
}

func (sg *Sandwich) ManagerCreateEndpoint(ctx *fasthttp.RequestCtx) {
	createManagerArguments := sandwich_structs.CreateManagerArguments{}

	err := sandwichjson.Unmarshal(ctx.PostBody(), &createManagerArguments)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	_, ok := sg.Managers.Load(createManagerArguments.Identifier)

	if ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: ErrDuplicateManagerPresent.Error(),
		})

		return
	}

	defaultConfiguration := ManagerConfiguration{
		Identifier:         createManagerArguments.Identifier,
		ProducerIdentifier: createManagerArguments.ProducerIdentifier,
		FriendlyName:       createManagerArguments.FriendlyName,
		Token:              createManagerArguments.Token,
		Messaging: struct {
			ClientName      string "json:\"client_name\" yaml:\"client_name\""
			ChannelName     string "json:\"channel_name\" yaml:\"channel_name\""
			UseRandomSuffix bool   "json:\"use_random_suffix\" yaml:\"use_random_suffix\""
		}{
			ClientName:      createManagerArguments.ClientName,
			ChannelName:     createManagerArguments.ChannelName,
			UseRandomSuffix: true,
		},
	}

	manager := sg.NewManager(&defaultConfiguration)

	sg.Managers.Store(createManagerArguments.Identifier, manager)

	sg.configurationMu.Lock()
	sg.Configuration.Managers = append(sg.Configuration.Managers, &defaultConfiguration)
	sg.configurationMu.Unlock()

	sg.configurationMu.RLock()
	defer sg.configurationMu.RUnlock()

	err = sg.SaveConfiguration(sg.Configuration, sg.ConfigurationLocation)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	go sg.PublishSimpleWebhook(
		fmt.Sprintf(
			"Created new manager `%s`",
			defaultConfiguration.Identifier,
		),
		"",
		fmt.Sprintf(
			"User: %s",
			ctx.UserValue(userAttrKey).(discord.User).Username,
		),
		EmbedColourSandwich,
	)

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: fmt.Sprintf("Manager '%s' created", createManagerArguments.Identifier),
	})
}

func (sg *Sandwich) ManagerInitializeEndpoint(ctx *fasthttp.RequestCtx) {
	managerName := gotils_strconv.B2S(ctx.QueryArgs().Peek("manager"))

	manager, ok := sg.Managers.Load(managerName)

	if !ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: ErrNoManagerPresent.Error(),
		})

		return
	}

	forceRestartProducers := gotils_strconv.B2S(ctx.QueryArgs().Peek("forceRestartProducers")) == "true"

	err := manager.Initialize(forceRestartProducers)
	if err != nil {
		writeResponse(ctx, http.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: "Manager initialized, you may start up shardgroups now",
	})
}

func (sg *Sandwich) ManagerUpdateEndpoint(ctx *fasthttp.RequestCtx) {
	managerConfiguration := ManagerConfiguration{}

	err := sandwichjson.Unmarshal(ctx.PostBody(), &managerConfiguration)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	manager, ok := sg.Managers.Load(managerConfiguration.Identifier)

	if !ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: ErrNoManagerPresent.Error(),
		})

		return
	}

	manager.configurationMu.Lock()
	manager.Configuration = &managerConfiguration
	manager.configurationMu.Unlock()

	manager.clientMu.Lock()
	manager.Client = NewClient(baseURL, manager.Configuration.Token)
	manager.clientMu.Unlock()

	forceRestartProducers := gotils_strconv.B2S(ctx.QueryArgs().Peek("forceRestartProducers")) == "true"

	err = manager.Initialize(forceRestartProducers)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	sg.configurationMu.Lock()
	defer sg.configurationMu.Unlock()

	managers := make([]*ManagerConfiguration, 0)

	for _, manager := range sg.Configuration.Managers {
		if manager.Identifier != managerConfiguration.Identifier {
			managers = append(managers, manager)
		}
	}

	managers = append(managers, &managerConfiguration)

	sg.Configuration.Managers = managers

	err = sg.SaveConfiguration(sg.Configuration, sg.ConfigurationLocation)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	go sg.PublishSimpleWebhook(
		fmt.Sprintf(
			"Updated manager `%s`",
			managerConfiguration.Identifier,
		),
		"",
		fmt.Sprintf(
			"User: %s",
			ctx.UserValue(userAttrKey).(discord.User).Username,
		),
		EmbedColourSandwich,
	)

	// Save event blacklist/producer blacklist to managers.
	go func() {
		m, ok := sg.Managers.Load(managerConfiguration.Identifier)

		if !ok {
			return
		}

		if managerConfiguration.Events.EventBlacklist != nil {
			m.eventBlacklistMu.Lock()
			m.eventBlacklist = managerConfiguration.Events.EventBlacklist
			m.eventBlacklistMu.Unlock()
		}

		if managerConfiguration.Events.ProduceBlacklist != nil {
			m.produceBlacklistMu.Lock()
			m.produceBlacklist = managerConfiguration.Events.ProduceBlacklist
			m.produceBlacklistMu.Unlock()
		}

		/*if managerConfiguration.Bot.DefaultPresence.Status != "" {
			p := managerConfiguration.Bot.DefaultPresence
			// Update presence.
			m.ShardGroups.Range(func(shardGroupID int32, shardGroup *ShardGroup) bool {
				shardGroup.Shards.Range(func(shardID int32, shard *Shard) bool {
					shard.UpdatePresence(ctx, &p)
					return false
				})

				return false
			})
		}*/

		sg.Logger.Info().Msg("Updated event blacklist and producer blacklist")
	}()

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: "Changes applied. You may need to make a new shard group to apply changes",
	})
}

func (sg *Sandwich) ManagerDeleteEndpoint(ctx *fasthttp.RequestCtx) {
	managerName := gotils_strconv.B2S(ctx.QueryArgs().Peek("manager"))
	manager, ok := sg.Managers.Load(managerName)

	if !ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: ErrNoManagerPresent.Error(),
		})

		return
	}

	manager.Close()

	sg.Managers.Delete(managerName)

	sg.configurationMu.Lock()
	defer sg.configurationMu.Unlock()

	managers := make([]*ManagerConfiguration, 0)

	for _, manager := range sg.Configuration.Managers {
		if manager.Identifier != managerName {
			managers = append(managers, manager)
		}
	}

	sg.Configuration.Managers = managers

	err := sg.SaveConfiguration(sg.Configuration, sg.ConfigurationLocation)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	go sg.PublishSimpleWebhook(
		fmt.Sprintf(
			"Deleted manager `%s`",
			managerName,
		),
		"",
		fmt.Sprintf(
			"User: %s",
			ctx.UserValue(userAttrKey).(discord.User).Username,
		),
		EmbedColourSandwich,
	)

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: "Removed manager.",
	})
}

func (sg *Sandwich) ShardGroupCreateEndpoint(ctx *fasthttp.RequestCtx) {
	shardGroupArguments := sandwich_structs.CreateManagerShardGroupArguments{}

	err := sandwichjson.Unmarshal(ctx.PostBody(), &shardGroupArguments)
	if err != nil {
		writeResponse(ctx, fasthttp.StatusInternalServerError, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	manager, ok := sg.Managers.Load(shardGroupArguments.Identifier)

	if !ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: ErrNoManagerPresent.Error(),
		})

		return
	}

	shardIDs, shardCount := manager.getInitialShardCount(
		shardGroupArguments.ShardCount,
		shardGroupArguments.ShardIDs,
		shardGroupArguments.AutoSharded,
	)

	sg.Logger.Debug().
		Interface("shardIDs", shardIDs).Int32("shardCount", shardCount).
		Str("identifier", manager.Identifier.Load()).Msg("Creating new ShardGroup")

	shardGroup := manager.Scale(shardIDs, shardCount)

	_, err = shardGroup.Open()
	if err != nil {
		// Cleanup ShardGroups to remove failed ShardGroup.
		manager.ShardGroups.Delete(shardGroup.ID)

		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: err.Error(),
		})

		return
	}

	go sg.PublishSimpleWebhook(
		"Launched new shardgroup",
		fmt.Sprintf(
			"Shard count: `%d` - Shards: `%s`",
			shardGroupArguments.ShardCount,
			shardGroupArguments.ShardIDs,
		),
		fmt.Sprintf(
			"Manager: %s ShardGroup: %d User: %s",
			manager.Identifier.Load(),
			shardGroup.ID,
			ctx.UserValue(userAttrKey).(discord.User).Username,
		),
		EmbedColourSandwich,
	)

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: "ShardGroup successfully created",
	})
}

func (sg *Sandwich) ShardGroupStopEndpoint(ctx *fasthttp.RequestCtx) {
	managerName := gotils_strconv.B2S(ctx.QueryArgs().Peek("manager"))

	manager, ok := sg.Managers.Load(managerName)

	if !ok {
		writeResponse(ctx, fasthttp.StatusBadRequest, sandwich_structs.BaseRestResponse{
			Ok:    false,
			Error: ErrNoManagerPresent.Error(),
		})

		return
	}

	manager.Close()

	writeResponse(ctx, fasthttp.StatusOK, sandwich_structs.BaseRestResponse{
		Ok:   true,
		Data: "Manager shardgroups closed",
	})
}
