package users

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"

	"ai-buddy/internal/platform/database"
	"ai-buddy/internal/platform/httpx"
)

const (
	RoleIntern = "intern"
	RoleMentor = "mentor"
)

type userKey struct{}

func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}
func CurrentUser(ctx context.Context) User {
	u, _ := ctx.Value(userKey{}).(User)
	return u
}

type Reader interface {
	Get(context.Context, string) (User, error)
	List(context.Context) ([]User, error)
}
type Store struct{ DB *sql.DB }

const columns = `id,name,username,email,role,team,to_char(start_date,'YYYY-MM-DD'),goals,expectations`

func scan(s database.RowScanner) (User, error) {
	var u User
	err := s.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.Role, &u.Team, &u.StartDate, &u.Goals, &u.Expectations)
	return u, err
}
func (p *Store) Get(ctx context.Context, id string) (User, error) {
	return scan(p.DB.QueryRowContext(ctx, `SELECT `+columns+` FROM profiles WHERE id=$1`, id))
}
func (p *Store) List(ctx context.Context) ([]User, error) {
	return database.Query(ctx, p.DB, `SELECT `+columns+` FROM profiles ORDER BY id`, scan)
}

type Selector struct{ Profiles Reader }

func fail(w http.ResponseWriter, status int, message string) {
	httpx.JSON(w, status, map[string]string{"error": message})
}
func (s *Selector) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("demo_user")
		if err != nil || c.Value == "" {
			fail(w, 400, "select a demo_user")
			return
		}
		u, err := s.Profiles.Get(r.Context(), c.Value)
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, 400, "unknown demo_user")
			return
		}
		if err != nil {
			fail(w, 503, "profiles unavailable")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), u)))
	})
}
func (s *Selector) RequireRoles(next http.Handler, roles ...string) http.Handler {
	return s.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := CurrentUser(r.Context())
		if !slices.Contains(roles, u.Role) {
			fail(w, 403, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	}))
}
func (s *Selector) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
		list, err := s.Profiles.List(r.Context())
		if err != nil {
			fail(w, 503, "profiles unavailable")
			return
		}
		httpx.JSON(w, 200, map[string]any{"users": list})
	})
	mux.Handle("GET /api/me", s.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { u := CurrentUser(r.Context()); httpx.JSON(w, 200, u) })))
}
