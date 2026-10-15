FROM golang:1.26.9-bookworm@sha256:d9c68c2c51161e12fd77e4c6320687c9cd86e1af1e3ad6e6cd63ff970641453c AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -trimpath -o /llm-proxy ./cmd/llm-proxy

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
USER 1000:1000
ENV HOME=/data/auth
COPY --from=build /llm-proxy /llm-proxy
COPY THIRD_PARTY_LICENSE_CLIProxyAPI.txt /licenses/CLIProxyAPI.txt
COPY config.example.yaml /etc/llm-proxy/config.yaml
EXPOSE 8317
HEALTHCHECK --interval=30s --timeout=8s --start-period=20s --retries=3 \
  CMD ["/llm-proxy", "-healthcheck"]
ENTRYPOINT ["/llm-proxy"]
CMD ["-config", "/etc/llm-proxy/config.yaml"]
