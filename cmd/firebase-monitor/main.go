package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/knq/firebase"
)

var (
	flagCredentials = flag.String("creds", "", "path to google service account credentials")
	flagRef         = flag.String("ref", "/", "firebase ref to monitor")
)

func main() {
	var err error

	flag.Parse()

	// check credentials
	if *flagCredentials == "" {
		fmt.Fprintf(os.Stderr, "error: invalid credentials file\n")
		os.Exit(1)
	}

	// create database ref
	ref, err := firebase.NewDatabaseRef(
		firebase.GoogleServiceAccountCredentialsFile(*flagCredentials),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// watch ref
	ch, err := ref.Ref(*flagRef).Watch(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// output events as received
	for ev := range ch {
		// unmarshal data
		var v map[string]interface{}
		err = json.Unmarshal(ev.Data, &v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// pretty format
		buf, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		log.Printf("%s: %s", strings.ToUpper(string(ev.Type)), string(buf))
	}
}
