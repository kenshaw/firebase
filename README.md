# About firebase

Package firebase provides a Firebase v3.0.0+ compatible API.

## Installation

Install in the usual way:

```sh
go get -u github.com/knq/firebase
```

## Usage

Please see [the GoDoc API page](http://godoc.org/github.com/knq/firebase) for a
full API listing.

Below is a short example showing basic usage. Additionally, a [more complete
example](example/example.go) is available.

```go
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
	)
	if err != nil {
		log.Fatal(err)
	}

	// set up a watch context
	ctxt, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start watch
	go startWatch(db.Ref("/people"), ctxt)

	// do a short wait (otherwise the watch won't start in time to see the
	// calls below)
	time.Sleep(5 * time.Second)

	john := &Person{
		Name:      "john doe",
		Birthdate: time.Now().Add(-18 * 365 * 24 * time.Hour),
	}

	// push john
	log.Printf("pushing: %+v", john)
	id, err := db.Ref("/people").Push(john)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("created: %s", id)

	// retrieve john
	var res Person
	err = db.Ref("/people/" + id).Get(&res)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("retrieved: %+v", res)

	// set john (causes all values to be overwritten)
	john.Name = "Jon Dunce"
	err = db.Ref("/people/" + id).Set(john)
	if err != nil {
		log.Fatal(err)
	}

	// update a value on john
	err = db.Ref("/people/" + id).Update(map[string]interface{}{
		"nickname": "JD",
	})
	if err != nil {
		log.Fatal(err)
	}

	// retrieve a shallow map (ie, the keys)
	keys := make(map[string]interface{})
	err = db.Ref("/people").Get(&keys, firebase.Shallow)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("keys: %+v", keys)

	// delete keys
	for key, _ := range keys {
		err = db.Ref("/people/" + key).Remove()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("deleted: %s", key)
	}

	// wait before returning to see at least one keep alive event
	time.Sleep(45 * time.Second)
}

func startWatch(r *firebase.Ref, ctxt context.Context) {
	log.Printf("starting watch on %s", r.URL().String())
	evs, err := r.Watch(ctxt)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case e := <-evs:
			if e == nil {
				log.Printf("events channel closed")
				return
			}
			log.Printf("server event: %s", e.String())
		case <-ctxt.Done():
			log.Printf("context done")
			return
		}
	}
}
```
