package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"

	// Project packages.
	"backend/internal/users"
	"backend/internal/users/service"
)

type userService interface {
	Get(context.Context, string) (users.User, error)
}

type userContextKey struct{}

func UserFromContext(ctx context.Context) (users.User, bool) {
	user, ok := ctx.Value(userContextKey{}).(users.User)

	return user, ok
}

func Middleware(users userService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("demo_user")
			if err != nil || cookie.Value == "" {
				http.Error(w, "select a user", http.StatusUnauthorized)

				return
			}

			id, err := url.PathUnescape(cookie.Value)
			if err != nil {
				http.Error(w, "invalid demo_user", http.StatusBadRequest)

				return
			}

			user, err := users.Get(r.Context(), id)
			if errors.Is(err, service.ErrNotFound) {
				http.Error(w, "unknown demo_user", http.StatusUnauthorized)

				return
			}

			if err != nil {
				log.Printf("get selected user: %v", err)
				http.Error(w, "failed to get user", http.StatusInternalServerError)

				return
			}

			ctx := context.WithValue(r.Context(), userContextKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
