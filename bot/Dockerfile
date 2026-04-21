# Context must be the bot module directory (where go.mod lives), e.g. from repo root:
#   docker build -f bot/Dockerfile -t lifesoundtrack-bot bot
# Or from bot/: docker build -t lifesoundtrack-bot .

FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
# Must exist under bot/ when you build (e.g. copy from .env.example). Values are baked into the image;
# override at runtime with docker run -e … (existing env wins over the file).
COPY .env.production .env
COPY --from=build /out/bot /usr/local/bin/bot
ENTRYPOINT ["/usr/local/bin/bot"]
