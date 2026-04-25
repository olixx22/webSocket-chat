FROM golang:1.25.9-alpine AS builder

WORKDIR /app

RUN apk --no-cache add bash

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/output/messenger-server ./messenger-server/cmd/wsserver/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/output/migrator ./messenger-server/cmd/migrator/main.go


FROM alpine:3.22.3

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/output/messenger-server ./messenger-server
COPY --from=builder /app/output/migrator ./migrator
COPY messenger-server/config ./config
COPY messenger-server/migrations ./migrations
COPY messenger-server/scripts/entrypoint.sh ./entrypoint.sh

RUN chmod +x ./entrypoint.sh && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
