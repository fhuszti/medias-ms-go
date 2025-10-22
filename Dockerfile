# syntax=docker/dockerfile:1.6

ARG GO_VERSION=1.25.3

FROM golang:${GO_VERSION}-alpine AS base

WORKDIR /app

ENV PATH="/go/bin:${PATH}" \
    CGO_ENABLED=0

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN GOWORK=off go mod download

FROM base AS dev

RUN go install github.com/cosmtrek/air@v1.52.0

COPY . .

CMD ["air", "-c", ".air.toml"]

FROM base AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o /app/bin/medias-ms ./cmd/api

FROM scratch AS prod

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /app/bin/medias-ms /app/medias-ms

EXPOSE 8081

ENTRYPOINT ["/app/medias-ms"]
