package gateway

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"
)

// RestResponse is the response when returning rest requests
type RestResponse struct {
	Success  bool        `json:"success"`
	Response interface{} `json:"response,omitempty"`
	Error    error       `json:"error,omitempty"`
}

// HandleRequest handles any incomming HTTP requests
func (sg *Sandwich) HandleRequest(ctx *fasthttp.RequestCtx) {
	var res []byte
	var err error

	defer func() {
		sg.Logger.Info().Msgf("%s %s %s %d",
			ctx.RemoteAddr(),
			ctx.Request.Header.Method(),
			ctx.Request.URI().Path(),
			ctx.Response.StatusCode())
	}()

	path := string(ctx.Request.URI().Path())
	if strings.HasPrefix(path, "/static") {
		_, filename := filepath.Split(path)
		root, _ := os.Getwd()
		filepath := filepath.Join(root, "web/static", filename)

		if _, err := os.Stat(filepath); !os.IsNotExist(err) {
			ctx.SendFile(filepath)
		} else {
			ctx.SetStatusCode(404)
		}
	} else {
		switch path {
		case "/":
			b, _ := ioutil.ReadFile("web/spa.html")
			ctx.Response.Header.Set("content-type", "text/html;charset=UTF-8")
			ctx.Write(b)

			// ctx.SendFile("web/spa.html")
			ctx.SetStatusCode(200)

		case "/api/configuration":
			if sg.Configuration.HTTP.Enabled {
				res, err = json.Marshal(RestResponse{true, sg.Managers, nil})
			} else {
				res, err = json.Marshal(RestResponse{false, "HTTP Interface is not enabled", nil})
			}

			if err == nil {
				ctx.Write(res)
				ctx.Response.Header.Set("content-type", "application/javascript;charset=UTF-8")
			}

		default:
			ctx.SetStatusCode(404)
		}
	}

	if err != nil {
		sg.Logger.Warn().Err(err).Msg("Failed to process request")

		if res, err = json.Marshal(RestResponse{false, nil, err}); err == nil {
			ctx.Write(res)
			ctx.Response.Header.Set("content-type", "application/javascript;charset=UTF-8")
		}
		ctx.SetStatusCode(500)
	}

	// GET /managers - lists all managers
	// GET /manager/<> - gets details on manager such as shardgroup, shards and status
	// GET /manager/<>/shards - gets more detailed info on each shard and shardgroups

	// PUT /managers - creates a manager
	// PUT /manager/<>/shardgroup - creates a new shard group

	// POST /manager/<> - update config
	// POST /manager/<>/signal - change status such as turn off and on

}

func (sg *Sandwich) handleRequests() {
	// if !sg.Configuration.HTTP.Enabled {
	// 	return
	// }

	for {
		sg.Logger.Info().Msgf("Running HTTP server at %s", sg.Configuration.HTTP.Host)
		err := fasthttp.ListenAndServe(sg.Configuration.HTTP.Host, sg.HandleRequest)
		sg.Logger.Error().Err(err).Msg("Error occured whilst running fasthttp")
	}
}
