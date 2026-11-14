package auth

import (
	"database/sql"
	"encoding/gob"
	"net/http"
	"time"

	"ai-buddy/backend-tracker/internal/config"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
)

const (
	SessionKeyOIDCState  = "oidc_state"
	SessionKeyOIDCNonce  = "oidc_nonce"
	SessionKeyOIDCPKCE   = "oidc_pkce"
	SessionKeyOIDCExpiry = "oidc_expiry"
	SessionKeyUser       = "user"
)

// SCS stores values as gob interfaces, including the pending login expiry.
func init() { gob.Register(time.Time{}) }

func NewSessions(db *sql.DB, c config.Config) (*scs.SessionManager, func()) {
	store := postgresstore.New(db)
	s := scs.New()
	s.Store = store
	s.Lifetime = c.SessionLifetime
	s.IdleTimeout = c.SessionIdleTimeout
	s.Cookie.Name = "__Host-session"
	s.Cookie.Secure = true
	s.Cookie.SameSite = http.SameSiteLaxMode
	s.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		jsonError(w, "session unavailable", http.StatusInternalServerError)
	}
	return s, store.StopCleanup
}
