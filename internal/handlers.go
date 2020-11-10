package gateway

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/hashicorp/go-uuid"
)

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
			http.Error(rw, "Missing CSRF state", http.StatusInternalServerError)
			return
		}

		if _csrfString != csrfString {
			http.Error(rw, "Mismatched CSRF states", http.StatusUnauthorized)
		}

		// Just to be sure, remove the CSRF after we have compared the CSRF
		delete(session.Values, "oauth_csrf")

		// Create an OAuth exchange with the code we were given.
		code := urlQuery.Get("code")
		token, err := sg.Configuration.OAuth.Exchange(ctx, code)
		if err != nil {
			http.Error(rw, "Failed to exchange code: "+err.Error(), http.StatusInternalServerError)
		}

		// Create a client with our exchanged token and retrieve a user.
		client := sg.Configuration.OAuth.Client(ctx, token)
		resp, err := client.Get(discordUsersMe)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		discordUserResponse := &structs.DiscordUser{}
		err = json.Unmarshal(body, &discordUserResponse)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
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
