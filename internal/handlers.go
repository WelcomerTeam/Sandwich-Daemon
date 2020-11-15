package gateway

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"sync/atomic"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/hashicorp/go-uuid"
)

const forbiddenMessage = "You are not elevated"

var colours = [][]string{
	{"rgba(149, 165, 165, 0.5)", "#7E8C8D"},
	{"rgba(236, 240, 241, 0.5)", "#BEC3C7"},
	{"rgba(232, 76, 61, 0.5)", "#C1392B"},
	{"rgba(231, 126, 35, 0.5)", "#D25400"},
	{"rgba(241, 196, 15, 0.5)", "#F39C11"},
	{"rgba(52, 73, 94, 0.5)", "#2D3E50"},
	{"rgba(155, 88, 181, 0.5)", "#8F44AD"},
	{"rgba(53, 152, 219, 0.5)", "#2A80B9"},
	{"rgba(45, 204, 112, 0.5)", "#27AE61"},
	{"rgba(27, 188, 155, 0.5)", "#16A086"},
}

func passResponse(rw http.ResponseWriter, data interface{}, success bool, status int) {
	var resp []byte
	var err error
	if success {
		resp, err = json.Marshal(structs.BaseResponse{
			Success: true,
			Data:    data,
		})
	} else {
		resp, err = json.Marshal(structs.BaseResponse{
			Success: false,
			Error:   data.(string),
		})
	}

	if err != nil {
		resp, _ := json.Marshal(structs.BaseResponse{
			Success: false,
			Error:   err.Error(),
		})
		http.Error(rw, string(resp), http.StatusInternalServerError)
		return
	}

	if success {
		rw.WriteHeader(status)
		rw.Write(resp)
	} else {
		http.Error(rw, string(resp), status)
	}
	return
}

// LogoutHandler handles clearing a user session
func LogoutHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		defer session.Save(r, rw)

		session.Values = make(map[interface{}]interface{})
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
	}
}

// LoginHandler handles CSRF and AuthCode redirection
func LoginHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		defer session.Save(r, rw)

		// Create a simple CSRF string to verify clients and 500 if we
		// cannot generate one.
		csrfString, err := uuid.GenerateUUID()
		if err != nil {
			http.Error(rw, "Internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Store the CSRF in the session then redirect the user to the
		// OAuth page.
		session.Values["oauth_csrf"] = csrfString

		url := sg.Configuration.OAuth.AuthCodeURL(csrfString)
		http.Redirect(rw, r, url, http.StatusTemporaryRedirect)
	}
}

// OAuthCallbackHandler handles authenticating discord OAuth and creating
// a user profile if necessary
func OAuthCallbackHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		defer session.Save(r, rw)

		urlQuery := r.URL.Query()
		ctx := context.Background()

		// Validate the CSRF in the session and in the HTTP request.
		// If there is no CSRF in the session it is likely our fault :)
		_csrfString := urlQuery.Get("state")
		csrfString, ok := session.Values["oauth_csrf"].(string)
		if !ok {
			// http.Error(rw, "Missing CSRF state", http.StatusInternalServerError)
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		if _csrfString != csrfString {
			// http.Error(rw, "Mismatched CSRF states", http.StatusUnauthorized)
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Just to be sure, remove the CSRF after we have compared the CSRF
		delete(session.Values, "oauth_csrf")

		// Create an OAuth exchange with the code we were given.
		code := urlQuery.Get("code")
		token, err := sg.Configuration.OAuth.Exchange(ctx, code)
		if err != nil {
			// http.Error(rw, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Create a client with our exchanged token and retrieve a user.
		client := sg.Configuration.OAuth.Client(ctx, token)
		resp, err := client.Get(discordUsersMe)
		if err != nil {
			// http.Error(rw, err.Error(), http.StatusInternalServerError)
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			// http.Error(rw, err.Error(), http.StatusInternalServerError)
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		discordUserResponse := &structs.DiscordUser{}
		err = json.Unmarshal(body, &discordUserResponse)
		if err != nil {
			// http.Error(rw, err.Error(), http.StatusInternalServerError)
			http.Redirect(rw, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		session.Values["user"] = body

		// Once the user has logged in, send them back to the home page.
		http.Redirect(rw, r, "/", http.StatusTemporaryRedirect)
	}
}

// APIMeHandler handles the /api/me request which returns the user
// object and if they are elevated for the dashboard
func APIMeHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		defer session.Save(r, rw)

		// Authenticate the user
		auth, user := sg.AuthenticateSession(session)

		passResponse(rw, structs.APIMe{
			Authenticated: auth,
			User:          user,
		}, true, http.StatusOK)
	}
}

// APIStatusHandler handles the /api/status request which does not
// require elevation and provides basic information
func APIStatusHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		guildCounts := make(map[string]int64)
		now := time.Now().UTC()

		_result := structs.APIStatusResult{
			Managers: make([]structs.APIStatusManager, 0, len(sg.Managers)),
			Uptime:   now.Sub(sg.Start).Round(time.Millisecond).Milliseconds(),
		}

		for _, manager := range sg.Managers {
			guildCount, ok := guildCounts[manager.Configuration.Caching.RedisPrefix]
			if !ok {
				guildCount, err := sg.RedisClient.HLen(context.Background(), manager.CreateKey("guilds")).Result()
				if err != nil {
					passResponse(rw, err.Error(), false, http.StatusInternalServerError)
					return
				}
				guildCounts[manager.Configuration.Caching.RedisPrefix] = guildCount
			}
			_manager := structs.APIStatusManager{
				DisplayName: manager.Configuration.DisplayName,
				Guilds:      guildCount,
				ShardGroups: make([]structs.APIStatusShardGroup, 0, len(manager.ShardGroups)),
			}

			for _, shardgroup := range manager.ShardGroups {
				shardgroup.StatusMu.RLock()
				_shardgroup := structs.APIStatusShardGroup{
					ID:     shardgroup.ID,
					Status: shardgroup.Status,
					Shards: make([]structs.APIStatusShard, 0, len(shardgroup.Shards)),
				}
				shardgroup.StatusMu.RUnlock()

				shardgroup.ShardsMu.RLock()
				for _, shard := range shardgroup.Shards {
					shard.StatusMu.RLock()
					_shard := structs.APIStatusShard{
						Status:  shard.Status,
						Latency: shard.Latency(),
						Uptime:  now.Sub(shard.Start).Round(time.Millisecond).Milliseconds(),
					}
					shard.StatusMu.RUnlock()
					_shardgroup.Shards = append(_shardgroup.Shards, _shard)
				}
				shardgroup.ShardsMu.RUnlock()
				_manager.ShardGroups = append(_manager.ShardGroups, _shardgroup)
			}
			_result.Managers = append(_result.Managers, _manager)
		}

		passResponse(rw, _result, true, http.StatusOK)
	}
}

// ConstructAnalytics returns a LineChart struct based off of manager analytics
func (sg *Sandwich) ConstructAnalytics() structs.LineChart {
	datasets := make([]structs.Dataset, 0, len(sg.Managers))

	// Create and sort x axis keys
	mankeys := make([]string, 0, len(sg.Managers))
	for key := range sg.Managers {
		mankeys = append(mankeys, key)
	}
	sort.Strings(mankeys)

	for i, ident := range mankeys {
		mg := sg.Managers[ident]
		if mg.Analytics == nil {
			continue
		}

		mg.Analytics.RLock()
		data := make([]interface{}, 0, len(mg.Analytics.Samples))

		for _, sample := range mg.Analytics.Samples {
			data = append(data, structs.DataStamp{Time: sample.StoredAt, Value: sample.Value})
		}
		mg.Analytics.RUnlock()

		colour := colours[i%len(colours)]
		datasets = append(datasets, structs.Dataset{
			Label:            mg.Configuration.DisplayName,
			BackgroundColour: colour[0],
			BorderColour:     colour[1],
			Data:             data,
		})
	}

	return structs.LineChart{
		Datasets: datasets,
	}
}

// APIAnalyticsHandler handles the /api/analytics request which
// requires elevation
func APIAnalyticsHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		if auth, _ := sg.AuthenticateSession(session); !auth {
			passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
			return
		}

		managers := make([]structs.ManagerInformation, 0, len(sg.Managers))
		guildCounts := make(map[string]int64, 0)

		for _, manager := range sg.Managers {
			manager.ConfigurationMu.RLock()

			statuses := make(map[int32]structs.ShardGroupStatus)

			manager.ShardGroupsMu.RLock()
			for i, sg := range manager.ShardGroups {
				statuses[i] = sg.Status
			}
			manager.ShardGroupsMu.RUnlock()

			_guildCount, ok := guildCounts[manager.Configuration.Caching.RedisPrefix]
			if !ok {
				guildCount, err := sg.RedisClient.HLen(context.Background(), manager.CreateKey("guilds")).Result()
				if err != nil {
					passResponse(rw, err.Error(), false, http.StatusInternalServerError)
					return
				}
				guildCounts[manager.Configuration.Caching.RedisPrefix] = guildCount
			}

			_manager := structs.ManagerInformation{
				Name:      manager.Configuration.DisplayName,
				Guilds:    _guildCount,
				Status:    statuses,
				AutoStart: manager.Configuration.AutoStart,
			}
			manager.ConfigurationMu.RUnlock()
			managers = append(managers, _manager)
		}

		now := time.Now()
		guildCount := int64(0)
		for _, count := range guildCounts {
			guildCount += count
		}

		_result := structs.APIAnalyticsResult{
			Graph:    sg.ConstructAnalytics(),
			Guilds:   guildCount,
			Uptime:   DurationTimestamp(now.Sub(sg.Start)),
			Events:   atomic.LoadInt64(sg.TotalEvents),
			Managers: managers,
		}

		passResponse(rw, _result, true, http.StatusOK)
	}
}

// APIManagersResult is the structure of the /api/managers endpoint
type APIManagersResult map[string]map[int32]*ShardGroup

// APIManagersHandler handles the /api/managers endpoint
func APIManagersHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		if auth, _ := sg.AuthenticateSession(session); !auth {
			passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
			return
		}

		_result := APIManagersResult{}
		sg.ManagersMu.RLock()
		for key, manager := range sg.Managers {
			_result[key] = manager.ShardGroups
		}
		sg.ManagersMu.RUnlock()

		passResponse(rw, _result, true, http.StatusOK)
	}
}

// APIConfigurationHandler handles the /api/configuration endpoint
func APIConfigurationHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		if auth, _ := sg.AuthenticateSession(session); !auth {
			passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
			return
		}

		sg.ConfigurationMu.RLock()
		defer sg.ConfigurationMu.RUnlock()

		pl := structs.APIConfigurationResponse{
			Start:             sg.Start,
			RestTunnelEnabled: sg.RestTunnelEnabled.IsSet(),
		}

		sg.ConfigurationMu.RLock()
		pl.Configuration = sg.Configuration
		sg.ConfigurationMu.RUnlock()

		pl.Managers = make(map[string]structs.APIConfigurationResponseManager, 0)

		sg.ManagersMu.RLock()
		for managerID, manager := range sg.Managers {
			mg := structs.APIConfigurationResponseManager{}

			manager.ConfigurationMu.RLock()
			mg.Configuration = manager.Configuration
			manager.ConfigurationMu.RUnlock()

			manager.GatewayMu.RLock()
			mg.Gateway = manager.Gateway
			manager.GatewayMu.RUnlock()

			manager.ErrorMu.RLock()
			mg.Error = manager.Error
			manager.ErrorMu.RUnlock()

			mg.ShardGroups = make(map[int32]structs.APIConfigurationResponseShardGroup, 0)

			manager.ShardGroupsMu.RLock()
			for shardgroupID, shardgroup := range manager.ShardGroups {
				shg := structs.APIConfigurationResponseShardGroup{
					Start:      shardgroup.Start,
					ID:         shardgroup.ID,
					ShardCount: shardgroup.ShardCount,
					ShardIDs:   shardgroup.ShardIDs,
					WaitingFor: atomic.LoadInt32(shardgroup.WaitingFor),
				}

				shardgroup.StatusMu.RLock()
				shg.Status = shardgroup.Status
				shardgroup.StatusMu.RUnlock()

				shardgroup.ErrorMu.RLock()
				shg.Error = shardgroup.Error
				shardgroup.ErrorMu.RUnlock()

				shg.Shards = make(map[int]interface{}, 0)

				shardgroup.ShardsMu.RLock()
				for shardID, shard := range shardgroup.Shards {
					shd := structs.APIConfigurationResponseShard{
						ShardID:              shard.ShardID,
						User:                 shard.User,
						HeartbeatInterval:    shard.HeartbeatInterval,
						MaxHeartbeatFailures: shard.MaxHeartbeatFailures,
						Start:                shard.Start,
						Retries:              atomic.LoadInt32(shard.Retries),
					}

					shard.StatusMu.RLock()
					shd.Status = shard.Status
					shard.StatusMu.RUnlock()

					shard.LastHeartbeatMu.RLock()
					shd.LastHeartbeatAck = shard.LastHeartbeatAck
					shd.LastHeartbeatSent = shard.LastHeartbeatSent
					shard.LastHeartbeatMu.RUnlock()

					shg.Shards[shardID] = shd
				}
				shardgroup.ShardsMu.RUnlock()

				mg.ShardGroups[shardgroupID] = shg
			}
			manager.ShardGroupsMu.RUnlock()

			pl.Managers[managerID] = mg
		}
		sg.ManagersMu.RUnlock()

		passResponse(rw, pl, true, http.StatusOK)
	}
}

// APIRestTunnelHandler handles the /api/resttunnel endpoint
func APIRestTunnelHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		if auth, _ := sg.AuthenticateSession(session); !auth {
			passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
			return
		}

		if sg.RestTunnelEnabled.IsNotSet() {
			passResponse(rw, "RestTunnel is not enabled", true, http.StatusOK)
			return
		}

		_url, err := url.Parse(sg.Configuration.RestTunnel.URL)
		if err != nil {
			passResponse(rw, err.Error(), false, http.StatusInternalServerError)
			return
		}

		resp, err := http.Get(_url.Scheme + "://" + _url.Host + "/resttunnel/analytics")
		if err != nil {
			passResponse(rw, err.Error(), false, http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			passResponse(rw, err.Error(), false, http.StatusInternalServerError)
			return
		}

		// We want to write directly as its a proxied request
		rw.WriteHeader(resp.StatusCode)
		rw.Write(body)
	}
}

// APIRPCHandler handles the /api/rpc endpoint
func APIRPCHandler(sg *Sandwich) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, _ := sg.Store.Get(r, sessionName)
		if auth, _ := sg.AuthenticateSession(session); !auth {
			passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			passResponse(rw, err.Error(), false, http.StatusInternalServerError)
			return
		}

		RPCMessage := structs.RPCRequest{}
		err = json.Unmarshal(body, &RPCMessage)
		if err != nil {
			passResponse(rw, "Invalid payload sent", false, http.StatusBadRequest)
			return
		}

		ok := executeRequest(sg, RPCMessage, rw)
		if !ok {
			passResponse(rw, fmt.Sprintf("Unknown method: %s", RPCMessage.Method), false, http.StatusBadRequest)
			return
		}

		return
	}
}

// session, _ := sg.Store.Get(r, sessionName)
// if auth, _ := sg.AuthenticateSession(session); !auth {
// 	passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
// 	return
// }

// return func(rw http.ResponseWriter, r *http.Request) {
// 	session, _ := sg.Store.Get(r, sessionName)
// 	if auth, _ := sg.AuthenticateSession(session); !auth {
// 		passResponse(rw, forbiddenMessage, false, http.StatusForbidden)
// 		return
// 	}

// 	passResponse(rw, "OK", true, http.StatusOK)
// }
