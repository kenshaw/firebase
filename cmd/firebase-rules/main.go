package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/knq/firebase"
)

var (
	flagCredentials = flag.String("creds", "", "path to google service account credentials")
	flagRulesFile   = flag.String("rules", "rules.json", "path to rules file")
	flagClearRules  = flag.Bool("clear", false, "clear rules")
)

func main() {
	var err error

	flag.Parse()

	// check credentials
	if *flagCredentials == "" {
		fmt.Fprintf(os.Stderr, "error: invalid credentials file\n")
		os.Exit(1)
	}

	// load rules
	buf := []byte(emptyRules)
	if !*flagClearRules {
		buf, err = ioutil.ReadFile(*flagRulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	// decode
	var v map[string]interface{}
	err = json.Unmarshal(buf, &v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// create ref
	ref, err := firebase.NewDatabaseRef(
		firebase.GoogleServiceAccountCredentialsFile(*flagCredentials),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// set rules
	err = ref.SetRulesJSON(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// emptyRules are the empty rule set for firebase (allow no reads/writes).
const emptyRules = `{
  "rules": {
    ".read": "false",
    ".write": "false"
  }
}`
