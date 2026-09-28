# syntax=docker/dockerfile:1

# Sandbox daemon for the assessment supervisor (see supervisor.go): a nested
# dockerd (dind) with the runsc (gVisor) OCI runtime registered, so every
# untrusted submission container the supervisor launches runs under gVisor
# instead of on this host's kernel. compose points the backend at this
# service over DOCKER_HOST.
FROM docker:28-dind

RUN apk add --no-cache curl bzip2

# gVisor's release bucket ships one bundled archive per arch (path matches
# `uname -m`: aarch64 / x86_64) holding runsc, its containerd shim, and a
# gvisor-bin/ directory of sidecar binaries (gvisor_sentry etc.) that runsc
# shells out to at a path relative to itself — it must land at
# /usr/local/bin/gvisor-bin, next to the runsc binary in /usr/local/bin.
RUN set -eux; \
    arch="$(uname -m)"; \
    url="https://storage.googleapis.com/gvisor/releases/release/latest/${arch}/gvisor.tar.bz2"; \
    curl -fsSL -o /tmp/gvisor.tar.bz2 "${url}"; \
    tar -C /usr/local/bin -xjf /tmp/gvisor.tar.bz2 runsc containerd-shim-runsc-v1 gvisor-bin; \
    rm /tmp/gvisor.tar.bz2; \
    chmod -R 0755 /usr/local/bin/runsc /usr/local/bin/containerd-shim-runsc-v1 /usr/local/bin/gvisor-bin

# dockerd reads this automatically on startup, registering runsc alongside
# the default runc so ContainerSpec.Runtime = "runsc" resolves.
COPY dind.daemon.json /etc/docker/daemon.json
