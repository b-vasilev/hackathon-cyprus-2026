FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /onboarding-api ./cmd/api

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app && \
    mkdir -p /app/data && chown app:app /app/data
WORKDIR /app
COPY --from=build /onboarding-api /usr/local/bin/onboarding-api
USER app
ENV HTTP_ADDR=0.0.0.0:8080 DATABASE_PATH=/app/data/onboarding.db LOG_FORMAT=json
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/onboarding-api"]
