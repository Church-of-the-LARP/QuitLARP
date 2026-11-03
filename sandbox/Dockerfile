# Docker-in-Docker daemon that runs every untrusted submission container under
# gVisor (runsc). The backend reaches it over plain TCP on the compose network
# (DOCKER_HOST=tcp://dind:2375); the port is never published to the host.
FROM docker:dind

# The release tarball ships runsc, the containerd shim and a gvisor-bin/
# directory of helpers that runsc expects to find next to itself.
RUN apk add --no-cache curl bzip2 \
 && URL="https://storage.googleapis.com/gvisor/releases/release/latest/$(uname -m)/gvisor.tar.bz2" \
 && curl -fsSL "$URL" | tar -xj -C /usr/local/bin \
 && apk del curl

COPY daemon.json /etc/docker/daemon.json
