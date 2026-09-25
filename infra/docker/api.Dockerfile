# Build context: repository root (needs content/ and assets/licenses for `eduplay seed`).
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/eduplay ./cmd/eduplay

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 eduplay
WORKDIR /app
COPY --from=build /out/eduplay /usr/local/bin/eduplay
COPY content /app/content
COPY assets/licenses /app/assets/licenses
COPY assets/registry.json /app/assets/registry.json
ENV CONTENT_DIR=/app/content ASSETS_DIR=/app/assets STORAGE_DIR=/data/storage HTTP_ADDR=:8080
RUN mkdir -p /data/storage && chown -R eduplay /data
USER eduplay
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["eduplay"]
CMD ["serve"]
