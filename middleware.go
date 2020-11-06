package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func WithMethodFilter(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.ToUpper(r.Method) != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = fmt.Fprintln(w, "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func WithInjection(next http.Handler, contextInjection map[string]interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if len(contextInjection) > 0 {
			for k, v := range contextInjection {
				ctx = context.WithValue(ctx, k, v)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}