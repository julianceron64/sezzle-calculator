# syntax=docker/dockerfile:1

# 1. Build the SPA.
FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 2. Build the API. The backend is stdlib only, so there is nothing to download.
FROM golang:1.24-alpine AS api
WORKDIR /src/backend
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# 3. One small non-root image: the Go binary serves the API and the built SPA (ADR-011).
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api /out/server /app/server
COPY --from=web /src/frontend/dist /app/web
ENV STATIC_DIR=/app/web PORT=8080
EXPOSE 8080
USER nonroot:nonroot
# distroless has no curl, so the binary checks itself.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 CMD ["/app/server", "-healthcheck"]
ENTRYPOINT ["/app/server"]
