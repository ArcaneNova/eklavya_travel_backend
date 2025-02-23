package middleware

import (
    "log"
    "net/http"
    "time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Create a custom response writer to capture status code
        wrw := &responseWriter{
            ResponseWriter: w,
            status:        http.StatusOK,
        }

        // Process request
        next.ServeHTTP(wrw, r)

        // Log request details
        duration := time.Since(start)
        log.Printf(
            "%s - [%s] %s %s %d %v",
            r.RemoteAddr,
            time.Now().Format(time.RFC3339),
            r.Method,
            r.URL.Path,
            wrw.status,
            duration,
        )
    })
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}