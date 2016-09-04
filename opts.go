package firebase

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/knq/jwt"
	"github.com/knq/oauth2util"
)

const (
	// DefaultTokenExpiration is the default expiration for generated OAuth2 tokens.
	DefaultTokenExpiration = 1 * time.Hour
)

// Option is an option to modify a Firebase ref.
type Option func(r *Ref) error

// URL is an option to set Firebase base URL.
func URL(urlstr string) Option {
	return func(r *Ref) error {
		u, err := url.Parse(urlstr)
		if err != nil {
			return fmt.Errorf("could not parse url: %v", err)
		}

		r.url = u

		return nil
	}
}

// Transport is an option to set the HTTP transport.
func Transport(roundTripper http.RoundTripper) Option {
	return func(r *Ref) error {
		r.transport = roundTripper
		return nil
	}
}

// WatchBufferLen is an option that sets the watch buffer size.
func WatchBufferLen(len int) Option {
	return func(r *Ref) error {
		r.watchBufLen = len
		return nil
	}
}

// GoogleServiceAccountCredentialsJSON loads Firebase credentials from the JSON
// encoded buf.
func GoogleServiceAccountCredentialsJSON(buf []byte) Option {
	return func(r *Ref) error {
		var err error

		var v struct {
			ProjectID   string `json:"project_id"`
			ClientEmail string `json:"client_email"`
			PrivateKey  string `json:"private_key"`
			TokenURI    string `json:"token_uri"`
		}

		// decode settings into v
		err = json.Unmarshal(buf, &v)
		if err != nil {
			return fmt.Errorf("could not unmarshal service account credentials: %v", err)
		}

		// simple check
		if v.ProjectID == "" || v.ClientEmail == "" || v.PrivateKey == "" {
			return errors.New("google service account credentials missing project_id, client_email or private_key")
		}

		// set URL
		err = URL("https://" + v.ProjectID + ".firebaseio.com/")(r)
		if err != nil {
			return err
		}

		// create token signer
		signer, err := jwt.RS256.New(jwt.PEM{[]byte(v.PrivateKey)})
		if err != nil {
			return fmt.Errorf("could not create jwt signer for auth token source: %v", err)
		}

		// create auth token source
		r.auth, err = oauth2util.JWTBearerGrantTokenSource(
			signer, v.TokenURI, context.Background(),
			oauth2util.ExpiresIn(DefaultTokenExpiration),
			oauth2util.IssuedAt(true),
		)
		if err != nil {
			return fmt.Errorf("could not create auth token source: %v", err)
		}

		// add the claims for firebase
		r.auth.AddClaim("iss", v.ClientEmail)
		r.auth.AddClaim("sub", v.ClientEmail)
		r.auth.AddClaim("aud", v.TokenURI)
		r.auth.AddClaim("scope", strings.Join([]string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/firebase.database",
		}, " "))

		// set source
		r.source = oauth2.ReuseTokenSource(nil, r.auth)

		return nil
	}
}

// GoogleServiceAccountCredentialsFile loads Firebase credentials from the
// specified path on disk and configures Firebase accordingly.
//
// Account credentials can be downloaded from the Google Cloud console.
func GoogleServiceAccountCredentialsFile(path string) Option {
	return func(r *Ref) error {
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read google service account credentials file: %v", err)
		}

		return GoogleServiceAccountCredentialsJSON(buf)(r)
	}
}

// DefaultQueryOptions is an option that sets the default query options on the
// ref.
func DefaultQueryOptions(opts ...QueryOption) Option {
	return func(r *Ref) error {
		r.SetQueryOptions(opts...)
		return nil
	}
}

// UserID is an option that sets the auth user id ("uid") via the
// auth_variable_override on the ref.
func UserID(uid string) Option {
	return func(r *Ref) error {
		return DefaultQueryOptions(
			AuthOverride(map[string]interface{}{
				"uid": uid,
			}),
		)(r)
	}
}

// WithClaims is an option that adds additional claims to the auth token source.
func WithClaims(claims map[string]interface{}) Option {
	return func(r *Ref) error {
		return r.AddTokenSourceClaim("claims", claims)
	}
}

// httpLogger handles logging http requests and responses.
type httpLogger struct {
	transport                 http.RoundTripper
	requestLogf, responseLogf Logf
}

// RoundTrip satisifies the http.RoundTripper interface.
func (hl *httpLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	trans := hl.transport
	if trans == nil {
		trans = http.DefaultTransport
	}

	reqBody, _ := httputil.DumpRequestOut(req, true)
	res, err := trans.RoundTrip(req)
	resBody, _ := httputil.DumpResponse(res, true)

	hl.requestLogf("%s", reqBody)
	hl.responseLogf("%s", resBody)

	return res, err
}

// Logf is a logging func.
type Logf func(string, ...interface{})

// Log is an option that writes all HTTP request and response data to the
// respective logger.
func Log(requestLogf, responseLogf Logf) Option {
	return func(r *Ref) error {
		return Transport(&httpLogger{
			transport:    r.transport,
			requestLogf:  requestLogf,
			responseLogf: responseLogf,
		})(r)
	}
}

// QueryOption is an option used to modify the underlying http.Request for
// Firebase.
type QueryOption func(url.Values) error

// Shallow is a query option that toggles Firebase to return a shallow result.
func Shallow(v url.Values) error {
	v.Add("shallow", "true")
	return nil
}

// PrintPretty is a query option that toggles pretty formatting for Firebase
// results.
func PrintPretty(v url.Values) error {
	v.Add("print", "pretty")
	return nil
}

// jsonQuery returns a QueryOption for a field and json encodes the val.
func jsonQuery(field string, val interface{}) QueryOption {
	// json encode
	buf, err := json.Marshal(val)
	if err != nil {
		err = fmt.Errorf("could not marshal query option: %v", err)
	}

	return func(v url.Values) error {
		if err != nil {
			return err
		}

		v.Add(field, string(buf))
		return nil
	}
}

// uintQuery returns a QueryOption for a field that converts n into a string.
func uintQuery(field string, n uint) QueryOption {
	val := strconv.FormatUint(uint64(n), 10)
	return func(v url.Values) error {
		v.Add(field, val)
		return nil
	}
}

// OrderBy is a query option that sets Firebase's returned result order.
func OrderBy(field string) QueryOption {
	return jsonQuery("orderBy", field)
}

// EqualTo is a query option that sets the order by filter to equalTo val.
func EqualTo(val interface{}) QueryOption {
	return jsonQuery("equalTo", val)
}

// StartAt is a query option that sets the order by filter to startAt val.
func StartAt(val interface{}) QueryOption {
	return jsonQuery("startAt", val)
}

// EndAt is a query option that sets the order by filter to endAt val.
func EndAt(val interface{}) QueryOption {
	return jsonQuery("endAt", val)
}

// AuthOverride is a query option that sets the auth_variable_override.
func AuthOverride(val interface{}) QueryOption {
	return jsonQuery("auth_variable_override", val)
}

// LimitToFirst is a query option that limit's Firebase's returned results to
// the first n items.
func LimitToFirst(n uint) QueryOption {
	return uintQuery("limitToFirst", n)
}

// LimitToLast is a query option that limit's Firebase's returned results to
// the last n items.
func LimitToLast(n uint) QueryOption {
	return uintQuery("limitToLast", n)
}
