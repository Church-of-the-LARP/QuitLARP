package supervisor

import (
	"archive/tar"
	"bytes"
	_ "embed"
	"io"
	"time"
)

//go:embed harness.Dockerfile
var harnessDockerfile []byte

// HarnessDockerfilePath is the name of the Dockerfile inside the build context
// returned by HarnessImageContext. Callers pass it as the dockerfilePath of
// BuildImage.
const HarnessDockerfilePath = "harness.Dockerfile"

// HarnessDockerfile returns a copy of the embedded Dockerfile that builds the
// platform harness image.
func HarnessDockerfile() []byte {
	out := make([]byte, len(harnessDockerfile))
	copy(out, harnessDockerfile)
	return out
}

// HarnessImageContext returns the build context for the harness image: a tar
// holding just the Dockerfile. The UFT libraries a chapter links against are
// private dune libraries that opam cannot install, so they are vendored into
// each assessment's own build context instead of living in this image.
func HarnessImageContext() (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	header := &tar.Header{
		Name:     HarnessDockerfilePath,
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		Size:     int64(len(harnessDockerfile)),
		ModTime:  time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write(harnessDockerfile); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}
