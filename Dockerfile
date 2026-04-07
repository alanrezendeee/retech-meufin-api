# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H app
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY migrations /app/migrations
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
