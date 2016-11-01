package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/knq/firebase"
)

var (
	flagCreds = flag.String("creds", "", "google service account credentials file")
	flagRef   = flag.String("ref", "/", "firebase database path ref to merge data to")
	flagFile  = flag.String("file", "", "json encoded file")
)

func main() {
	var err error

	flag.Parse()

	// check flags
	if *flagCreds == "" || *flagFile == "" {
		log.Fatal("creds or file not specified")
	}

	// create firebase ref
	db, err := firebase.NewDatabaseRef(
		firebase.GoogleServiceAccountCredentialsFile(*flagCreds),
	)
	if err != nil {
		log.Fatal(err)
	}

	// decode json file
	buf, err := ioutil.ReadFile(*flagFile)
	if err != nil {
		log.Fatal(err)
	}

	// create json decoder
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.UseNumber()

	// unmarshal json data
	var d map[string]interface{}
	err = dec.Decode(&d)
	if err != nil {
		log.Fatal(err)
	}

	// get base ref
	r := db.Ref(*flagRef)

	// overwrite each node from data
	for k, v := range d {
		log.Printf("writing %s", k)
		err = r.Ref("/" + k).Set(v)
		if err != nil {
			log.Fatal(err)
		}
	}
}
