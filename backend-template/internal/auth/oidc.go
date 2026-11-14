package auth

import (
	"context"
	"crypto/subtle"
	"errors"

	"ai-buddy/backend-template/internal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProvider exchanges authorization codes and verifies the identity claims.
type OIDCProvider struct {
	oauthConfig oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

func NewOIDCProvider(ctx context.Context, c config.Config) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, c.Issuer)
	if err != nil {
		return nil, errors.New("OIDC discovery failed")
	}
	return &OIDCProvider{
		oauthConfig: oauth2.Config{
			ClientID: c.ClientID, ClientSecret: c.ClientSecret,
			RedirectURL: c.RedirectURI, Endpoint: provider.Endpoint(),
			Scopes: []string{oidc.ScopeOpenID, "profile", "email", "ai_buddy"},
		},
		verifier: provider.Verifier(&oidc.Config{
			ClientID: c.ClientID, SupportedSigningAlgs: []string{oidc.RS256},
		}),
	}, nil
}

func (o *OIDCProvider) AuthorizationURL(state, nonce, pkce string) string {
	return o.oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(pkce))
}

// Authenticate returns only the validated user snapshot; OAuth tokens are discarded.
func (o *OIDCProvider) Authenticate(ctx context.Context, code, pkce, nonce string) (User, error) {
	invalid := errors.New("OIDC authentication failed")
	if code == "" || pkce == "" || nonce == "" {
		return User{}, invalid
	}
	token, err := o.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(pkce))
	if err != nil {
		return User{}, invalid
	}
	raw, _ := token.Extra("id_token").(string)
	id, err := o.verifier.Verify(ctx, raw)
	if err != nil || id.Subject == "" || subtle.ConstantTimeCompare([]byte(id.Nonce), []byte(nonce)) != 1 {
		return User{}, invalid
	}
	var u User
	if id.Claims(&u) != nil || !validUser(u) {
		return User{}, invalid
	}
	return u, nil
}
