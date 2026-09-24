# syntax=docker/dockerfile:1

# The push validation gate regenerates a chapter's python bindings with the utf
# CLI from the universal testing framework. utf is an OCaml program, so it is
# built here from the framework sources and only the binary is carried over;
# compose passes the checkout as the "utf-src" build context (see compose.yaml,
# UTF_SRC overrides the path).
FROM ocaml/opam:alpine-3.23-ocaml-5.6 AS utf
RUN opam update
RUN opam install -y dune cmdliner ppxlib
# The checkout arrives root-owned, and its _build holds artifacts from whatever
# machine it was last built on; drop them and hand the tree to opam to build.
USER root
COPY --from=utf-src / /utf
RUN rm -rf /utf/_build && chown -R opam:opam /utf
USER opam
RUN cd /utf && opam exec -- dune build bin/main.exe

# The assessment supervisor ships these UFT sources to the dind daemon to build
# the harness image (see backend/supervisor/harness.Dockerfile), so the runtime
# image carries them at /utf-src. _build and .git are stripped in their own
# stage so they never land in a final image layer.
FROM alpine:3.23 AS utf-src-clean
COPY --from=utf-src / /utf-src
RUN rm -rf /utf-src/_build /utf-src/.git \
    && find /utf-src -name '._*' -delete \
    && find /utf-src -name '.DS_Store' -delete

FROM golang:1.26-alpine
WORKDIR /app
# The local git server (handlers/git.go) shells out to git for upload-pack,
# receive-pack and repo creation; the git update hook posts its push
# validation request with curl.
RUN apk add --no-cache git curl
# The gate runs utf directly (VALIDATION_GENERATE_COMMAND in compose.yaml).
# Going through `dune exec` instead would also need an OCaml compiler in this
# image, and dune would run an executable of that name shipped by the pushed
# repository.
COPY --from=utf /utf/_build/default/bin/main.exe /usr/local/bin/utf
# UFT sources for the harness builder image (ASSESSMENT_UTF_SRC_DIR).
COPY --from=utf-src-clean /utf-src /utf-src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8888
CMD ["go", "run", "."]
