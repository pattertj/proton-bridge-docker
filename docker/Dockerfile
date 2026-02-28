# Stage 1: Build Proton Bridge from source
FROM golang:1.24-bookworm AS bridge-builder

ARG BRIDGE_VERSION=v3.22.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    git gcc libc6-dev libsecret-1-dev pkg-config libfido2-dev libcbor-dev && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build
RUN git clone --depth 1 --branch ${BRIDGE_VERSION} \
    https://github.com/ProtonMail/proton-bridge.git .

# Build CLI-only (no Qt/GUI dependencies)
RUN make build-nogui

# Stage 2: Build Web UI
FROM golang:1.24-bookworm AS webui-builder

WORKDIR /webui
COPY webui/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /webui-server .

# Stage 3: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    pass \
    gnupg \
    socat \
    ca-certificates \
    procps \
    libfido2-1 \
    libcbor0.8 \
    libsecret-1-0 \
    dbus-x11 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=bridge-builder /build/bridge /usr/bin/bridge
COPY --from=webui-builder /webui-server /usr/bin/webui-server
COPY docker/entrypoint.sh /app/entrypoint.sh
COPY docker/setup.sh /app/setup.sh
COPY docker/GPGparams.txt /app/GPGparams.txt

RUN chmod +x /app/entrypoint.sh /app/setup.sh

EXPOSE 25 143 3000

VOLUME /root

LABEL org.opencontainers.image.title="Proton Mail Bridge" \
      org.opencontainers.image.description="Proton Mail IMAP/SMTP Bridge with Web UI for Unraid" \
      org.opencontainers.image.source="https://github.com/pattertj/proton-bridge-docker" \
      org.opencontainers.image.licenses="GPL-3.0"

ENTRYPOINT ["/app/entrypoint.sh"]
