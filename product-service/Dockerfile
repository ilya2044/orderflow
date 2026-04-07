FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY pkg/ ./pkg/
COPY product-service/ ./product-service/

WORKDIR /app/product-service

ENV GOWORK=off
ENV GONOSUMDB=*
RUN go mod download all
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -mod=mod \
    -ldflags="-w -s" \
    -o /bin/product-service \
    ./cmd/server

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /bin/product-service /product-service

EXPOSE 8083

ENTRYPOINT ["/product-service"]
