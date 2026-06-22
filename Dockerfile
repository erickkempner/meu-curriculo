FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto
RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

FROM alpine:3.19
RUN adduser -D -g '' appuser
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/assets ./assets

USER appuser

CMD ["./server"]
