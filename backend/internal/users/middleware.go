package users

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"net/url"
)

type userGetter interface {
	Get(context.Context, string) (User, error)
}

type userContextKey struct{}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey{}).(User)
	return user, ok
}

func Middleware(repository userGetter) func(http.Handler) http.Handler {
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

			user, err := repository.Get(r.Context(), id)
			if errors.Is(err, sql.ErrNoRows) {
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
