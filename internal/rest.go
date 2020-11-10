package gateway

import (
	"net/http"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/snowflake"
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

// DiscordUser is the structure of a /users/@me request
type DiscordUser struct {
	ID            snowflake.ID `json:"id" msgpack:"id"`
	Username      string       `json:"username" msgpack:"username"`
	Discriminator string       `json:"discriminator" msgpack:"discriminator"`
	Avatar        string       `json:"avatar" msgpack:"avatar"`
	MFAEnabled    bool         `json:"mfa_enabled,omitempty" msgpack:"mfa_enabled,omitempty"`
	Locale        string       `json:"locale,omitempty" msgpack:"locale,omitempty"`
	Verified      bool         `json:"verified,omitempty" msgpack:"verified,omitempty"`
	Email         string       `json:"email,omitempty" msgpack:"email,omitempty"`
	Flags         int          `json:"flags" msgpack:"flags"`
	PremiumType   int          `json:"premium_type" msgpack:"premium_type"`
}

// AuthenticateSession verifies the session is valid. We simply store the user object
// in the session. There are 100% better ways to do this but for our case this is
// good enough.
func (sg *Sandwich) AuthenticateSession(session *sessions.Session) (auth bool) {
	user, ok := session.Values["user"].(string)
	if !ok {
		return false
	}

	_user := &structs.User{}
	err := json.Unmarshal([]byte(user), &_user)
	if err != nil {
		sg.Logger.Error().Err(err).Msg("Failed to unmarshal user")
		return false
	}

	return true
}

func createEndpoints(sg *Sandwich) (router *MethodRouter) {
	router = NewMethodRouter()

	// login
	// logout
	// oauth/callback

	// api/me

	// GET /api/analytics
	// GET /api/configuration
	// GET /api/cluster
	// GET /api/resttunnel

	// PUT /api/manager/<manager>/cluster/<cluster>/shardgroup/create
	// POST /api/manager/<manager>/cluster/cluster/shardgroup/1/stop

	return
}
