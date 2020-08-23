package gateway

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/structs"
	"github.com/valyala/fasthttp"
)

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
		case "/api/analytics":
			if sg.Configuration.HTTP.Enabled {

				clusters := make([]ClusterInformation, 0, len(sg.Managers))
				for _, mg := range sg.Managers {
					statuses := make(map[int32]structs.ShardGroupStatus)
					for i, sg := range mg.ShardGroups {
						statuses[i] = sg.Status
					}

					guildCount, err := mg.RedisClient.HLen(mg.ctx, mg.CreateKey("guilds")).Result()
					if err != nil {
						mg.Logger.Error().Err(err).Msg("Failed to retrieve Hashset Length")
					}

					clusters = append(clusters, ClusterInformation{
						Name:      mg.Configuration.DisplayName,
						Guilds:    guildCount,
						Status:    statuses,
						AutoStart: mg.Configuration.AutoStart,
					})
				}

				now := time.Now()
				response := AnalyticResponse{
					Graph:    sg.ConstructAnalytics(),
					Uptime:   now.Sub(sg.Start).Round(time.Second).String(),
					Events:   atomic.LoadInt64(sg.TotalEvents),
					Clusters: clusters,
				}

				res, err = json.Marshal(RestResponse{true, response, nil})
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

// // ConstructAnalytics returns a LineChart struct based off of manager analytics
// func (sg *Sandwich) ConstructAnalytics() LineChart {
// 	labels := make(map[time.Time]map[string]int64)
// 	for _, mg := range sg.Managers {
// 		if mg.Analytics != nil {
// 			for _, sample := range mg.Analytics.Samples {
// 				if _, ok := labels[sample.StoredAt]; !ok {
// 					labels[sample.StoredAt] = make(map[string]int64)
// 				}

// 				labels[sample.StoredAt][mg.Configuration.Identifier] = sample.Value
// 			}
// 		}
// 	}

// 	keys := make([]time.Time, 0, len(labels))
// 	for key := range labels {
// 		keys = append(keys, key)
// 	}

// 	sort.Slice(keys, func(i, j int) bool {
// 		return keys[i].Before(keys[j])
// 	})

// 	// Get last 15 minutes only
// 	index := len(keys) - 60
// 	if index < 0 {
// 		index = 0
// 	}

// 	keys = keys[index:]

// 	_keys := make([]string, 0, len(keys))
// 	for _, key := range keys {
// 		_keys = append(_keys, key.Round(time.Second*5).Format(time.Stamp))
// 	}

// 	datasets := make([]Dataset, 0, len(sg.Managers))

// 	mankeys := make([]string, 0, len(sg.Managers))
// 	for key := range sg.Managers {
// 		mankeys = append(mankeys, key)
// 	}
// 	sort.Strings(mankeys)

// 	for i, ident := range mankeys {
// 		mg := sg.Managers[ident]
// 		data := make([]interface{}, 0, len(_keys))
// 		for _, time := range keys {
// 			if val, ok := labels[time][mg.Configuration.Identifier]; ok {
// 				data = append(data, val)
// 			} else {
// 				data = append(data, nil)
// 			}
// 		}

// 		colour := colours[i%len(colours)]
// 		datasets = append(datasets, Dataset{
// 			Label:            mg.Configuration.DisplayName,
// 			BackgroundColour: colour[0],
// 			BorderColour:     colour[1],
// 			Data:             data,
// 		})
// 	}

// 	return LineChart{_keys, datasets}
// }

// ConstructAnalytics returns a LineChart struct based off of manager analytics
func (sg *Sandwich) ConstructAnalytics() LineChart {
	datasets := make([]Dataset, 0, len(sg.Managers))

	mankeys := make([]string, 0, len(sg.Managers))
	for key := range sg.Managers {
		mankeys = append(mankeys, key)
	}
	sort.Strings(mankeys)

	// labels := make(map[time.Time]map[string]int64)
	// for _, mg := range sg.Managers {
	// 	if mg.Analytics != nil {
	// 		for _, sample := range mg.Analytics.Samples {
	// 			if _, ok := labels[sample.StoredAt]; !ok {
	// 				labels[sample.StoredAt] = make(map[string]int64)
	// 			}

	// 			labels[sample.StoredAt][mg.Configuration.Identifier] = sample.Value
	// 		}
	// 	}
	// }

	for i, ident := range mankeys {
		mg := sg.Managers[ident]
		data := make([]interface{}, 0, len(mg.Analytics.Samples))

		for _, sample := range mg.Analytics.Samples {
			data = append(data, DataStamp{sample.StoredAt, sample.Value})
		}

		colour := colours[i%len(colours)]
		datasets = append(datasets, Dataset{
			Label:            mg.Configuration.DisplayName,
			BackgroundColour: colour[0],
			BorderColour:     colour[1],
			Data:             data,
		})
	}

	return LineChart{
		Datasets: datasets,
	}
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

// AnalyticResponse is the analytic response when you request the analytics
type AnalyticResponse struct {
	Graph    LineChart            `json:"chart"`
	Uptime   string               `json:"uptime"`
	Events   int64                `json:"events"`
	Clusters []ClusterInformation `json:"clusters"`
}

// ClusterInformation is cluster information...
type ClusterInformation struct {
	Name      string                             `json:"name"`
	Guilds    int64                              `json:"guilds"`
	Status    map[int32]structs.ShardGroupStatus `json:"status"`
	AutoStart bool                               `json:"autostart"`
}

// DataStamp is a struct to store a time and a corresponding value
type DataStamp struct {
	Time  interface{} `json:"x"`
	Value interface{} `json:"y"`
}

// LineChart is a struct to store LineChart data easier
type LineChart struct {
	Labels   []string  `json:"labels,omitempty"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset is a struct to store data for a Chart
type Dataset struct {
	Label            string        `json:"label"`
	BackgroundColour string        `json:"backgroundColor,omitempty"`
	BorderColour     string        `json:"borderColor,omitempty"`
	Data             []interface{} `json:"data"`
}
