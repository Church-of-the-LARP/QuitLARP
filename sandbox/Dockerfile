# Docker-in-Docker daemon that runs every untrusted submission container under
# gVisor (runsc). The backend reaches it over plain TCP on the compose network
# (DOCKER_HOST=tcp://dind:2375); the port is never published to the host.
FROM docker:dind

# Pinned gVisor release. To bump: change the date and recompute both digests with
#   curl -fsSL https://storage.googleapis.com/gvisor/releases/release/<date>/<arch>/gvisor.tar.bz2 | sha512sum
# (gVisor publishes no checksum file for gvisor.tar.bz2).
ARG GVISOR_RELEASE=20261102
ARG GVISOR_SHA512_AMD64=2e6671bb2d033b871c5811ab57e16ac570988b0e0054c525f911cfaf191bab3ca4efb08e0a98daa4b9e7d248f9e5776ef2ef2f8a6b82d829646a42924c98bb05
ARG GVISOR_SHA512_ARM64=574b5775b4417618f63c8506e6bb237e370df85722832a5d0c3831382155e55e59d4cfa993a1e9bbe1b1cf04346ffd48b20aaa879c6ed9a52c680f762fd19980
# Set by buildx to the *target* platform (uname -m would give the build host's).
ARG TARGETARCH

# The release tarball ships runsc, the containerd shim and a gvisor-bin/
# directory of helpers that runsc expects to find next to itself.
RUN set -eu; \
    case "${TARGETARCH:-$(apk --print-arch | sed 's/^x86_64$/amd64/;s/^aarch64$/arm64/')}" in \
      amd64) ARCH=x86_64;  SHA="$GVISOR_SHA512_AMD64" ;; \
      arm64) ARCH=aarch64; SHA="$GVISOR_SHA512_ARM64" ;; \
      *) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    apk add --no-cache curl bzip2; \
    curl -fsSL -o /tmp/gvisor.tar.bz2 \
      "https://storage.googleapis.com/gvisor/releases/release/${GVISOR_RELEASE}/${ARCH}/gvisor.tar.bz2"; \
    echo "${SHA}  /tmp/gvisor.tar.bz2" | sha512sum -c -; \
    tar -xjf /tmp/gvisor.tar.bz2 -C /usr/local/bin; \
    rm /tmp/gvisor.tar.bz2; \
    apk del curl

COPY daemon.json /etc/docker/daemon.json
