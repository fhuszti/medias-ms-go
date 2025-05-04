# ┌───────────────────────────────┐
# │ 1) Build stage (static bin)   │
# └───────────────────────────────┘
FROM golang:1.24.2-alpine3.21 AS builder

# install git so 'go mod download' works if you import modules by repo URL
RUN apk add --no-cache git

WORKDIR /src
# cache deps
COPY go.mod go.sum ./
RUN go mod download

# build your service
COPY . .
# force static binary (no cgo), strip debug info
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /medias-ms ./cmd/api

# ┌───────────────────────────────┐
# │ 2) Runtime stage (tiny rpm)   │
# └───────────────────────────────┘
FROM alpine:3.21

# for HTTPS calls (OpenAI API)
RUN apk add --no-cache ca-certificates

WORKDIR /app
# copy the statically‑linked binary
COPY --from=builder /medias-ms .

# drop to non‑root user for safety
USER 65532:65532

EXPOSE 8081  # or whatever port you listen on
ENTRYPOINT ["./medias-ms"]
