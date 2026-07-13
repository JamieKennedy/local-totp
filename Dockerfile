# syntax=docker/dockerfile:1.7

FROM node:26.5.0-bookworm-slim AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
COPY api/openapi.json ../api/openapi.json
RUN npm run generate:api && npm run build

FROM golang:1.26.5-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /src/web/dist ./internal/webui/dist
ARG VERSION=dev
ARG COMMIT=unknown
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -tags webui_dist -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/local-totp ./cmd/local-totp && \
    mkdir -p /out/data && touch /out/data/.keep

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/JamieKennedy/local-totp"
ENV LOCAL_TOTP_LISTEN_ADDR=:8080 \
    LOCAL_TOTP_DATA_DIR=/data
COPY --from=build --chown=65532:65532 /out/local-totp /local-totp
COPY --from=build --chown=65532:65532 /out/data /data
WORKDIR /data
USER 65532:65532
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/local-totp"]
CMD ["serve"]
