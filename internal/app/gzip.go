package app

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
)

// gzipResponseWriter wraps http.ResponseWriter to optionally gzip-compress
// text-based responses when the client supports it via Accept-Encoding.
type gzipResponseWriter struct {
	http.ResponseWriter
	gz        *gzip.Writer
	compress  bool
	wroteBody bool
}

// maybeGzip creates a gzipResponseWriter. If the client does not advertise gzip
// or the request includes a Range header (binary serve), compression is skipped.
func maybeGzip(w http.ResponseWriter, r *http.Request) *gzipResponseWriter {
	gw := &gzipResponseWriter{ResponseWriter: w}
	if acceptsGzip(r.Header.Get("Accept-Encoding")) && r.Header.Get("Range") == "" && r.Method != http.MethodHead {
		gw.compress = true
		w.Header().Set("Vary", "Accept-Encoding")
	}
	return gw
}

// Close flushes the gzip writer if compression is active.
func (w *gzipResponseWriter) Close() {
	if w.gz != nil {
		w.gz.Close()
	}
}

// WriteHeader inspects Content-Type before sending headers. If the content type
// is set and is not a compressible text type, compression is disabled for this
// response (Content-Encoding header is not set).
func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteBody {
		return
	}
	w.wroteBody = true

	if w.compress {
		ct := w.Header().Get("Content-Type")
		if !statusAllowsBody(code) || (ct != "" && !isCompressibleContentType(ct)) {
			// Non-text or bodyless response: disable compression before any
			// gzip writer is created so the raw response body is never polluted.
			w.compress = false
			w.Header().Del("Content-Encoding")
		}
	}

	if w.compress {
		w.Header().Set("Content-Encoding", "gzip")
		// Compressed size differs from source; remove Content-Length so Go
		// uses chunked transfer-encoding or calculates the compressed length.
		w.Header().Del("Content-Length")
		w.gz = gzip.NewWriter(w.ResponseWriter)
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write writes data through gzip if compression is active.
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteBody {
		w.WriteHeader(http.StatusOK)
	}
	if w.compress {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// isCompressibleContentType returns true for text-based content types that
// benefit from gzip compression.
func isCompressibleContentType(ct string) bool {
	ct = strings.TrimSpace(ct)
	// Strip parameters like charset.
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	switch ct {
	case "text/html", "text/css", "text/plain", "application/javascript", "application/json":
		return true
	}
	return strings.HasPrefix(ct, "text/")
}

func acceptsGzip(acceptEncoding string) bool {
	for _, part := range strings.Split(acceptEncoding, ",") {
		params := strings.Split(strings.ToLower(strings.TrimSpace(part)), ";")
		if strings.TrimSpace(params[0]) != "gzip" {
			continue
		}
		for _, param := range params[1:] {
			param = strings.ReplaceAll(strings.TrimSpace(param), " ", "")
			if strings.HasPrefix(param, "q=") {
				q, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64)
				return err != nil || q > 0
			}
		}
		return true
	}
	return false
}

func statusAllowsBody(code int) bool {
	return code != http.StatusNoContent && code != http.StatusNotModified && code >= 200
}
