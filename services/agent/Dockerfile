FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY pkg/ pkg/
COPY services/agent/ services/agent/
WORKDIR /app/services/agent
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/agent ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/agent /app/agent
EXPOSE 8084
USER 10001:10001
ENTRYPOINT ["/app/agent"]
