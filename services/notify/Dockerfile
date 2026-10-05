FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY pkg/ pkg/
COPY services/notify/ services/notify/
WORKDIR /app/services/notify
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/notify ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/notify /app/notify
EXPOSE 8083
USER 10001:10001
ENTRYPOINT ["/app/notify"]
