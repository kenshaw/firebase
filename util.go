package firebase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// checkServerError looks at a http.Response and determines if it encountered
// an error, and marshals the error into a Error if it did.
func checkServerError(res *http.Response) error {
	// some kind of server error
	if res.StatusCode < 200 || res.StatusCode > 299 {
		buf, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return &Error{
				Err: fmt.Sprintf("unable to read server error: %v", err),
			}
		}
		if len(buf) < 1 {
			return &Error{
				Err: fmt.Sprintf("empty server error: %s (%d)", res.Status, res.StatusCode),
			}
		}

		var e Error
		err = json.Unmarshal(buf, &e)
		if err != nil {
			return &Error{
				Err: fmt.Sprintf("unknown server error: %s (%d)", string(buf), res.StatusCode),
			}
		}

		return &e
	}

	return nil
}
