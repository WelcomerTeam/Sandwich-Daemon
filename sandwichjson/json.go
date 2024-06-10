package sandwichjson

import (
	"io"
	"runtime"

	"github.com/bytedance/sonic"
	jsoniter "github.com/json-iterator/go"
)

const UseSonic = runtime.GOARCH == "amd64" && runtime.GOOS == "linux"

func Unmarshal(data []byte, v any) error {
	if UseSonic {
		return sonic.Unmarshal(data, v)
	} else {
		return jsoniter.Unmarshal(data, v)
	}
}

func UnmarshalReader(reader io.Reader, v any) error {
	if UseSonic {
		return sonic.ConfigDefault.NewDecoder(reader).Decode(v)
	} else {
		return jsoniter.NewDecoder(reader).Decode(v)
	}
}

func Marshal(v any) ([]byte, error) {
	if UseSonic {
		return sonic.Marshal(v)
	} else {
		return jsoniter.Marshal(v)
	}
}

func MarshalToWriter(writer io.Writer, v any) error {
	if UseSonic {
		return sonic.ConfigDefault.NewEncoder(writer).Encode(v)
	} else {
		return jsoniter.NewEncoder(writer).Encode(v)
	}
}
