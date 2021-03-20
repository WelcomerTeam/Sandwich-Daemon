export GO111MODULE=on

go get google.golang.org/protobuf/cmd/protoc-gen-go \
         google.golang.org/grpc/cmd/protoc-gen-go-grpc

export PATH="$PATH:$(go env GOPATH)/bin"

protoc --proto_path=protobuf \
    --go_out=protobuf --go_opt=paths=source_relative \
    --go-grpc_out=protobuf --go-grpc_opt=paths=source_relative \
    protobuf/gateway.proto