package gateway

import (
	"github.com/valyala/fasthttp"
)

// RestResponse is the response when returning rest requests
type RestResponse struct {
	Success  bool        `json:"success"`
	Response interface{} `json:"response,omitempty"`
	Error    error       `json:"error,omitempty"`
}

// HandleFastHTTP handles any incomming HTTP requests
func (sg *Sandwich) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/managers":
		res, err := json.Marshal(RestResponse{
			Success:  true,
			Response: sg.Managers,
		})
		if err != nil {
			ctx.SetStatusCode(500)
		} else {
			ctx.Write(res)
		}
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
	if !sg.Configuration.HTTP.Enabled {
		return
	}

	for {
		sg.Logger.Info().Msgf("Running HTTP server at %s", sg.Configuration.HTTP.Host)
		err := fasthttp.ListenAndServe(sg.Configuration.HTTP.Host, sg.HandleFastHTTP)
		sg.Logger.Error().Err(err).Msg("Error occured whilst running fasthttp")
	}
}
