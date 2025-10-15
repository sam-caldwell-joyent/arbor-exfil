# syntax=docker/dockerfile:1.6

# -------- Builder Stage --------
FROM ubuntu:22.04 AS builder

ARG DEBIAN_FRONTEND=noninteractive
ARG GO_VERSION=1.22.7
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl git \
    && rm -rf /var/lib/apt/lists/*

# Install Go toolchain
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -o /tmp/go.tgz \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm -f /tmp/go.tgz
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /src

# Pre-cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary for distroless runtime
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o /out/arbor-exfil ./

# Prepare writable data dir placeholder (owned by nonroot in runtime)
RUN mkdir -p /opt/data


# -------- Runtime Stage --------
FROM gcr.io/distroless/static:nonroot AS runtime

WORKDIR /opt/service

# Copy binary and manifest; ensure nonroot ownership
COPY --from=builder --chown=nonroot:nonroot /out/arbor-exfil /opt/service/arbor-exfil
COPY --chown=nonroot:nonroot manifests/inspection_report.yaml /opt/service/inspection_report.yaml

# Create writable data directory
COPY --from=builder --chown=nonroot:nonroot /opt/data /opt/data

# Default config: run from /opt/service, write to /opt/data
ENV ARBOR_EXFIL_MANIFEST=/opt/service/inspection_report.yaml
ENV ARBOR_EXFIL_OUT=/opt/data/output.txt

# Expose a volume for outputs (optional for host bind mounts)
VOLUME ["/opt/data"]

# By default, require target and user via env (ARBOR_EXFIL_TARGET, ARBOR_EXFIL_USER)
# Disable strict host key unless the user mounts known_hosts + sets flag
ENTRYPOINT ["/opt/service/arbor-exfil"]
CMD ["--strict-host-key=false"]

