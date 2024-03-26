FROM golang:1.22 AS build_base

RUN apt update -y \
    && apt install -y git build-essential cmake zlib1g-dev

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
COPY --from=build_base /tmp/sandwich-daemon/web/dist /web/dist

CMD ["/app/sandwich"]

LABEL org.opencontainers.image.source https://github.com/WelcomerTeam/Sandwich-Daemon