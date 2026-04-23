# Build from repository root: docker build -f bot/Dockerfile ./bot
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/bot /usr/local/bin/bot
USER nobody
ENTRYPOINT ["/usr/local/bin/bot"]
