package middleware

import (
    "log"
    "net/http"
    "runtime/debug"
)

func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v\nStack trace:\n%s", err, debug.Stack())
                
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusInternalServerError)
                w.Write([]byte(`{"error": "Internal server error", "code": 500}`))
            }
        }()
        next.ServeHTTP(w, r)
    })
}