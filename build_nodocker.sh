echo "Build Web Distributable"
cd sandwich
yarn build
cd ..

echo "Simplify"
gofmt -l -s -w . && gofumpt -l -w . && gci --write . && goimports -local -w .

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 LD_LIBRARY_PATH='/usr/local/lib' \
    go build -a --trimpath -o ./out/sandwich ./cmd/main.go
