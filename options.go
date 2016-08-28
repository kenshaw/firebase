package fireauth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"github.com/knq/jwt"
)

// Option represents a fireauth option.
type Option func(*fireauth) error

// ProjectID is an Option that sets the project id.
func ProjectID(id string) Option {
	return func(f *fireauth) error {
		f.projectID = id
		return nil
	}
}

// ServiceAccountEmail is an Option that sets the service account email (ie,
// the issuer and subject).
func ServiceAccountEmail(email string) Option {
	return func(f *fireauth) error {
		f.serviceAccountEmail = email
		return nil
	}
}

// PEM is an Option that reads the key from pem.
//
// Note: please see knq/jwt.PEM for information on using this.
func PEM(pem jwt.PEM) Option {
	return func(f *fireauth) error {
		var err error
		f.signer, err = jwt.RS256.New(pem)
		return err
	}
}

// CredentialsJSON is an Option that reads all the relevant settings from the
// supplied JSON-encoded buf (ie, the contents of the JSON file downloaded from
// the Firebase console).
func CredentialsJSON(buf []byte) Option {
	return func(f *fireauth) error {
		var err error

		// the members to extract from the json data.
		vals := struct {
			ProjectID   string `json:"project_id"`
			ClientEmail string `json:"client_email"`
			PrivateKey  string `json:"private_key"`
		}{}

		// read vals from json
		err = json.Unmarshal(buf, &vals)
		if err != nil {
			return err
		}

		if vals.ProjectID == "" || vals.ClientEmail == "" || vals.PrivateKey == "" {
			return errors.New("credentials missing project_id, client_email or private_key")
		}

		// set project id
		err = ProjectID(vals.ProjectID)(f)
		if err != nil {
			return err
		}

		// set service account email
		err = ServiceAccountEmail(vals.ClientEmail)(f)
		if err != nil {
			return err
		}

		// set private key
		return PEM(jwt.PEM{[]byte(vals.PrivateKey)})(f)
	}
}

// CredentialsJSONString is an Option that reads all the relevant settings from
// the supplied JSON-encoded str (ie, the contents of the JSON file downloaded
// from the Firebase console).
func CredentialsJSONString(str string) Option {
	return CredentialsJSON([]byte(str))
}

// CredentialsFile is a fireauth Option that reads all the relevant settings
// from the supplied JSON-encoded file located at path.
func CredentialsFile(path string) Option {
	return func(f *fireauth) error {
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		return CredentialsJSON(buf)(f)
	}
}

// ExpirationDuration is an Option that sets the expiration duration for
// generate tokens.
func ExpirationDuration(d time.Duration) Option {
	return func(f *fireauth) error {
		if d != 0 {
			f.enableExpiration = true
			f.expirationDuration = d
		} else {
			f.enableExpiration = false
			f.expirationDuration = 0
		}

		return nil
	}
}

// IssuedAt is an option that sets whether or not the Issued At ("iat") field
// is set on the token.
func IssuedAt(enable bool) Option {
	return func(f *fireauth) error {
		f.enableIssuedAt = enable
		return nil
	}
}

// NotBefore is an option that sets whether or not the Not Before ("nbf") field
// is set on the token.
func NotBefore(enable bool) Option {
	return func(f *fireauth) error {
		f.enableNotBefore = enable
		return nil
	}
}

// AuthUserID is an option to set the authenticated user id ("uid").
//
// This will be copied on all subsequent calls to Token, and can be overridden
// using the UserID TokenOption.
func AuthUserID(uid string) Option {
	return func(f *fireauth) error {
		return nil
	}
}

// TokenOption represents a fireauth claims option for generated tokens.
type TokenOption func(*Claims)

// UserID is a token option that sets (overriding, if previously set via
// AuthUserID) the authenticated user id ("uid").
//
// See also: AuthUserID
func UserID(uid string) TokenOption {
	return func(c *Claims) {
		c.UserID = uid
	}
}

// AuthData is a token option that sets the extra authenticated claims data to v.
//
// The data will be encoded into the token claims, and can be accessed via the
// Firebase `auth` and `request.auth` security rules variables.
//
// Note: v must be json.Marshal-able.
func AuthData(v interface{}) TokenOption {
	return func(c *Claims) {
		c.Claims = v
	}
}
