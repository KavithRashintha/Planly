FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY pkg/ pkg/
COPY services/gateway/ services/gateway/
WORKDIR /app/services/gateway
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/gateway ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/gateway /app/gateway
EXPOSE 8080
USER 10001:10001
ENTRYPOINT ["/app/gateway"]
