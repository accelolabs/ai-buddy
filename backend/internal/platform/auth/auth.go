package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/oauth2"
)

type User struct {
	ID       string `json:"authentik_id"`
	Role     string `json:"role"`
	Team     string `json:"team"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"preferred_username"`
}

type Auth struct {
	Sessions     *scs.SessionManager
	oidcProvider *OIDCProvider
}

func New(sessions *scs.SessionManager, provider *OIDCProvider) *Auth {
	return &Auth{Sessions: sessions, oidcProvider: provider}
}

func (a *Auth) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/login", a.login)
	mux.HandleFunc("GET /api/auth/callback", a.callback)
	mux.Handle("GET /api/auth/me", a.RequireUser(http.HandlerFunc(a.me)))
	mux.HandleFunc("POST /api/auth/logout", a.logout)
	return a.WithSession(mux)
}

func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	state := oauth2.GenerateVerifier()
	nonce := oauth2.GenerateVerifier()
	pkce := oauth2.GenerateVerifier()
	ctx := r.Context()
	a.Sessions.Put(ctx, SessionKeyOIDCState, state)
	a.Sessions.Put(ctx, SessionKeyOIDCNonce, nonce)
	a.Sessions.Put(ctx, SessionKeyOIDCPKCE, pkce)
	a.Sessions.Put(ctx, SessionKeyOIDCExpiry, time.Now().Add(5*time.Minute))
	http.Redirect(w, r, a.oidcProvider.AuthorizationURL(state, nonce, pkce), http.StatusFound)
}

func (a *Auth) callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state := a.Sessions.PopString(ctx, SessionKeyOIDCState)
	nonce := a.Sessions.PopString(ctx, SessionKeyOIDCNonce)
	pkce := a.Sessions.PopString(ctx, SessionKeyOIDCPKCE)
	expiry := a.Sessions.PopTime(ctx, SessionKeyOIDCExpiry)
	query := r.URL.Query()
	if state == "" ||
		nonce == "" ||
		pkce == "" ||
		!time.Now().Before(expiry) ||
		subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(state)) != 1 ||
		query.Get("error") != "" ||
		query.Get("code") == "" {
		http.Error(w, "invalid login", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	user, err := a.oidcProvider.Authenticate(ctx, query.Get("code"), pkce, nonce)
	if err != nil {
		http.Error(w, "invalid login", http.StatusBadRequest)
		return
	}

	if a.Sessions.RenewToken(ctx) != nil {
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}

	data, _ := json.Marshal(user)
	a.Sessions.Put(ctx, SessionKeyUser, string(data))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *Auth) me(w http.ResponseWriter, r *http.Request) {
	user, _ := CurrentUser(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	if a.Sessions.Destroy(r.Context()) != nil {
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
