// Package fireauth provides a simple access token generator for the Firebase v3.0.0+ API.
package fireauth

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/knq/jwt"
)

const (
	// Audience is the required value for the audience ("aud") field for
	// Firebase.
	Audience = "https://identitytoolkit.googleapis.com/google.identity.identitytoolkit.v1.IdentityToolkit"

	// DefaultExpirationDuration is the default expiration duration.
	DefaultExpirationDuration time.Duration = 2 * time.Hour
)

// Claims is the JWT payload claims for a Firebase access token.
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

// fireauth wraps a token signer with the required data.
type fireauth struct {
	signer jwt.Signer

	projectID           string
	serviceAccountEmail string

	enableExpiration bool
	enableIssuedAt   bool
	enableNotBefore  bool

	expirationDuration time.Duration
}

// New creates a new fireauth using the provided options.
func New(opts ...Option) (*fireauth, error) {
	var err error

	// default options
	f := &fireauth{
		enableExpiration: true,
		enableIssuedAt:   true,
		enableNotBefore:  true,

		expirationDuration: DefaultExpirationDuration,
	}

	// apply options
	for _, o := range opts {
		err = o(f)
		if err != nil {
			return nil, err
		}
	}

	// check that signer is defined
	if f.signer == nil {
		return nil, errors.New("no private key was provided")
	}

	// check issuer and subject
	if f.serviceAccountEmail == "" {
		return nil, errors.New("no service account email was provided")
	}

	return f, nil
}

// Token generates a token with the provided token options.
func (f *fireauth) Token(opts ...TokenOption) ([]byte, error) {
	tok := &Claims{
		Issuer:   f.serviceAccountEmail,
		Subject:  f.serviceAccountEmail,
		Audience: Audience,
	}

	now := time.Now()
	n := json.Number(strconv.FormatInt(now.Unix(), 10))

	// set expiration
	if f.enableExpiration {
		tok.Expiration = json.Number(strconv.FormatInt(now.Add(f.expirationDuration).Unix(), 10))
	}

	// set issued at
	if f.enableIssuedAt {
		tok.IssuedAt = n
	}

	//  set not before
	if f.enableNotBefore {
		tok.NotBefore = n
	}

	// apply options
	for _, o := range opts {
		o(tok)
	}

	// encode
	return f.signer.Encode(tok)
}

// TokenString generates a token with the provided token options.
func (f *fireauth) TokenString(opts ...TokenOption) (string, error) {
	buf, err := f.Token(opts...)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}
