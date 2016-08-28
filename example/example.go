// example/example.go
package main

import (
	"log"

	"github.com/knq/fireauth"
)

func main() {
	// read credentials from json
	auth, err := fireauth.New(
		fireauth.CredentialsFile("./test-1470ffbcc1d8.json"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// create a token for with a user id ("uid") and specific auth data
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
