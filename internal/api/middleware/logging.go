package middleware

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxBodyBytes int64 = 1 << 20

type responseRecorder struct {
	w           http.ResponseWriter
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func (r *responseRecorder) Header() http.Header {
	return r.w.Header()
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.w.WriteHeader(status)
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	_, _ = r.body.Write(p)
	return r.w.Write(p)
}

func (r *responseRecorder) Flush() {
	if f, ok := r.w.(http.Flusher); ok {
		f.Flush()
	}
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/swagger") {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()

		reqBody, reqTruncated := readBody(r.Body, maxBodyBytes)
		r.Body = io.NopCloser(bytes.NewReader(reqBody))

		rec := &responseRecorder{w: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		resBody := rec.body.Bytes()
		resTruncated := false
		if int64(len(resBody)) > maxBodyBytes {
			resBody = resBody[:maxBodyBytes]
			resTruncated = true
		}

		log.Printf(
			"api request | ip=%s | method=%s | path=%s | query=%s | status=%d | duration_ms=%d | req_bytes=%d | req_truncated=%t | res_bytes=%d | res_truncated=%t | req_body=%s | res_body=%s",
			clientIP(r),
			r.Method,
			r.URL.Path,
			r.URL.RawQuery,
			rec.status,
			time.Since(start).Milliseconds(),
			len(reqBody),
			reqTruncated,
			len(resBody),
			resTruncated,
			sanitizeLogBody(reqBody),
			sanitizeLogBody(resBody),
		)
	})
}

func readBody(rc io.ReadCloser, max int64) ([]byte, bool) {
	if rc == nil {
		return nil, false
	}
	defer rc.Close()
	var truncated bool
	data, err := io.ReadAll(io.LimitReader(rc, max+1))
	if err != nil {
		return []byte{}, false
	}
	if int64(len(data)) > max {
		data = data[:max]
		truncated = true
	}
	return data, truncated
}

func clientIP(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); v != "" {
		parts := strings.Split(v, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
		return v
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func sanitizeLogBody(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	return s
}
