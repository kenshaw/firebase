package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/knq/firebase"
)

var (
	flagCredentials = flag.String("creds", "", "path to google service account credentials")
	flagRef         = flag.String("ref", "/", "firebase ref to retrieve")
	flagVerbose     = flag.Bool("v", false, "verbose logging")
)

func main() {
	var err error

	flag.Parse()

	// check credentials
	if *flagCredentials == "" {
		fmt.Fprintf(os.Stderr, "error: invalid credentials file\n")
		os.Exit(1)
	}

	// build firebase options
	opts := []firebase.Option{
		firebase.GoogleServiceAccountCredentialsFile(*flagCredentials),
	}
	if *flagVerbose {
		opts = append(opts, firebase.Log(log.Printf, log.Printf))
	}

	// create database ref
	ref, err := firebase.NewDatabaseRef(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// retrieve ref
	var v interface{}
	err = ref.Ref(*flagRef).Get(&v)
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

	fmt.Fprintf(os.Stdout, "%s\n", string(buf))
}
