# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/opslog-server ./cmd/opslog-server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/opslog-server /app/opslog-server
COPY configs/opslog.yml /app/opslog.yml
RUN mkdir -p /app/data
EXPOSE 8600/tcp 8141/tcp 8900/tcp
VOLUME ["/app/data"]
ENTRYPOINT ["/app/opslog-server"]
CMD ["-config", "/app/opslog.yml"]
