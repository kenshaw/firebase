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
	"time"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/knq/jwt/gserviceaccount"
)

const (
	// DefaultTokenExpiration is the default expiration for generated OAuth2
	// tokens.
	DefaultTokenExpiration = 1 * time.Hour
)

// requiredScopes are the oauth2 scopes required when using Google service
// accounts with firebase.
var requiredScopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/firebase.database",
	// will this be required in the future?
	//"https://www.googleapis.com/auth/identitytoolkit",
}

// Option is an option to modify a Firebase database ref.
type Option func(r *DatabaseRef) error

// URL is an option to set Firebase database base ref (ie, URL) to urlstr.
func URL(urlstr string) Option {
	return func(r *DatabaseRef) error {
		u, err := url.Parse(urlstr)
		if err != nil {
			return fmt.Errorf("could not parse url: %v", err)
		}

		r.url = u

		return nil
	}
}

// ProjectID is an option that sets the Firebase database base ref (ie, URL) as
// https://<projectID>.firebaseio.com/.
func ProjectID(projectID string) Option {
	return func(r *DatabaseRef) error {
		if projectID == "" {
			return errors.New("project id cannot be empty")
		}

		// set url
		err := URL("https://" + projectID + ".firebaseio.com/")(r)
		if err != nil {
			return errors.New("invalid project id")
		}

		return nil
	}
}

// Transport is an option to set the underlying HTTP transport used when making
// requests against a Firebase database ref.
func Transport(roundTripper http.RoundTripper) Option {
	return func(r *DatabaseRef) error {
		r.transport = roundTripper
		return nil
	}
}

// WatchBufferLen is an option that sets the channel buffer size for the
// returned event channels from Watch and Listen.
func WatchBufferLen(len int) Option {
	return func(r *DatabaseRef) error {
		r.watchBufLen = len
		return nil
	}
}

// GoogleServiceAccountCredentialsJSON is an option that loads Google Service
// Account credentials for use with the Firebase database ref from a JSON
// encoded buf.
//
// Google Service Account credentials can be downloaded from the Google Cloud
// console: https://console.cloud.google.com/iam-admin/serviceaccounts/
func GoogleServiceAccountCredentialsJSON(buf []byte) Option {
	return func(r *DatabaseRef) error {
		var err error

		// load service account credentials
		gsa, err := gserviceaccount.FromJSON(buf)
		if err != nil {
			return err
		}

		// simple check
		if gsa.ProjectID == "" || gsa.ClientEmail == "" || gsa.PrivateKey == "" {
			return errors.New("google service account credentials missing project_id, client_email or private_key")
		}

		// set ref url
		err = ProjectID(gsa.ProjectID)(r)
		if err != nil {
			return err
		}

		// create token source
		ts, err := gsa.TokenSource(nil, requiredScopes...)
		if err != nil {
			return err
		}

		// as of v4 it appears that including the subject with the token is
		// longer necessary, and will cause a 401 unauthorized error with newly
		// created firebase databases.
		//
		// add subject
		/*err = bearer.Claim("sub", gsa.ClientEmail)(ts)
		if err != nil {
			return err
		}*/

		// wrap with a reusable token source
		r.source = oauth2.ReuseTokenSource(nil, ts)

		return nil
	}
}

// GoogleServiceAccountCredentialsFile is an option that loads Google Service
// Account credentials for use with the Firebase database ref from the
// specified file.
//
// Google Service Account credentials can be downloaded from the Google Cloud
// console: https://console.cloud.google.com/iam-admin/serviceaccounts/
func GoogleServiceAccountCredentialsFile(path string) Option {
	return func(r *DatabaseRef) error {
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read google service account credentials file: %v", err)
		}

		return GoogleServiceAccountCredentialsJSON(buf)(r)
	}
}

// GoogleComputeCredentials is an option that loads the Google Service Account
// credentials from the GCE metadata associated with the GCE compute instance.
// If serviceAccount is empty, then the default service account credentials
// associated with the GCE instance will be used.
func GoogleComputeCredentials(serviceAccount string) Option {
	return func(r *DatabaseRef) error {
		var err error

		// get compute metadata scopes associated with the service account
		scopes, err := metadata.Scopes(serviceAccount)
		if err != nil {
			return err
		}

		// check if all the necessary scopes are provided
		for _, s := range requiredScopes {
			if !sliceContains(scopes, s) {
				// NOTE: if you are seeing this error, you probably need to
				// recreate your compute instance with the correct scope
				//
				// as of August 2016, there is not a way to add a scope to an
				// existing compute instance
				return fmt.Errorf("missing required scope %s in compute metadata", s)
			}
		}

		// get compute metadata project id
		projectID, err := metadata.ProjectID()
		if err != nil {
			return err
		}
		if projectID == "" {
			return errors.New("could not retrieve project id from compute metadata service")
		}

		// set ref url
		err = ProjectID(projectID)(r)
		if err != nil {
			return err
		}

		// set transport as the oauth2.Transport
		return Transport(&oauth2.Transport{
			Source: google.ComputeTokenSource(serviceAccount),
			Base:   r.transport,
		})(r)
	}
}

// DefaultQueryOptions is an option that sets the default query options on the
// database ref.
func DefaultQueryOptions(opts ...QueryOption) Option {
	return func(r *DatabaseRef) error {
		r.rw.Lock()
		defer r.rw.Unlock()

		r.queryOpts = opts

		return nil
	}
}

// DefaultAuthOverride is an option that sets the default
// auth_variable_override variable on the database ref.
func DefaultAuthOverride(val interface{}) Option {
	return func(r *DatabaseRef) error {
		return DefaultQueryOptions(AuthOverride(val))(r)
	}
}

// DefaultAuthUID is an option that sets the default auth user id ("uid") via
// the auth_variable_override on the database ref.
func DefaultAuthUID(uid string) Option {
	return func(r *DatabaseRef) error {
		return DefaultQueryOptions(AuthUID(uid))(r)
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
//
// NOTE: this Option will not work with Watch/Listen.
func Log(requestLogf, responseLogf Logf) Option {
	return func(r *DatabaseRef) error {
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

// Shallow is a query option that toggles a query to return shallow result (ie, the keys only).
func Shallow(v url.Values) error {
	v.Add("shallow", "true")
	return nil
}

// PrintPretty is a query option that toggles pretty formatting for query
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

// AuthUID is a query option that sets the auth user id ("uid") via the
// auth_variable_override for a single query.
func AuthUID(uid string) QueryOption {
	return AuthOverride(map[string]interface{}{
		"uid": uid,
	})
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

// sliceContains returns true if haystack contains needle.
func sliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}

	return false
}
