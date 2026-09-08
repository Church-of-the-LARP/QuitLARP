package middleware

import (
	"bytes"
	"context"
	"log"
	"net/http"
)

// huma operations have no access to the raw http.ResponseWriter, and headers
// appended from huma middleware run too late (the response is already
// written). So cookies are buffered in the request context and flushed by
// this plain-HTTP middleware, which wraps the huma mux: the (small JSON)
// response body is captured, the Set-Cookie headers are attached, and only
// then is the response sent.
type cookieJarKey struct{}

type cookieJar struct {
	cookies []*http.Cookie
}

// AddCookie queues a cookie to be attached to the current response. Call it
// from any handler running under the CookieJar middleware.
func AddCookie(ctx context.Context, cookie *http.Cookie) {
	jar, ok := ctx.Value(cookieJarKey{}).(*cookieJar)
	if !ok {
		log.Printf("AddCookie called outside the CookieJar middleware")
		return
	}
	jar.cookies = append(jar.cookies, cookie)
}

// CookieJar wraps next, buffering responses so cookies added through
// AddCookie land on the real response.
func CookieJar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jar := &cookieJar{}
		r = r.WithContext(context.WithValue(r.Context(), cookieJarKey{}, jar))

		captured := &capturedResponse{header: http.Header{}}
		next.ServeHTTP(captured, r)

		// Copy inner headers, then append the cookies.
		for k, values := range captured.header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		for _, c := range jar.cookies {
			w.Header().Add("Set-Cookie", c.String())
		}
		status := captured.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if captured.body.Len() > 0 {
			_, _ = w.Write(captured.body.Bytes())
		}
	})
}

// capturedResponse records headers/status/body instead of writing them.
type capturedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (c *capturedResponse) Header() http.Header { return c.header }
func (c *capturedResponse) Write(p []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	return c.body.Write(p)
}
func (c *capturedResponse) WriteHeader(status int) { c.status = status }
