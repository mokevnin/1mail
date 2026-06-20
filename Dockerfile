# Production image: builds the frontend, then a self-contained Go binary with
# the SPA and tracker embedded, then a minimal runtime. Used for `docker build`
# and docker-compose. Releases (GoReleaser) use Dockerfile.release instead,
# which just wraps the prebuilt binary.

# --- Stage 1: frontend (SPA + tracker bundle) ---
FROM node:25-alpine AS frontend
RUN npm install -g pnpm@11
WORKDIR /src
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm build:tracker && pnpm build

# --- Stage 2: Go binary with embedded assets ---
FROM golang:1.26-alpine AS gobuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed targets: tracker snippet + built SPA from the frontend stage.
COPY --from=frontend /src/packages/analytics/dist/t.js internal/server/assets/t.js
COPY --from=frontend /src/dist/ internal/server/assets/spa/
ARG VERSION=docker
ARG COMMIT=none
RUN go build -tags embed_spa \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /1mail ./cmd/server

# --- Stage 3: runtime ---
FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata
COPY --from=gobuild /1mail /usr/local/bin/1mail
EXPOSE 3000
ENTRYPOINT ["1mail"]
