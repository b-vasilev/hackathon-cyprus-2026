FROM node:24-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build && npm prune --omit=dev

FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /onboarding-api ./cmd/api

FROM node:24-alpine
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app && \
    mkdir -p /app/data && chown app:app /app/data
WORKDIR /app
COPY --from=build /onboarding-api /usr/local/bin/onboarding-api
COPY --from=web /web/dist /app/web
COPY --from=web /web/node_modules /opt/packager/node_modules
USER app
ENV HTTP_ADDR=0.0.0.0:8080 DATABASE_PATH=/app/data/onboarding.db LOG_FORMAT=json
ENV FRONTEND_DIR=/app/web REPOMIX_BIN=/opt/packager/node_modules/.bin/repomix
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/onboarding-api"]
