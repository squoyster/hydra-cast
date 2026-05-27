FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/hydracast \
    ./cmd/hydracast

FROM python:3.12-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    ffmpeg \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir yt-dlp

COPY --from=build /out/hydracast /usr/local/bin/hydracast

VOLUME ["/data"]

ENTRYPOINT ["hydracast"]
CMD ["sync", "--config", "/data/config.yaml"]
