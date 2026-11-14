package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

const (
	RoleIntern = "intern"
	RoleMentor = "mentor"
)

type userKey struct{}

func CurrentUser(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}

func validUser(u User) bool {
	return strings.TrimSpace(u.ID) != "" && (u.Role == RoleIntern || u.Role == RoleMentor) && strings.TrimSpace(u.Team) != ""
}

// WithSession wraps a route group once, outside its authentication middleware.
func (a *Auth) WithSession(next http.Handler) http.Handler {
	return http.NewCrossOriginProtection().Handler(a.Sessions.LoadAndSave(next))
}

func (a *Auth) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := a.Sessions.GetString(r.Context(), SessionKeyUser)
		if raw == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var u User
		if json.Unmarshal([]byte(raw), &u) != nil || !validUser(u) {
			http.Error(w, "invalid session", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

// RequireRoles authenticates the user and permits any of the supplied roles.
// An empty role list denies access to every authenticated user.
func (a *Auth) RequireRoles(next http.Handler, roles ...string) http.Handler {
	return a.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := CurrentUser(r.Context())
		if !slices.Contains(roles, user.Role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
