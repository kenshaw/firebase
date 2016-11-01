// example/full/full.go
package main

import (
	"flag"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/knq/firebase"
)

type Person struct {
	Name      string                   `json:"name"`
	Birthdate time.Time                `json:"birth_date"`
	Created   firebase.ServerTimestamp `json:"created"`
}

var (
	flagCredentialsFile = flag.String("c", "test-1470ffbcc1d8.json", "credentials file")
)

func main() {
	var err error

	flag.Parse()

	// create initial firebase database ref using Google service account
	// credentials as downloaded from the Google cloud console
	db, err := firebase.NewDatabaseRef(
		firebase.GoogleServiceAccountCredentialsFile(*flagCredentialsFile),
		//firebase.Log(log.Printf, log.Printf), // uncomment this to see the actual HTTP requests
	)
	if err != nil {
		log.Fatal(err)
	}

	// apply security rules
	log.Printf("setting security rules")
	err = db.SetRulesJSON([]byte(securityRules))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("security rules applied successfully")

	// set up a listen context and start listener
	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startListen(db.Ref("/people"), ctxt)

	// do a short wait
	time.Sleep(5 * time.Second)

	john := &Person{
		Name:      "john doe",
		Birthdate: time.Now().Add(-18 * 365 * 24 * time.Hour),
	}

	// push john
	log.Printf("pushing john: %+v", john)
	johnID, err := db.Ref("/people").Push(john)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("created: john (%s)", johnID)

	// retrieve john
	var john1 Person
	err = db.Ref("/people/" + johnID).Get(&john1)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("retrieved john (%s): %+v", johnID, john1)

	// set john (causes all values to be overwritten)
	john.Name = "Jon Dunce"
	log.Printf("setting john (%s) to: %+v", johnID, john)
	err = db.Ref("/people/" + johnID).Set(john)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("successfully set john (%s)", johnID)

	// update a value on john
	log.Printf("adding nickname to john (%s)", johnID)
	err = db.Ref("/people/" + johnID).Update(map[string]interface{}{
		"nickname": "JD",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("successfully updated john (%s)", johnID)

	// get john again
	log.Printf("retrieving john (%s)", johnID)
	john2 := make(map[string]interface{})
	err = db.Ref("/people/" + johnID).Get(&john2)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("successfully retrieved john (%s): %+v", johnID, john2)

	//-------------------------------------------------------
	emily := Person{
		Name:      "Emily Smith",
		Birthdate: time.Now().Add(-22 * 365 * 24 * time.Hour),
	}

	// create Emily
	log.Printf("pushing emily")
	emilyID, err := db.Ref("/people/").Push(emily)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("created emily (%s)", emilyID)

	// create an authenticated ref for Emily and retrieve john as emily
	emilyDB := db.Ref("/people", firebase.UserID(emilyID))
	var johnE Person
	log.Printf("retrieving john (%s) as emily (%s)", johnID, emilyID)
	err = emilyDB.Ref(johnID).Get(&johnE)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("emily got john: %+v", johnE)

	// try to write as emily to john's data (should error)
	err = emilyDB.Ref(johnID).Update(map[string]interface{}{
		"name": "not john",
	})
	if err == nil {
		log.Fatal("emily should not be able to write to john's entry")
	}
	log.Printf("emily could not write to john as expected (got: %v)", err)

	// create authenticated "admin" ref
	adminDB := db.Ref("/people")
	adminDB.SetQueryOptions(
		firebase.AuthOverride(map[string]interface{}{
			"uid":   "<admin>",
			"admin": true,
		}),
	)

	// retrieve a shallow map (ie, the keys) using the admin ref
	log.Printf("retrieving all keys as admin")
	keys := make(map[string]interface{})
	err = adminDB.Get(&keys, firebase.Shallow)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("admin retrieved keys: %+v", keys)

	// delete keys
	for key, _ := range keys {
		log.Printf("admin removing %s", key)
		err = adminDB.Ref(key).Remove()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("admin removed %s", key)
	}

	// serialize type with time values
	now := time.Now()
	type myTimeType struct {
		ServerTimestamp firebase.ServerTimestamp `json:"sts,omitempty"`
		MyTime          firebase.Time            `json:"mtm,omitempty"`
	}

	x := myTimeType{
		MyTime: firebase.Time(now),
	}
	timeID, err := db.Ref("/time-test").Push(x)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("pushed time %s (sts: %s, mtm: %s)", timeID, x.ServerTimestamp, x.MyTime)
	var y myTimeType
	err = db.Ref("/time-test/" + timeID).Get(&y)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("retrieved time %s (sts: %s, mtm: %s)", timeID, y.ServerTimestamp, y.MyTime)

	// wait before returning to see at least one keep alive event
	log.Printf("waiting 45 seconds to see at least one keep alive event")
	time.Sleep(45 * time.Second)
}

// startListen starts a listener on the ref.
func startListen(r *firebase.Ref, ctxt context.Context) {
	eventTypes := []firebase.EventType{
		firebase.EventTypePut,
		firebase.EventTypePatch,
		firebase.EventTypeKeepAlive,
	}

	log.Printf("listening for events %v on %s", eventTypes, r.URL().String())

	evs := r.Listen(ctxt, eventTypes)
	for e := range evs {
		if e == nil {
			log.Printf("listen events channel closed")
			return
		}

		log.Printf("server event: %s", e.String())
	}
}

// securityRules provides security rules where only authenticated users can
// read the /people/*/ data, but only the owner or an administrator can write.
const securityRules = `{
  "rules": {
    ".read": "auth !== null",
    ".write": "false",
    "people": {
      "$uid": {
        ".read": "auth !== null && auth.uid !== null",
        ".write": "auth !== null && auth.uid !== null && ($uid === auth.uid || auth.admin === true)"
      }
    }
  }
}`
