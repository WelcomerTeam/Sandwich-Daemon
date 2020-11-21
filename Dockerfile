FROM golang:1.15-alpine AS build_base

RUN apk add --no-cache git build-base pkgconfig zlib-dev

WORKDIR /tmp/sandwich-daemon

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o ./out/sandwich ./cmd/main.go

FROM alpine:3.9
RUN apk add ca-certificates

COPY --from=build_base /tmp/sandwich-daemon/out/sandwich /app/sandwich
COPY --from=build_base /tmp/sandwich-daemon/web /web

EXPOSE 5469
CMD ["/app/sandwich"]
