# About fireauth

Package fireauth provides a simple Firebase v3.0.0+ JWT access tokens using
credentials downloaded from the Firebase console.

# Usage

Please see [the GoDoc page](http://godoc.org/github.com/knq/fireauth) for a
full API listing.

You can use the package like the following:

```go
// example/example.go
package main

import (
	"log"

	"github.com/knq/fireauth"
)

func main() {
	// create a Firebase auth token generator using a credentials file from
	// disk. note that this can can be obtained from your Firebase console
	auth, err := fireauth.New(
		fireauth.CredentialsFile("./test-1470ffbcc1d8.json"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// generate a token with a specified user id ("uid") and extra auth data
	tok, err := auth.TokenString(
		fireauth.UserID("a really cool user"),
		fireauth.AuthData(map[string]interface{}{
			"premiumAccount": true,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("token: %s", tok)

	// now pass the token to the firebase API ...
	// client, err := firebase.NewClient("https://<project>.firebaseio.com/", token, nil)
}
```
