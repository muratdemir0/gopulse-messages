package middleware

import (
	"fmt"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Tracing(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(
			next,
			serviceName,
			otelhttp.WithFilter(func(r *http.Request) bool {
				return r.URL.Path != "/health" && r.URL.Path != "/healthz"
			}),
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
			}),
			otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		)
	}
}

func TracingWithCustomName(operationName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, operationName)
	}
}
