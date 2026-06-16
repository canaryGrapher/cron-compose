# Build context is the repo root (see docker-compose.prod.yml).
FROM golang:1.25 AS build
WORKDIR /src
COPY proxy/ ./proxy/
WORKDIR /src/proxy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/proxy ./cmd/proxy

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/proxy /proxy
# Non-root distroless user.
USER 65532:65532
ENTRYPOINT ["/proxy"]
