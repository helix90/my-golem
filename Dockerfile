FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy root module (the library)
COPY go.mod go.sum ./
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/
COPY *.go ./

# Copy examples-module (contains telegram_bot.go)
COPY examples-module/ ./examples-module/

# Build the telegram bot
WORKDIR /build/examples-module
RUN go mod download && \
    go build -o /telegram-bot telegram_bot.go

# ---

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /telegram-bot /app/telegram-bot
COPY testdata/ /app/testdata/

# What the bot remembers about each user, and their reminders
RUN mkdir -p /app/data && chmod 700 /app/data
VOLUME /app/data

ENV AIML_PATH=/app/testdata
ENV VERBOSE=false
ENV USER_STORE_PATH=/app/data/users.json
ENV REMINDERS_PATH=/app/data/reminders.json

ENTRYPOINT ["/app/telegram-bot"]
