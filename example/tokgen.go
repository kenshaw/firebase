// example/tokgen.go
package main

import (
	"log"

	"github.com/knq/firebase/tokgen"
)

func main() {
	// create a Firebase auth token generator using a credentials file from
	// disk. note that this can can be obtained from your Firebase console
	tg, err := tokgen.New(
		tokgen.CredentialsFile("./test-1470ffbcc1d8.json"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// generate a token with a specified user id ("uid") and extra auth data
	tok, err := tg.TokenString(
		tokgen.UserID("a really cool user"),
		tokgen.AuthData(map[string]interface{}{
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
