FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY pkg/ pkg/
COPY services/auth/ services/auth/
WORKDIR /app/services/auth
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/auth ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/auth /app/auth
EXPOSE 8081
USER 10001:10001
ENTRYPOINT ["/app/auth"]
