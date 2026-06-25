# syntax=docker/dockerfile:1

# 1) Build the frontend
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2) Build the backend with the frontend embedded
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Bring in the freshly built frontend so go:embed picks it up.
COPY --from=web /app/web/dist ./web/dist
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /homescape ./cmd/homescape
# Pre-create the data dir so a fresh named volume inherits nonroot ownership.
RUN mkdir -p /data-empty

# 3) Minimal runtime image
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /homescape /homescape
COPY --from=build --chown=65532:65532 /data-empty /data
EXPOSE 8080
VOLUME /data
ENTRYPOINT ["/homescape"]
