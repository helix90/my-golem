FROM golang:1.21-alpine AS builder

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

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /telegram-bot /app/telegram-bot
COPY testdata/ /app/testdata/

ENV AIML_PATH=/app/testdata
ENV VERBOSE=false

ENTRYPOINT ["/app/telegram-bot"]
