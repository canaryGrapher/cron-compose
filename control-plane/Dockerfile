# Multi-stage build for the CronCompose control plane.

FROM golang:1.25-alpine AS build
WORKDIR /src

# Copy the local proto module so the replace directive resolves.
COPY proto /src/proto
COPY control-plane /src/control-plane

WORKDIR /src/control-plane
RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/control-plane ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 ccompose
USER ccompose
WORKDIR /home/ccompose
COPY --from=build /out/control-plane /usr/local/bin/control-plane

# REST + gRPC. TLS material lives under /home/ccompose/tls (TLS_DIR) by default.
EXPOSE 8080 9090
ENTRYPOINT ["/usr/local/bin/control-plane"]
