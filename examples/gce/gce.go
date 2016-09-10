// example/gce/gce.go
package main

import (
	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/knq/firebase"
)

func main() {
	var err error

	// use the Google compute credentials with an oauth2 transport
	db, err := firebase.NewDatabaseRef(
		firebase.Transport(&oauth2.Transport{
			Source: google.ComputeTokenSource(""),
		}),
	)
	log.Fatal(err)

	// retrieve the /people keys
	keys := make(map[string]interface{})
	err = db.Ref("/people").Get(&keys, firebase.Shallow)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("keys: %+v", keys)
}
