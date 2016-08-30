// Package tokgen provides a Firebase v3.0.0+ compatible JWT access token
// generator.
package tokgen

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/knq/jwt"
)

const (
	// Audience is the required value for the JWT Audience ("aud") field for
	// Firebase.
	Audience = "https://identitytoolkit.googleapis.com/google.identity.identitytoolkit.v1.IdentityToolkit"

	// DefaultTokenExpirationDuration is the default expiration duration.
	DefaultTokenExpirationDuration time.Duration = 2 * time.Hour
)

// Claims is the JWT payload claims for a Firebase access token.
//
// Mostly copied from knq/jwt.Claims.
type Claims struct {
	// Issuer ("iss") identifies the principal that issued the JWT.
	Issuer string `json:"iss,omitempty"`

	// Subject ("sub") identifies the principal that is the subject of the JWT.
	Subject string `json:"sub,omitempty"`

	// Audience ("aud") identifies the recipients that the JWT is intended for.
	Audience string `json:"aud,omitempty"`

	// IssuedAt ("iat") identifies the time at which the JWT was issued.
	IssuedAt json.Number `json:"iat,omitempty"`

	// NotBefore ("nbf") identifies the time before which the JWT MUST NOT be
	// accepted for processing.
	NotBefore json.Number `json:"nbf,omitempty"`

	// Expiration ("exp") identifies the expiration time on or after which the
	// JWT MUST NOT be accepted for processing.
	Expiration json.Number `json:"exp,omitempty"`

	// UserID identifies the authenticated user to Firebase.
	//
	// This is copied from the parent, but can be overridden with the UserID
	// TokenOption.
	UserID string `json:"uid,omitempty"`

	// Claims is the additional authenticated user data for Firebase.
	//
	// This can be set with the AuthData TokenOption, and is then available in
	// the Firebase `auth` and `request.auth` variables.
	Claims interface{} `json:"claims,omitempty"`
}

// TokenGenerator wraps a jwt.Signer using the appropriate claims for a Firebase token.
type TokenGenerator struct {
	Signer jwt.Signer

	ProjectID           string
	ServiceAccountEmail string

	EnableExpiration bool
	EnableIssuedAt   bool
	EnableNotBefore  bool

	ExpirationDuration time.Duration

	UserID string
}

// New creates a new TokenGenerator using the provided options.
func New(opts ...Option) (*TokenGenerator, error) {
	var err error

	// defaults
	tg := &TokenGenerator{
		EnableExpiration: true,
		EnableIssuedAt:   true,

		ExpirationDuration: DefaultTokenExpirationDuration,
	}

	// apply options
	for _, o := range opts {
		err = o(tg)
		if err != nil {
			return nil, err
		}
	}

	// check that signer is defined
	if tg.Signer == nil {
		return nil, errors.New("no private key was provided")
	}

	// check issuer and subject
	if tg.ServiceAccountEmail == "" {
		return nil, errors.New("no service account email was provided")
	}

	return tg, nil
}

// Token generates a token from the token generator configuration with the
// provided token options.
func (tg *TokenGenerator) Token(opts ...TokenOption) ([]byte, error) {
	tok := &Claims{
		Issuer:   tg.ServiceAccountEmail,
		Subject:  tg.ServiceAccountEmail,
		Audience: Audience,
	}

	now := time.Now()
	n := json.Number(strconv.FormatInt(now.Unix(), 10))

	// set expiration
	if tg.EnableExpiration {
		tok.Expiration = json.Number(strconv.FormatInt(now.Add(tg.ExpirationDuration).Unix(), 10))
	}

	// set issued at
	if tg.EnableIssuedAt {
		tok.IssuedAt = n
	}

	//  set not before
	if tg.EnableNotBefore {
		tok.NotBefore = n
	}

	// set user id
	if tg.UserID != "" {
		tok.UserID = tg.UserID
	}

	// apply options
	for _, o := range opts {
		o(tok)
	}

	// encode
	return tg.Signer.Encode(tok)
}

// TokenString generates a token from the provided token generator configuration as a string the provided token options.
func (tg *TokenGenerator) TokenString(opts ...TokenOption) (string, error) {
	buf, err := tg.Token(opts...)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}
