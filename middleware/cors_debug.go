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

        // For preflight requests
        if r.Method == "OPTIONS" {
            log.Printf("[CORS Debug] Handling preflight request")
            w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Origin")
            w.WriteHeader(http.StatusOK)
            return
        }

        // Call the next handler
        next.ServeHTTP(w, r)

        // Log response headers
        log.Printf("[CORS Debug] Response Headers: %v", w.Header())
    })
} 