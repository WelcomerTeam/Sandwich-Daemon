FROM golang:1.17 AS build_base

WORKDIR /go/brotli-cgo

RUN apt update -y \
    && apt install -y git build-essential cmake zlib1g-dev

RUN cd /usr/local \
    && git clone https://github.com/google/brotli \
    && cd brotli && mkdir out && cd out && ../configure-cmake \
    && make \
    && make install

WORKDIR /tmp/sandwich-daemon

RUN cd /tmp/sandwich-daemon

COPY go.mod .
COPY go.sum .
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 LD_LIBRARY_PATH='/usr/local/lib' \
    go build -a --trimpath -o ./out/sandwich ./cmd/main.go

FROM alpine:3

RUN apk add ca-certificates libc6-compat

COPY --from=build_base /usr/local/lib /usr/local/lib

COPY --from=build_base /tmp/sandwich-daemon/out/sandwich /app/sandwich
COPY --from=build_base /tmp/sandwich-daemon/sandwich/dist /sandwich/dist

CMD ["/app/sandwich"]
