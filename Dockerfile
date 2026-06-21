## Build locally: CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/main.go
## Then: docker compose build

FROM alpine:3.19

RUN adduser -D -g '' appuser
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY bin/server .
COPY assets ./assets

USER appuser
EXPOSE 8080

CMD ["./server"]
