FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY pkg/ pkg/
COPY services/planner/ services/planner/
WORKDIR /app/services/planner
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/planner ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/bin/planner /app/planner
EXPOSE 8082
USER 10001:10001
ENTRYPOINT ["/app/planner"]
