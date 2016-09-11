// example/gce/gce.go
package main

import (
	"fmt"
	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/knq/firebase"
)

func main() {
	var err error

	// use the Google compute credentials with an oauth2 transport
	//
	// NOTE: your compute instance needs to have been created using the
	// apropriate firebase scopes for this to work!
	//
	// For example, using the gcloud tool:
	//
	// gcloud compute instances create my-test-project-1 --scopes userinfo-email,https://www.googleapis.com/auth/firebase.database
	db, err := firebase.NewDatabaseRef(
		firebase.URL("https://<PROJECT ID>.firebaseio.com/"),
		firebase.Transport(&oauth2.Transport{
			Source: google.ComputeTokenSource(""),
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// set some data
	for i := 0; i < 5; i++ {
		_, err = db.Ref("/people").Push(map[string]interface{}{
			"name": fmt.Sprintf("person %d", i),
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// retrieve the /people keys
	keys := make(map[string]interface{})
	err = db.Ref("/people").Get(&keys, firebase.Shallow)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("keys: %+v", keys)

	// delete all the keys
	for k, _ := range keys {
		err = db.Ref("/people/" + k).Remove()
		if err != nil {
			log.Fatal(err)
		}
	}
}
