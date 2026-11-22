# Build from backend/ (includes shared pkg at pkg/):
#   docker build -f md-backend-monolith/Dockerfile -t md-backend-monolith .

FROM golang:1.25-alpine AS build

RUN apk add --no-cache ca-certificates git

COPY pkg /pkg
WORKDIR /src
COPY md-backend-monolith/go.mod md-backend-monolith/go.sum ./
COPY md-backend-monolith/ .

ENV GOWORK=off
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/md-backend-monolith ./cmd/server

FROM arigaio/atlas:latest-alpine

COPY --from=build /out/md-backend-monolith /md-backend-monolith
COPY --from=build /src/migrations /migrations

USER nobody

EXPOSE 8080

ENTRYPOINT ["/md-backend-monolith"]