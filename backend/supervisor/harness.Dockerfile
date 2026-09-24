# Platform builder for the OCaml UFT harness, built once on the sandbox
# daemon. The chapter specs link against the UFT libraries (utf_core,
# utf_bridge, utf_ppx), which are private dune libraries and therefore vendored
# into every assessment build context by the backend; this image only provides
# the compiler, dune and ppxlib.
# The base is Debian 13 to match the Debian-based assessment images the built
# harness runs in. OCaml 5.4 is a released compiler; the 5.6 image tracks a
# trunk snapshot whose ppxlib preview fails to build.
FROM ocaml/opam:debian-13-ocaml-5.4
USER opam
RUN opam install -y dune ppxlib
