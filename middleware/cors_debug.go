package middleware

import (
    "log"
    "net/http"
)

func CORSDebugMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Log request details
        log.Printf("[CORS Debug] Request from Origin: %s", r.Header.Get("Origin"))
        log.Printf("[CORS Debug] Request Method: %s", r.Method)
        log.Printf("[CORS Debug] Request Headers: %v", r.Header)

        // Wrap response writer to capture headers
        wrappedWriter := &responseWriterWrapper{ResponseWriter: w}

        // Call the next handler
        next.ServeHTTP(wrappedWriter, r)

        // Log response details
        log.Printf("[CORS Debug] Response Status: %d", wrappedWriter.status)
        log.Printf("[CORS Debug] Response Headers: %v", wrappedWriter.Header())
    })
}

type responseWriterWrapper struct {
    http.ResponseWriter
    status int
}

func (w *responseWriterWrapper) WriteHeader(status int) {
    w.status = status
    w.ResponseWriter.WriteHeader(status)
} 