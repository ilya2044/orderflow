FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY pkg/ ./pkg/
COPY notification-service/ ./notification-service/

WORKDIR /app/notification-service

ENV GOWORK=off
ENV GONOSUMDB=*
RUN go mod download all
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -mod=mod \
    -ldflags="-w -s" \
    -o /bin/notification-service \
    ./cmd/server

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/notification-service /notification-service

EXPOSE 8087

ENTRYPOINT ["/notification-service"]
