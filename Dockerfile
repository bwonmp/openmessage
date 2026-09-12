# OpenMessage — headless MCP server
#
# Run the OpenMessage backend as a server, exposing the local message store
# and MCP endpoint to assistants. Useful when you want to keep the inbox
# running on a home server / NAS and connect from desktop clients.
#
# Build:   docker build -t openmessage .
# Run:     docker run -p 7007:7007 -v openmessage-data:/data openmessage
# Pair:    docker exec -it <container> openmessage pair
# Connect: claude mcp add -s user --transport sse openmessage http://<host>:7007/mcp/sse

# 1.26, not 1.25: the whatsmeow bump raised go.mod's directive to `go 1.26.0`
# (a transitive requirement), and a 1.25 toolchain refuses to build a module
# that asks for more than it provides.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/openmessage \
      .

# ── signal-cli ───────────────────────────────────────────────
# Fetched in its own stage so neither the 113MB tarball nor curl ends up in
# the runtime layer. The "Linux-native" asset is a GraalVM native-image build,
# so it needs NO JVM — that is why this adds ~250MB rather than a JRE plus the
# JVM distribution on top.
#
# Verified by checksum because this is a plain GitHub release download over
# the network at build time, and an unverified binary that talks to Signal on
# your behalf is not something to take on trust.
FROM debian:stable-slim AS signalcli
ARG SIGNAL_CLI_VERSION=0.14.8
ARG SIGNAL_CLI_SHA256=36569af20c709e0c5e6e677b74f50f147b21f3740620b8a7affde70f6027f82a
RUN set -eux; \
    apt-get update -qq; \
    apt-get install -y -qq --no-install-recommends curl ca-certificates; \
    curl -fsSL -o /tmp/signal-cli.tar.gz \
      "https://github.com/AsamK/signal-cli/releases/download/v${SIGNAL_CLI_VERSION}/signal-cli-${SIGNAL_CLI_VERSION}-Linux-native.tar.gz"; \
    echo "${SIGNAL_CLI_SHA256}  /tmp/signal-cli.tar.gz" | sha256sum -c -; \
    tar xzf /tmp/signal-cli.tar.gz -C /usr/local/bin signal-cli; \
    chmod 0755 /usr/local/bin/signal-cli; \
    rm -f /tmp/signal-cli.tar.gz

# ── runtime ──────────────────────────────────────────────────
# Debian, NOT alpine, and the reason is signal-cli.
#
# signal-cli's native build is a GraalVM image linked against glibc. On
# alpine (musl) it does not run at all, and the failure is maximally
# confusing: the kernel reports the *interpreter* as missing, so you get
#
#   sh: /usr/local/bin/signal-cli: not found
#
# on a file that plainly exists and is executable. Verified directly:
# alpine:3.20 gives exactly that, debian:stable-slim prints "signal-cli
# 0.14.8". gcompat is not a reliable fix for a native-image binary, so the
# base moves instead. openmessage itself is a static CGO_ENABLED=0 binary and
# does not care either way.
#
# wget is installed deliberately: it is not in debian-slim, and the compose
# healthcheck (`wget -qO- http://127.0.0.1:7007/api/status`) would otherwise
# fail on every probe and park the container as unhealthy forever.
FROM debian:stable-slim AS runtime
RUN set -eux; \
    apt-get update -qq; \
    apt-get install -y -qq --no-install-recommends ca-certificates tzdata wget; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd -r openmessage; \
    useradd -r -g openmessage -m -d /home/openmessage openmessage; \
    mkdir -p /data && chown openmessage:openmessage /data
USER openmessage
WORKDIR /home/openmessage
COPY --from=build /out/openmessage /usr/local/bin/openmessage
COPY --from=signalcli /usr/local/bin/signal-cli /usr/local/bin/signal-cli

ENV OPENMESSAGES_DATA_DIR=/data \
    OPENMESSAGES_HOST=0.0.0.0 \
    OPENMESSAGES_PORT=7007

VOLUME ["/data"]
EXPOSE 7007

# Default to running the server. Override with `pair`, `import`, etc.
ENTRYPOINT ["openmessage"]
CMD ["serve"]
