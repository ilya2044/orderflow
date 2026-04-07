FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY pkg/ ./pkg/
COPY payment-service/ ./payment-service/

WORKDIR /app/payment-service

ENV GOWORK=off
ENV GONOSUMDB=*
RUN go mod download all
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -mod=mod \
    -ldflags="-w -s" \
    -o /bin/payment-service \
    ./cmd/server

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/payment-service /payment-service
COPY --from=builder /app/payment-service/migrations /migrations

EXPOSE 8086

ENTRYPOINT ["/payment-service"]
