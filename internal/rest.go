package gateway

import (
	"net/http"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

const sessionName = "session"
const discordUsersMe = "https://discord.com/api/users/@me"
const discordRefreshDuration = time.Hour

// NewMethodRouter creates a new method router
func NewMethodRouter() *MethodRouter {
	return &MethodRouter{mux.NewRouter()}
}

// MethodRouter beepboop
type MethodRouter struct {
	*mux.Router
}

// HandleFunc registers a route that handles both paths and methods
func (mr *MethodRouter) HandleFunc(path string, f func(http.ResponseWriter, *http.Request), methods ...string) *mux.Route {
	if len(methods) == 0 {
		methods = []string{"GET"}
	}
	return mr.NewRoute().Path(path).Methods(methods...).HandlerFunc(f)
}

// AuthenticateSession verifies the session is valid. We simply store the user object
// in the session. There are 100% better ways to do this but for our case this is
// good enough. If HTTP.Public is enabled, it will not require authentication.
// Please only use this if its on a private IP but regardless, you shouldn't have
// this enabled
func (sg *Sandwich) AuthenticateSession(session *sessions.Session) (auth bool, user *structs.DiscordUser) {
	if sg.Configuration.HTTP.Public {
		auth = true
		return
	}

	userBody, ok := session.Values["user"].([]byte)
	if !ok {
		auth = false
		return
	}

	err := json.Unmarshal(userBody, &user)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to unmarshal user")
		auth = false
		return
	}

	for _, userID := range sg.Configuration.ElevatedUsers {
		if userID == user.ID.Int64() {
			return true, user
		}
	}

	auth = false
	return
}

func createEndpoints(sg *Sandwich) (router *MethodRouter) {
	router = NewMethodRouter()

	router.HandleFunc("/login", LoginHandler(sg), "GET")
	router.HandleFunc("/logout", LogoutHandler(sg), "GET")
	router.HandleFunc("/oauth2/callback", OAuthCallbackHandler(sg), "GET")

	router.HandleFunc("/api/me", APIMeHandler(sg), "GET")

	router.HandleFunc("/api/status", APIStatusHandler(sg), "GET")
	router.HandleFunc("/api/analytics", APIAnalyticsHandler(sg), "GET")

	router.HandleFunc("/api/managers", APIManagersHandler(sg), "GET")
	// GET /api/cluster       ELEVATED
	// GET /api/configuration ELEVATED
	// GET /api/resttunnel    ELEVATED

	// named cluster despite being manager

	// manager:shardgroup:create
	// manager:shardgroup:stop
	// manager:shardgroup:delete

	// manager:create
	// manager:update
	// manager:delete
	// manager:restart
	// manager:refresh_gateway

	// daemon:verify_resttunnel
	// daemon:update

	// POST /api/manager/<manager>/shardgroup/create              - create shardgroup
	// POST /api/manager/<manager>/shardgroup/<shardgroup>/stop   - stops shardgroup
	// POST /api/manager/<manager>/shardgroup/<shardgroup>/delete - deletes shardgroup

	// POST /api/manager/create                    - creates manager
	// POST /api/manager/<manager>/update          - updates manager
	// POST /api/manager/<manager>/delete          - deletes manager
	// POST /api/manager/<manager>/restart         - restarts manager
	// POST /api/manager/<manager>/refresh_gateway - refresh gateway from payload

	// POST /api/daemon/verify_resttunnel - reverify resttunnel
	// POST /api/daemon/update            - update daemon configuration

	return
}
