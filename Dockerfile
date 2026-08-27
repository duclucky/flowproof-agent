FROM node:24-bookworm-slim AS web-build
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/flowproof ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends chromium ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 10001 flowproof \
    && mkdir -p /app/data /app/web/dist \
    && chown -R flowproof:flowproof /app
WORKDIR /app
COPY --from=go-build /out/flowproof /app/flowproof
COPY --from=web-build /src/web/dist /app/web/dist
RUN chown -R flowproof:flowproof /app
USER flowproof
ENV PORT=8080 \
    FLOWPROOF_CHROME_PATH=/usr/bin/chromium \
    FLOWPROOF_WEB_DIR=/app/web/dist \
    FLOWPROOF_STATE_PATH=/app/data/runs.json
EXPOSE 8080
CMD ["/app/flowproof"]
