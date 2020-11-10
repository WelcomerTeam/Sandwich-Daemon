package structs

import jsoniter "github.com/json-iterator/go"

// RestResponse is the response when returning rest requests
type RestResponse struct {
	Success  bool        `json:"success"`
	Response interface{} `json:"response,omitempty"`
	Error    error       `json:"error,omitempty"`
}

// RPCRequest is the structure the client sends when an JSON-RPC call is made
type RPCRequest struct {
	Method string              `json:"method"`
	Params jsoniter.RawMessage `json:"params"`
	ID     string              `json:"id"`
}

// RPCResponse is the structure the server sends to respond to a JSON-RPC request
type RPCResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
	ID     string      `json:"id"`
}

// AnalyticResponse is the analytic response when you request the analytics
type AnalyticResponse struct {
	Graph    LineChart            `json:"chart"`
	Guilds   int64                `json:"guilds"`
	Uptime   string               `json:"uptime"`
	Events   int64                `json:"events"`
	Clusters []ClusterInformation `json:"clusters"`
}

// ClusterInformation represents cluster information.
type ClusterInformation struct {
	Name      string                             `json:"name"`
	Guilds    int64                              `json:"guilds"`
	Status    map[int32]structs.ShardGroupStatus `json:"status"`
	AutoStart bool                               `json:"autostart"`
}

// DataStamp stores time and its corresponding value
type DataStamp struct {
	Time  interface{} `json:"x"`
	Value interface{} `json:"y"`
}

// LineChart stores the data structure for a ChartJS LineChart
type LineChart struct {
	Labels   []string  `json:"labels,omitempty"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset is stores the representation of a Dataset in ChartJS
type Dataset struct {
	Label            string        `json:"label"`
	BackgroundColour string        `json:"backgroundColor,omitempty"`
	BorderColour     string        `json:"borderColor,omitempty"`
	Data             []interface{} `json:"data"`
}
