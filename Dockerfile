FROM golang:1.24-bookworm AS build_base

RUN apt update -y \
    && apt install -y git build-essential cmake zlib1g-dev

WORKDIR /go/src/app
COPY . .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 LD_LIBRARY_PATH='/usr/local/lib' \
    go build -a --trimpath -o ./out/sandwich ./main.go

RUN apt install -y npm
RUN cd web && npm i && npm run build

FROM alpine:3
RUN apk add ca-certificates libc6-compat curl
COPY --from=build_base /usr/local/lib /usr/local/lib
COPY --from=build_base ./out/sandwich /app/sandwich
COPY --from=build_base ./web/dist /web/dist
CMD ["/app/sandwich"]

LABEL org.opencontainers.image.source https://github.com/Anti-Raid/Sandwich-Daemon
