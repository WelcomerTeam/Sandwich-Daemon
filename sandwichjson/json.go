package sandwichjson

import (
	"runtime"

	"github.com/bytedance/sonic"
	jsoniter "github.com/json-iterator/go"
)

var useSonic = runtime.GOARCH == "amd64" && runtime.GOOS == "linux"

func Unmarshal(data []byte, v any) error {
	if useSonic {
		return sonic.UnmarshalString(string(data), v)
	} else {
		return jsoniter.Unmarshal(data, v)
	}
}

func Marshal(v any) ([]byte, error) {
	if useSonic {
		return sonic.Marshal(v)
	} else {
		return jsoniter.Marshal(v)
	}
}
